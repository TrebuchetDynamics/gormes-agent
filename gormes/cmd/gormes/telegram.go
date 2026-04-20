package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

// telegramCmd runs Gormes as a Telegram bot — the adapter previously
// shipped as the standalone cmd/gormes-telegram binary (Phase 2.B.1
// through 3.A). Unified into cmd/gormes under the < 100 MB binary
// ceiling; see the unification commit's message for rationale.
var telegramCmd = &cobra.Command{
	Use:          "telegram",
	Short:        "Run Gormes as a Telegram bot adapter",
	Long:         "Long-polls Telegram for DMs from the allowlisted chat, drives the same kernel + tool loop as the TUI, and persists turns to the SQLite memory store.",
	SilenceUsage: true,
	RunE:         runTelegram,
}

func runTelegram(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if cfg.Telegram.BotToken == "" {
		return fmt.Errorf("no Telegram bot token — set GORMES_TELEGRAM_TOKEN env or [telegram].bot_token in config.toml")
	}
	if cfg.Telegram.AllowedChatID == 0 && !cfg.Telegram.FirstRunDiscovery {
		return fmt.Errorf("no chat allowlist and discovery disabled — set one of [telegram].allowed_chat_id or [telegram].first_run_discovery = true")
	}
	if os.Getenv("GORMES_TELEGRAM_TOKEN") == "" {
		slog.Warn("bot_token read from config.toml; prefer GORMES_TELEGRAM_TOKEN env var for secrets")
	}

	// Phase 2.C — open the session map before the kernel so we can prime it.
	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return fmt.Errorf("session map: %w", err)
	}
	defer smap.Close()

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

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})
	reg.MustRegister(&tools.NowTool{})
	reg.MustRegister(&tools.RandIntTool{})

	tm := telemetry.New()

	var recallProv kernel.RecallProvider

	// Phase 3.D — semantic fusion wiring. Activated only when both the
	// feature flag is true AND an embedding model is named. Defaults to
	// Hermes.Endpoint if SemanticEndpoint is empty (Ollama often hosts
	// both /v1/chat/completions and /v1/embeddings on the same port).
	var semCache *memory.SemanticCache
	var ec *memory.EmbedClient
	if cfg.Telegram.RecallEnabled && cfg.Telegram.AllowedChatID != 0 &&
		cfg.Telegram.SemanticEnabled && cfg.Telegram.SemanticModel != "" {
		endpoint := cfg.Telegram.SemanticEndpoint
		if endpoint == "" {
			endpoint = cfg.Hermes.Endpoint
		}
		ec = memory.NewEmbedClient(endpoint, cfg.Hermes.APIKey)
		semCache = memory.NewSemanticCache()
	}

	if cfg.Telegram.RecallEnabled && cfg.Telegram.AllowedChatID != 0 {
		memProv := memory.NewRecall(mstore, memory.RecallConfig{
			WeightThreshold:       cfg.Telegram.RecallWeightThreshold,
			MaxFacts:              cfg.Telegram.RecallMaxFacts,
			Depth:                 cfg.Telegram.RecallDepth,
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
		MaxToolIterations: 10,
		MaxToolDuration:   30 * time.Second,
		InitialSessionID:  initialSID,
		Recall:            recallProv,
		ChatKey:           key,
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
		CoalesceMs:        cfg.Telegram.CoalesceMs,
		FirstRunDiscovery: cfg.Telegram.FirstRunDiscovery,
		SessionMap:        smap,
		SessionKey:        key,
	}, tc, k, slog.Default())

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go k.Run(rootCtx)
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
			shutdownCtx, cancelSd := context.WithTimeout(context.Background(), kernel.ShutdownBudget)
			defer cancelSd()
			if err := embedder.Close(shutdownCtx); err != nil {
				slog.Warn("embedder close", "err", err)
			}
		}()
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
	return bot.Run(rootCtx)
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
