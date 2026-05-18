package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli/modules/channels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
	telegram "github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/cron"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gonchotools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

// newTelegramCommand returns a fresh telegram bot adapter command.
// The adapter previously shipped as the standalone cmd/gormes-telegram
// binary (Phase 2.B.1 through 3.A). Unified into cmd/gormes under the
// < 100 MB binary ceiling. Constructor pattern keeps each
// newRootCommand() instance independent.
func newTelegramCommand() *cobra.Command {
	return channelsmodule.NewTelegramCommandWithSeams(channelsmodule.TelegramCommandSeams{
		Run: runTelegram,
	})
}

func runTelegram(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if cfg.Telegram.BotToken == "" {
		return fmt.Errorf("no Telegram bot token — set GORMES_TELEGRAM_BOT_TOKEN, GORMES_TELEGRAM_TOKEN, TELEGRAM_BOT_TOKEN, or [telegram].bot_token in config.toml")
	}
	if cfg.Telegram.AllowedChatID == 0 && len(cfg.Telegram.AllowedUserIDs) == 0 && !cfg.Telegram.FirstRunDiscovery {
		return fmt.Errorf("no chat/user allowlist and discovery disabled — set [telegram].allowed_chat_id, TELEGRAM_ALLOWED_USERS, or [telegram].first_run_discovery = true")
	}
	if os.Getenv("GORMES_TELEGRAM_BOT_TOKEN") == "" && os.Getenv("GORMES_TELEGRAM_TOKEN") == "" && os.Getenv("TELEGRAM_BOT_TOKEN") == "" && os.Getenv("TELEGRAM_TOKEN") == "" {
		slog.Warn("bot_token read from config.toml; prefer GORMES_TELEGRAM_BOT_TOKEN or TELEGRAM_BOT_TOKEN env var for secrets")
	}
	if _, err := ensureGatewayAgentTemplates(cfg, slog.Default()); err != nil {
		return err
	}

	// Phase 2.C — open the session map before the kernel so we can prime it.
	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return fmt.Errorf("session map: %w", err)
	}
	defer smap.Close()
	sessionMirror := startSessionIndexMirror(smap, slog.Default())
	defer sessionMirror.Stop()

	ctx := context.Background()
	var key string
	if cfg.Telegram.AllowedChatID != 0 {
		key = session.TelegramKey(cfg.Telegram.AllowedChatID)
		if cfg.Resume != "" {
			if err := smap.Put(ctx, key, cfg.Resume); err != nil {
				slog.Warn("failed to apply --resume override", "err", err)
			}
		}
	}
	var initialSID string
	if key != "" {
		if sid, err := smap.Get(ctx, key); err != nil {
			slog.Warn("could not load initial session_id", "key", key, "err", err)
		} else {
			initialSID = sid
			if sid != "" {
				slog.Info("resuming persisted session", "key", key, "session_id", sid)
			}
		}
	}

	// Phase 3.A — open the SQLite memory store; worker starts immediately.
	mstore, err := memory.OpenSqlite(config.MemoryDBPath(), cfg.Telegram.MemoryQueueCap, slog.Default())
	if err != nil {
		return fmt.Errorf("memory store: %w", err)
	}
	defer func() {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), kernel.ShutdownBudget)
		defer cancelShutdown()
		if err := mstore.Close(shutdownCtx); err != nil {
			slog.Warn("memory store close", "err", err)
		}
	}()

	// Phase 3.D.5 — start the Memory Mirror for operator auditability.
	mstore.StartMirror(memory.MirrorConfig{
		Enabled:  cfg.Telegram.MirrorEnabled,
		Path:     cfg.Telegram.MirrorPath,
		Interval: cfg.Telegram.MirrorInterval,
		Logger:   slog.Default(),
	})

	hc := hermes.NewHTTPClient(cfg.Hermes.Endpoint, cfg.Hermes.APIKey)

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	reg := buildDefaultRegistry(rootCtx, cfg, hc, cfg.Hermes.Model)
	gonchoCfg := cfg.Goncho.RuntimeConfig()
	gonchoCfg.SessionDirectory = smap
	gonchotools.RegisterHonchoTools(reg, newGonchoRuntimeService(mstore.DB(), gonchoCfg, slog.Default(), hc, cfg.Hermes.Model))

	tm := telemetry.New()
	toolAudit := audit.NewJSONLWriter(config.ToolAuditLogPath())

	var recallProv kernel.RecallProvider

	recallActive := cfg.Telegram.RecallEnabled && cfg.Telegram.AllowedChatID != 0

	// Phase 3.D — semantic fusion wiring. Activated only when recall is
	// active AND the feature flag is set AND an embedding model is named.
	// Falls back to Hermes.Endpoint when SemanticEndpoint is empty
	// (Ollama often hosts both /v1/chat/completions and /v1/embeddings).
	var semCache *memory.SemanticCache
	var ec *memory.EmbedClient
	if recallActive && cfg.Telegram.SemanticEnabled && cfg.Telegram.SemanticModel != "" {
		endpoint := cfg.Telegram.SemanticEndpoint
		if endpoint == "" {
			endpoint = cfg.Hermes.Endpoint
		}
		ec = memory.NewEmbedClient(endpoint, cfg.Hermes.APIKey)
		semCache = memory.NewSemanticCache()
	}

	if recallActive {
		memProv := memory.NewRecall(mstore, memory.RecallConfig{
			WeightThreshold:       cfg.Telegram.RecallWeightThreshold,
			MaxFacts:              cfg.Telegram.RecallMaxFacts,
			Depth:                 cfg.Telegram.RecallDepth,
			DecayHorizonDays:      cfg.Telegram.RecallDecayHorizonDays,
			SemanticModel:         cfg.Telegram.SemanticModel,
			SemanticTopK:          cfg.Telegram.SemanticTopK,
			SemanticMinSimilarity: cfg.Telegram.SemanticMinSimilarity,
			QueryEmbedTimeout:     cfg.Telegram.QueryEmbedTimeout,
		}, slog.Default())
		if ec != nil {
			memProv = memProv.WithEmbedClient(ec, semCache)
		}
		recallProv = &recallAdapter{p: memProv}
	}

	k := kernel.New(kernel.Config{
		Model:             cfg.Hermes.Model,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		Tools:             reg,
		MaxToolIterations: configuredMaxToolIterations(cfg),
		MaxToolDuration:   30 * time.Second,
		InitialSessionID:  initialSID,
		Recall:            recallProv,
		ChatKey:           key,
		ToolAudit:         toolAudit,
	}, hc, mstore, tm, slog.Default())

	// Phase 3.B — async LLM-assisted entity/relationship extractor.
	// Polls turns WHERE extracted=0 on a background goroutine; completely
	// decoupled from the kernel's hot path.
	ext := memory.NewExtractor(mstore, hc, memory.ExtractorConfig{
		Model:        cfg.Hermes.Model,
		BatchSize:    cfg.Telegram.ExtractorBatchSize,
		PollInterval: cfg.Telegram.ExtractorPollInterval,
	}, slog.Default())
	defer func() {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), kernel.ShutdownBudget)
		defer cancelShutdown()
		if err := ext.Close(shutdownCtx); err != nil {
			slog.Warn("extractor close", "err", err)
		}
	}()

	tc, err := telegram.NewRealClient(cfg.Telegram.BotToken)
	if err != nil {
		return err
	}

	bot := telegram.New(telegram.Config{
		AllowedChatID:     cfg.Telegram.AllowedChatID,
		AllowedChatIDs:    cfg.Telegram.AllowedChatIDs(),
		AllowedUserIDs:    cfg.Telegram.AllowedUserIDs,
		FirstRunDiscovery: cfg.Telegram.FirstRunDiscovery,
		RequireMention:    cfg.Telegram.RequireMention,
		GuestMode:         cfg.Telegram.GuestMode,
		BotUsername:       cfg.Telegram.BotUsername,
		Notifications:     cfg.Telegram.Notifications,
		AudioTranscriber:  resolveTelegramAudioTranscriber(),
		DynamicCommands:   gatewayTelegramDynamicCommands(rootCtx, cfg),
		TokenLockDir:      config.GatewayLockDir(),
	}, tc, slog.Default())
	go ext.Run(rootCtx)

	// Phase 3.D — Embedder worker bounded to rootCtx. No-op when ec is nil.
	if ec != nil {
		embedder := memory.NewEmbedder(mstore, ec, memory.EmbedderConfig{
			Model:        cfg.Telegram.SemanticModel,
			PollInterval: cfg.Telegram.EmbedderPollInterval,
			BatchSize:    cfg.Telegram.EmbedderBatchSize,
			CallTimeout:  cfg.Telegram.EmbedderCallTimeout,
		}, slog.Default(), semCache)
		go embedder.Run(rootCtx)
		defer func() {
			shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), kernel.ShutdownBudget)
			defer cancelShutdown()
			if err := embedder.Close(shutdownCtx); err != nil {
				slog.Warn("embedder close", "err", err)
			}
		}()
	}

	// Phase 2.D — cron scheduler + executor + mirror (opt-in via
	// cfg.Cron.Enabled). No-op when disabled — zero goroutines, zero
	// bbolt bucket, zero RAM.
	if cfg.Cron.Enabled && cfg.Telegram.AllowedChatID != 0 {
		// Reuse the existing session.db for the cron_jobs bucket.
		cronStore, err := cron.NewStore(smap.DB())
		if err != nil {
			return fmt.Errorf("cron: init store: %w", err)
		}
		cronRunStore := cron.NewRunStore(mstore.DB())

		sink := newTelegramDeliverySink(bot, cfg.Telegram.AllowedChatID)

		cronExec := cron.NewExecutor(cron.ExecutorConfig{
			Kernel:           k,
			JobStore:         cronStore,
			RunStore:         cronRunStore,
			Sink:             sink,
			CallTimeout:      cfg.Cron.CallTimeout,
			CronApprovalMode: cfg.Approvals.CronMode,
		}, slog.Default())

		cronSched := cron.NewScheduler(cron.SchedulerConfig{
			Store:    cronStore,
			Executor: cronExec,
		}, slog.Default())

		if err := cronSched.Start(rootCtx); err != nil {
			return fmt.Errorf("cron: start scheduler: %w", err)
		}
		defer func() {
			shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), kernel.ShutdownBudget)
			defer cancelShutdown()
			cronSched.Stop(shutdownCtx)
		}()

		cronMirror := cron.NewMirror(cron.MirrorConfig{
			JobStore: cronStore,
			RunStore: cronRunStore,
			Path:     cfg.CronMirrorPath(),
			Interval: cfg.Cron.MirrorInterval,
		}, slog.Default())
		go cronMirror.Run(rootCtx)
	}

	go func() {
		<-rootCtx.Done()
		time.AfterFunc(kernel.ShutdownBudget, func() {
			slog.Error("shutdown budget exceeded; forcing exit")
			os.Exit(3)
		})
	}()

	slog.Info("gormes telegram starting",
		"endpoint", cfg.Hermes.Endpoint,
		"allowed_chat_id", cfg.Telegram.AllowedChatID,
		"discovery", cfg.Telegram.FirstRunDiscovery,
		"sessions_db", config.SessionDBPath(),
		"memory_db", config.MemoryDBPath(),
		"extractor_batch_size", cfg.Telegram.ExtractorBatchSize,
		"extractor_poll_interval", cfg.Telegram.ExtractorPollInterval,
		"semantic_enabled", cfg.Telegram.SemanticEnabled,
		"semantic_model", cfg.Telegram.SemanticModel)

	mgr := gateway.NewManager(telegramManagerConfig(cfg, smap), k, slog.Default())
	if err := mgr.Register(bot); err != nil {
		return fmt.Errorf("register telegram: %w", err)
	}

	go k.Run(rootCtx)
	return mgr.Run(rootCtx)
}

// recallAdapter bridges *memory.Provider (which uses memory.RecallInput)
// to kernel.RecallProvider (which uses kernel.RecallParams). Same
// fields, distinct types — the adapter preserves package dependency
// isolation.
type recallAdapter struct {
	p *memory.Provider
}

func (a *recallAdapter) GetContext(ctx context.Context, params kernel.RecallParams) string {
	return a.p.GetContext(ctx, memory.RecallInput{
		UserMessage: params.UserMessage,
		ChatKey:     params.ChatKey,
		SessionID:   params.SessionID,
	})
}

// telegramBotSender is the narrow interface newTelegramDeliverySink
// needs — matches what *telegram.Bot exposes.
type telegramBotSender interface {
	SendToChat(ctx context.Context, chatID int64, text string) error
}

// newTelegramDeliverySink wraps the running Telegram bot as a
// cron.DeliverySink. Every cron-fired output is sent to the
// operator's configured AllowedChatID.
func newTelegramDeliverySink(bot telegramBotSender, chatID int64) cron.DeliverySink {
	return cron.FuncSink(func(ctx context.Context, text string) error {
		return bot.SendToChat(ctx, chatID, text)
	})
}

func telegramManagerConfig(cfg config.Config, smap session.Map) gateway.ManagerConfig {
	allowedChats := map[string]string{}
	if cfg.Telegram.AllowedChatID != 0 {
		allowedChats["telegram"] = strconv.FormatInt(cfg.Telegram.AllowedChatID, 10)
	}
	allowDiscovery := map[string]bool{
		"telegram": cfg.Telegram.FirstRunDiscovery,
	}
	allowedWhitelists := map[string]gateway.WhitelistConfig{}
	if wl := gateway.ParseWhitelistConfig(cfg.Telegram.AllowedChatIDs()); wl.Enabled {
		allowedWhitelists["telegram"] = wl
	}
	return gatewayManagerConfig(cfg, allowedChats, allowDiscovery, allowedWhitelists, smap, nil, nil, nil, gateway.RestartConfig{})
}
