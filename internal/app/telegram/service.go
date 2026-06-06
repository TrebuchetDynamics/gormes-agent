// Package telegram provides the core Telegram bot runtime behavior extracted
// from cmd/gormes/telegram.go. The RunTelegram function orchestrates session
// setup, memory store, kernel, cron, and the Telegram bot loop — the same
// contract as the root-level runTelegram, but injected dependencies for the
// two root-package functions (gatewayTelegramDynamicCommands and
// gatewayManagerConfig) that remain in cmd/gormes for now.
package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	telegram "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram"
	internalgoncho "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/app/audiotools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/channelmemory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

// RunOptions contains the dependencies injected from the root main package
// that cannot be imported by this internal package.
type RunOptions struct {
	// GatewayTelegramDynamicCommands returns the dynamic Telegram commands
	// from skills. Injected from gateway_channels.go.
	GatewayTelegramDynamicCommands func(ctx context.Context, cfg config.Config) []gateway.PlatformCommand

	// GatewayManagerConfig builds a ManagerConfig from config and policy maps.
	// Injected from main.go.
	GatewayManagerConfig func(cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, allowedWhitelists map[string]gateway.WhitelistConfig, smap session.Map) gateway.ManagerConfig

	// EnsureAgentTemplates ensures the agent template directory exists and
	// is populated. Injected from gatewaymodule (CLI module) to avoid an
	// app→CLI import.
	EnsureAgentTemplates func(cfg config.Config, log *slog.Logger) error

	// NewExitCodeError wraps an error with an exit code.
	NewExitCodeError func(code int, err error) error
}

// RunTelegram is the core Telegram bot runtime entry point. It was extracted
// from cmd/gormes/telegram.go's runTelegram function without behavior changes.
func RunTelegram(cmd *cobra.Command, _ []string, opts RunOptions) error {
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
	if err := opts.EnsureAgentTemplates(cfg, slog.Default()); err != nil {
		return err
	}

	// Phase 2.C — open the session map before the kernel so we can prime it.
	smap, boltMap, sessionNotice, err := OpenTelegramSessionMap()
	if err != nil {
		return fmt.Errorf("session map: %w", err)
	}
	defer smap.Close()
	if sessionNotice != "" {
		slog.Warn(sessionNotice, "sessions_db", config.SessionDBPath(), "action", "answering Telegram with in-memory session state; stop the other gormes owner or use `gormes gateway status` / `gormes gateway stop` to restore persistence")
	}
	if sessionMirror := gormescli.StartSessionIndexMirror(boltMap, slog.Default()); sessionMirror != nil {
		defer sessionMirror.Stop()
	}

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
	memorySettings := channelmemory.SettingsFromConfig(cfg)
	mstore, err := memory.OpenSqlite(config.MemoryDBPath(), memorySettings.QueueCap, slog.Default())
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
		Enabled:  memorySettings.MirrorEnabled,
		Path:     memorySettings.MirrorPath,
		Interval: memorySettings.MirrorInterval,
		Logger:   slog.Default(),
	})

	hc := llm.NewHTTPClient(cfg.Hermes.Endpoint, cfg.Hermes.APIKey)

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	reg := gormescli.BuildDefaultRegistry(rootCtx, cfg, hc, cfg.Hermes.Model)
	gonchoCfg := cfg.Goncho.RuntimeConfig()
	if boltMap != nil {
		gonchoCfg.SessionDirectory = &internalgoncho.SessionDirectoryAdapter{Map: boltMap}
	} else {
		slog.Warn("goncho cross-session metadata disabled while sessions.db is locked", "sessions_db", config.SessionDBPath())
	}
	svc := gormescli.NewChannelGonchoService(mstore.DB(), gonchoCfg, slog.Default(), hc, cfg.Hermes.Model)
	gormescli.RegisterChannelGonchoTools(reg, svc)

	tm := telemetry.New()
	toolAudit := audit.NewJSONLWriter(config.ToolAuditLogPath())

	legacyRecallActive := memorySettings.LegacyRecallActive(cfg.Telegram.AllowedChatID != 0)

	// Phase 3.D — semantic fusion wiring for the legacy recall fallback.
	var semCache *memory.SemanticCache
	var ec *memory.EmbedClient
	if memorySettings.SemanticFusionActive(legacyRecallActive) {
		endpoint := memorySettings.SemanticEndpoint
		if endpoint == "" {
			endpoint = cfg.Hermes.Endpoint
		}
		ec = memory.NewEmbedClient(endpoint, cfg.Hermes.APIKey)
		semCache = memory.NewSemanticCache()
	}

	gonchoStore, recallProv := channelmemory.Providers(channelmemory.Options{
		GonchoEnabled:       cfg.Goncho.Enabled,
		GonchoService:       svc,
		PeerID:              key,
		LegacyRecallEnabled: legacyRecallActive,
		LegacyRecall: func() kernel.RecallProvider {
			memProv := memory.NewRecall(mstore, memorySettings.Recall, slog.Default())
			if ec != nil {
				memProv = memProv.WithEmbedClient(ec, semCache)
			}
			return &recallAdapter{p: memProv}
		},
	})

	k := kernel.New(kernel.Config{
		Model:             cfg.Hermes.Model,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		Tools:             reg,
		MaxToolIterations: gormescli.ConfiguredMaxToolIterations(cfg),
		MaxToolDuration:   30 * time.Second,
		InitialSessionID:  initialSID,
		Recall:            recallProv,
		ChatKey:           key,
		Goncho:            gonchoStore,
		ToolAudit:         toolAudit,
		PrefillMessages:   gormescli.ConfiguredPrefillMessages(cfg),
	}, hc, mstore, tm, slog.Default())

	// Phase 3.B — async LLM-assisted entity/relationship extractor.
	ext := memory.NewExtractor(mstore, hc, memory.ExtractorConfig{
		Model:        cfg.Hermes.Model,
		BatchSize:    memorySettings.ExtractorBatchSize,
		PollInterval: memorySettings.ExtractorPollInterval,
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

	dynamicCmds := opts.GatewayTelegramDynamicCommands(rootCtx, cfg)
	bot := telegram.New(telegram.Config{
		AllowedChatID:     cfg.Telegram.AllowedChatID,
		AllowedChatIDs:    cfg.Telegram.AllowedChatIDs(),
		AllowedUserIDs:    cfg.Telegram.AllowedUserIDs,
		FirstRunDiscovery: cfg.Telegram.FirstRunDiscovery,
		RequireMention:    cfg.Telegram.RequireMention,
		GuestMode:         cfg.Telegram.GuestMode,
		BotUsername:       cfg.Telegram.BotUsername,
		Notifications:     cfg.Telegram.Notifications,
		AudioTranscriber:  audiotools.ResolveTelegramAudioTranscriber(),
		DynamicCommands:   dynamicCmds,
		TokenLockDir:      config.GatewayLockDir(),
	}, tc, slog.Default())
	go ext.Run(rootCtx)

	// Phase 3.D — Embedder worker bounded to rootCtx. No-op when ec is nil.
	if ec != nil {
		embedder := memory.NewEmbedder(mstore, ec, memory.EmbedderConfig{
			Model:        memorySettings.SemanticModel,
			PollInterval: memorySettings.EmbedderPollInterval,
			BatchSize:    memorySettings.EmbedderBatchSize,
			CallTimeout:  memorySettings.EmbedderCallTimeout,
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

	// Phase 2.D — cron scheduler + executor + mirror (opt-in via cfg.Cron.Enabled).
	if cfg.Cron.Enabled && cfg.Telegram.AllowedChatID != 0 {
		if boltMap == nil {
			slog.Warn("cron disabled while sessions.db is locked", "sessions_db", config.SessionDBPath())
		} else {
			cronStore, err := cron.NewStore(boltMap.DB())
			if err != nil {
				return fmt.Errorf("cron: init store: %w", err)
			}
			cronRunStore := cron.NewRunStore(mstore.DB())

			sink := NewTelegramDeliverySink(bot, cfg.Telegram.AllowedChatID)

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
		"extractor_batch_size", memorySettings.ExtractorBatchSize,
		"extractor_poll_interval", memorySettings.ExtractorPollInterval,
		"semantic_enabled", memorySettings.SemanticEnabled,
		"semantic_model", memorySettings.SemanticModel)

	mgr := gateway.NewManager(TelegramManagerConfig(cfg, smap, opts.GatewayManagerConfig), k, slog.Default())
	if err := mgr.Register(bot); err != nil {
		return fmt.Errorf("register telegram: %w", err)
	}

	go k.Run(rootCtx)
	return mgr.Run(rootCtx)
}

// recallAdapter bridges *memory.Provider to kernel.RecallProvider.
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

// TelegramBotSender is the narrow interface for cron delivery sink.
type TelegramBotSender interface {
	SendToChat(ctx context.Context, chatID int64, text string) error
}

// NewTelegramDeliverySink wraps the running Telegram bot as a cron.DeliverySink.
func NewTelegramDeliverySink(bot TelegramBotSender, chatID int64) cron.DeliverySink {
	return cron.FuncSink(func(ctx context.Context, text string) error {
		return bot.SendToChat(ctx, chatID, text)
	})
}

// OpenTelegramSessionMap opens the BoltDB session map, falling back to an
// in-memory map when the database is locked.
func OpenTelegramSessionMap() (session.Map, *session.BoltMap, string, error) {
	path := config.SessionDBPath()
	smap, err := session.OpenBolt(path)
	if err == nil {
		return smap, smap, "", nil
	}
	if errors.Is(err, session.ErrDBLocked) {
		return session.NewMemMap(), nil, "telegram session state: in-memory (sessions.db locked)", nil
	}
	return nil, nil, "", err
}

// TelegramManagerConfig builds the telegram-specific gateway ManagerConfig by
// calling the injected GatewayManagerConfig from the root package.
func TelegramManagerConfig(cfg config.Config, smap session.Map, gatewayMgrCfg func(config.Config, map[string]string, map[string]bool, map[string]gateway.WhitelistConfig, session.Map) gateway.ManagerConfig) gateway.ManagerConfig {
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
	return gatewayMgrCfg(cfg, allowedChats, allowDiscovery, allowedWhitelists, smap)
}