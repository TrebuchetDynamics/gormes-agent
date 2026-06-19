package gormescli

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

// dashboardMemoryQueueCap bounds the async write queue of the dashboard's
// SQLite transcript store.
const dashboardMemoryQueueCap = 1024

// dashboardChatSessionID is the default chat session id the apiserver submits
// turns under; the factory resumes this session's transcript on startup so the
// agent remembers the conversation across restarts. Keep in sync with the
// apiserver default of the same value.
const dashboardChatSessionID = "dashboard-chat"

// buildDashboardTurnLoop constructs a native kernel-backed turn loop for the
// dashboard chat surface and persists its transcript so chats survive restarts
// and show up in `gormes session list`.
//
// Transcripts are written to the shared SQLite memory store (memory.db), the
// same store the gateway uses and the session directory reads; SQLite tolerates
// concurrent processes, so this works even while the gateway runs. The chat
// session id is owned by the apiserver (it submits a stable/rotating id per
// turn), so this factory does not touch the bbolt session map — that leaves
// sessions.db free for the dashboard command to open the cron store.
//
// Any failure degrades rather than blocking dashboard startup: a missing
// provider disables chat entirely, and a missing transcript store falls back to
// an ephemeral no-op store.
func buildDashboardTurnLoop(ctx context.Context) (apiserver.TurnLoop, func(), error) {
	cfg, err := config.Load([]string{})
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}

	model := cfg.Hermes.Model
	provider := cfg.Hermes.Provider

	client, err := NewProviderHTTPClient(cfg, provider)
	if err != nil {
		return nil, nil, fmt.Errorf("provider client: %w", err)
	}

	log := slog.Default()

	// Transcript store (memory.db). Best-effort: fall back to a no-op store so
	// chat still works when the DB cannot be opened.
	var transcripts store.Store = store.NewNoop()
	var memStore *memory.SqliteStore
	if ms, err := memory.OpenSqlite(config.MemoryDBPath(), dashboardMemoryQueueCap, log); err != nil {
		log.Warn("dashboard: transcript persistence unavailable; chat will not be saved", "err", err)
	} else {
		transcripts = ms
		memStore = ms
	}

	runCtx, cancel := context.WithCancel(ctx)
	registry := BuildDefaultRegistry(runCtx, cfg, client, model)

	k := kernel.New(kernel.Config{
		Model:             model,
		Provider:          provider,
		Endpoint:          cfg.Hermes.Endpoint,
		SystemPrompt:      llm.AgentIdentityForProfile("default"),
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		Tools:             registry,
		MaxToolIterations: ConfiguredMaxToolIterations(cfg),
		MaxToolDuration:   30 * time.Second,
		ToolAudit:         audit.NewJSONLWriter(config.ToolAuditLogPath()),
		PrefillMessages:   ConfiguredPrefillMessages(cfg),
	}, client, transcripts, telemetry.New(), log)

	go k.Run(runCtx)

	// Reload prior conversation so the agent remembers across restarts: replay
	// the persisted dashboard-chat transcript into the kernel's context.
	// Best-effort — a missing/empty transcript just starts fresh.
	if history := loadDashboardChatHistory(); len(history) > 0 {
		if err := k.ResumeSession(dashboardChatSessionID, history); err != nil {
			log.Warn("dashboard: could not resume prior chat history", "err", err, "messages", len(history))
		} else {
			log.Info("dashboard: resumed prior chat history", "messages", len(history))
		}
	}

	cleanup := func() {
		cancel()
		if memStore != nil {
			_ = memStore.Close(context.Background())
		}
	}

	return apiserver.NewKernelTurnLoop(k), cleanup, nil
}

// loadDashboardChatHistory reads the persisted dashboard-chat transcript from
// memory.db and returns it as kernel messages for resume. It opens a short-
// lived read connection (the driver is registered via the memory package) and
// returns nil on any error or when there is no prior transcript.
func loadDashboardChatHistory() []llm.Message {
	db, err := sql.Open("sqlite3", config.MemoryDBPath())
	if err != nil {
		return nil
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, _ = db.Exec("PRAGMA busy_timeout=5000")
	msgs, err := transcript.LoadMessages(context.Background(), db, dashboardChatSessionID)
	if err != nil {
		return nil
	}
	out := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, llm.Message{Role: m.Role, Content: m.Content})
	}
	return out
}
