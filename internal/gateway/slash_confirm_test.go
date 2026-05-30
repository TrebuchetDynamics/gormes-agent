package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestManager_ResetClearsSlashConfirmationForTargetSessionOnly(t *testing.T) {
	for _, tc := range []struct {
		name string
		ev   InboundEvent
	}{
		{
			name: "event reset",
			ev:   InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventReset},
		},
		{
			name: "slash new",
			ev:   InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventSubmit, Text: "/new"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ch := newFakeChannel("telegram")
			fk := &fakeKernel{}
			q := NewSlashConfirmationQueue()
			if _, err := q.RegisterSlashConfirmation("telegram:42", SlashConfirmationRequest{Command: "reload-mcp"}); err != nil {
				t.Fatalf("Register target: %v", err)
			}
			other, err := q.RegisterSlashConfirmation("telegram:99", SlashConfirmationRequest{Command: "reload-mcp"})
			if err != nil {
				t.Fatalf("Register other: %v", err)
			}
			m := NewManagerWithSubmitter(ManagerConfig{
				AllowedChats:       map[string]string{"telegram": "42"},
				SlashConfirmations: q,
			}, fk, slog.Default())
			if err := m.Register(ch); err != nil {
				t.Fatalf("Register: %v", err)
			}

			if err := m.handleInbound(context.Background(), tc.ev); err != nil {
				t.Fatalf("handleInbound: %v", err)
			}
			if _, ok := q.PendingSlashConfirmation("telegram:42"); ok {
				t.Fatal("target session confirmation still pending after successful reset")
			}
			if pending, ok := q.PendingSlashConfirmation("telegram:99"); !ok || pending.Ticket != other {
				t.Fatalf("other session pending = (%+v, %v), want ticket %+v", pending, ok, other)
			}
			sent := ch.sentSnapshot()
			if len(sent) != 1 || !strings.Contains(sent[0].Text, "Session reset") {
				t.Fatalf("sent = %+v, want reset confirmation", sent)
			}
		})
	}
}

func TestManager_ResetDuringActiveTurnPreservesSlashConfirmation(t *testing.T) {
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{resetErr: kernel.ErrResetDuringTurn}
	q := NewSlashConfirmationQueue()
	ticket, err := q.RegisterSlashConfirmation("telegram:42", SlashConfirmationRequest{Command: "reload-mcp"})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:       map[string]string{"telegram": "42"},
		SlashConfirmations: q,
	}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(context.Background(), InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		Kind:     EventReset,
	}); err != nil {
		t.Fatalf("handleInbound: %v", err)
	}
	pending, ok := q.PendingSlashConfirmation("telegram:42")
	if !ok || pending.Ticket != ticket {
		t.Fatalf("pending after reset refusal = (%+v, %v), want ticket %+v", pending, ok, ticket)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "Cannot reset during active turn") {
		t.Fatalf("sent = %+v, want active-turn refusal", sent)
	}
}
