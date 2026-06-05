package goncho

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	extgoncho "github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func TestStoreObserveWritesScopedObservationAndAudit(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = store.Close(shutdownCtx)
	}()
	if err := extgoncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	svc := extgoncho.NewService(store.DB(), extgoncho.Config{
		WorkspaceID:    "gormes",
		ObserverPeerID: "gormes",
	}, nil)
	gonchoStore := NewStore(svc)
	success := true
	obs := kernel.GonchoObservation{
		Kind:       kernel.GonchoObservationUserPrompt,
		PeerID:     "telegram:42",
		SessionKey: "sess-observe",
		ContextID:  "turn-abc",
		Input:      "remember the auth decision",
		Success:    &success,
		Metadata: map[string]string{
			"platform": "telegram",
			"turn_key": "turn-abc",
		},
		Reason: "gormes hook capture",
	}

	if err := gonchoStore.Observe(ctx, obs); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if err := gonchoStore.Observe(ctx, obs); err != nil {
		t.Fatalf("Observe replay: %v", err)
	}

	list, err := extgoncho.ListObservations(ctx, store.DB(), extgoncho.ObservationQuery{
		WorkspaceID: "gormes",
		PeerID:      "telegram:42",
		SessionKey:  "sess-observe",
		ContextID:   "turn-abc",
		Kinds:       []extgoncho.ObservationKind{extgoncho.ObservationKindUserPrompt},
	})
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if list.Count != 1 {
		t.Fatalf("observation count = %d, want 1 replay-safe row", list.Count)
	}
	got := list.Observations[0]
	if got.WorkspaceID != "gormes" || got.PeerID != "telegram:42" || got.SessionKey != "sess-observe" || got.ContextID != "turn-abc" {
		t.Fatalf("observation scope = %#v, want workspace/peer/session/context populated", got)
	}
	if got.Input != "remember the auth decision" || got.Metadata["platform"] != "telegram" {
		t.Fatalf("observation payload = %#v, want input and metadata preserved", got)
	}

	audit, err := extgoncho.AuditTrail(ctx, store.DB(), extgoncho.AuditQuery{
		WorkspaceID: "gormes",
		TargetID:    got.ID,
	})
	if err != nil {
		t.Fatalf("AuditTrail: %v", err)
	}
	if audit.Count != 1 {
		t.Fatalf("audit count = %d, want 1 replay-safe audit event", audit.Count)
	}
	if audit.Events[0].Reason != "gormes hook capture" || audit.Events[0].TargetID != got.ID {
		t.Fatalf("audit event = %#v, want observe audit for observation", audit.Events[0])
	}
}
