package gormescli

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

// dashboardSessionKey is the session-map key under which dashboard chat resumes
// across restarts, analogous to session.TUIKey() for the interactive TUI.
const dashboardSessionKey = "dashboard:web"

// dashboardMemoryQueueCap bounds the async write queue of the dashboard's
// SQLite transcript store.
const dashboardMemoryQueueCap = 1024

// buildDashboardTurnLoop constructs a native kernel-backed turn loop for the
// dashboard chat surface and persists its transcript so chats survive restarts
// and show up in `gormes session list`.
//
// Persistence has two parts:
//   - Transcripts are written to the shared SQLite memory store (memory.db),
//     the same store the gateway uses and the session directory reads. SQLite
//     tolerates concurrent processes, so this works even while the gateway runs.
//   - The dashboard session id is mapped in the bbolt session map (sessions.db)
//     so a later dashboard run resumes the same session id. bbolt takes an
//     exclusive file lock, so when the gateway already holds sessions.db this
//     mapping is skipped best-effort while transcript persistence continues.
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

	// Session map (sessions.db). Best-effort: skip when locked (e.g. gateway
	// running) — transcript persistence does not depend on it.
	var smap *session.BoltMap
	var initialSID string
	if sm, err := session.OpenBolt(config.SessionDBPath()); err != nil {
		log.Warn("dashboard: session map unavailable; chat session id will not be persisted", "err", err)
	} else {
		smap = sm
		if sid, err := sm.Get(ctx, dashboardSessionKey); err == nil {
			initialSID = sid
		}
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
		InitialSessionID:  initialSID,
		ToolAudit:         audit.NewJSONLWriter(config.ToolAuditLogPath()),
		PrefillMessages:   ConfiguredPrefillMessages(cfg),
	}, client, transcripts, telemetry.New(), log)

	go k.Run(runCtx)

	// Persist the live session id to sessions.db via an independent render
	// subscription (Subscribe gives its own mailbox, so it does not steal
	// frames from the turn-loop's Render consumer).
	var release func()
	if smap != nil {
		sub, rel := k.Subscribe()
		release = rel
		go func() {
			last := initialSID
			for f := range sub {
				sid := strings.TrimSpace(f.SessionID)
				if sid == "" || sid == last {
					continue
				}
				if err := smap.Put(context.Background(), dashboardSessionKey, sid); err == nil {
					last = sid
				}
			}
		}()
	}

	cleanup := func() {
		cancel()
		if release != nil {
			release()
		}
		if smap != nil {
			_ = smap.Close()
		}
		if memStore != nil {
			_ = memStore.Close(context.Background())
		}
	}

	return apiserver.NewKernelTurnLoop(k), cleanup, nil
}
