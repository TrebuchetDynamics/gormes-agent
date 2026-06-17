package gormescli

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

// dashboardMemoryQueueCap bounds the async write queue of the dashboard's
// SQLite transcript store.
const dashboardMemoryQueueCap = 1024

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

	cleanup := func() {
		cancel()
		if memStore != nil {
			_ = memStore.Close(context.Background())
		}
	}

	return apiserver.NewKernelTurnLoop(k), cleanup, nil
}
