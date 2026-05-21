package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestGonchoAdapterObserveWritesScopedObservationAndAudit(t *testing.T) {
	ctx := context.Background()
	db, err := sqlOpenGoncho(filepath.Join(t.TempDir(), "memory.db"))
	if err != nil {
		t.Fatalf("sqlOpenGoncho: %v", err)
	}
	defer db.Close()

	svc := goncho.NewService(db, goncho.Config{
		WorkspaceID:    "gormes",
		ObserverPeerID: "gormes",
	}, nil)
	store := newGonchoAdapter(svc)
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

	if err := store.Observe(ctx, obs); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if err := store.Observe(ctx, obs); err != nil {
		t.Fatalf("Observe replay: %v", err)
	}

	list, err := goncho.ListObservations(ctx, db, goncho.ObservationQuery{
		WorkspaceID: "gormes",
		PeerID:      "telegram:42",
		SessionKey:  "sess-observe",
		ContextID:   "turn-abc",
		Kinds:       []goncho.ObservationKind{goncho.ObservationKindUserPrompt},
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

	audit, err := goncho.AuditTrail(ctx, db, goncho.AuditQuery{
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
