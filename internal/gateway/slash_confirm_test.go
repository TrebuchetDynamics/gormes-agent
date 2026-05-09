package gateway

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestSlashConfirmQueueRegisterSupersedesAndClearScoped(t *testing.T) {
	q := NewSlashConfirmationQueue()

	first, err := q.RegisterSlashConfirmation("session-a", SlashConfirmationRequest{Command: "reload-mcp"})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation first: %v", err)
	}
	second, err := q.RegisterSlashConfirmation("session-a", SlashConfirmationRequest{Command: "reload-mcp", Evidence: map[string]string{"source": "second"}})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation second: %v", err)
	}
	other, err := q.RegisterSlashConfirmation("session-b", SlashConfirmationRequest{Command: "reload-mcp"})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation other: %v", err)
	}

	if _, ok := q.SlashConfirmationOutcome(first); ok {
		t.Fatal("superseded confirmation recorded an outcome; want silent replacement")
	}
	pending, ok := q.PendingSlashConfirmation("session-a")
	if !ok {
		t.Fatal("session-a pending confirmation missing")
	}
	if pending.Ticket != second {
		t.Fatalf("session-a pending ticket = %+v, want second %+v", pending.Ticket, second)
	}
	if pending.Request.Evidence["source"] != "second" {
		t.Fatalf("pending evidence = %#v, want cloned second evidence", pending.Request.Evidence)
	}

	if cleared := q.ClearSlashConfirmationSession("session-a"); !cleared {
		t.Fatal("ClearSlashConfirmationSession(session-a) = false, want true")
	}
	if _, ok := q.PendingSlashConfirmation("session-a"); ok {
		t.Fatal("session-a still has pending confirmation after clear")
	}
	if _, ok := q.SlashConfirmationOutcome(second); ok {
		t.Fatal("cleared confirmation recorded an outcome; want no action/outcome")
	}
	if pending, ok := q.PendingSlashConfirmation("session-b"); !ok || pending.Ticket != other {
		t.Fatalf("session-b pending = (%+v, %v), want untouched ticket %+v", pending, ok, other)
	}
}

func TestSlashConfirmQueueResolveOnce(t *testing.T) {
	q := NewSlashConfirmationQueue()
	ticket, err := q.RegisterSlashConfirmation("session-a", SlashConfirmationRequest{Command: "reload-mcp"})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation: %v", err)
	}

	_, err = q.ResolveSlashConfirmation(context.Background(), SlashConfirmationResolution{
		SessionKey: "session-a",
		ID:         ticket.ID + 1,
		Choice:     SlashConfirmationChoiceOnce,
	})
	if !errors.Is(err, ErrSlashConfirmationIDMismatch) {
		t.Fatalf("ResolveSlashConfirmation wrong ID error = %v, want ErrSlashConfirmationIDMismatch", err)
	}
	if _, ok := q.PendingSlashConfirmation("session-a"); !ok {
		t.Fatal("wrong confirm ID cleared pending confirmation; want fail-closed preservation")
	}

	outcome, err := q.ResolveSlashConfirmation(context.Background(), SlashConfirmationResolution{
		SessionKey: "session-a",
		ID:         ticket.ID,
		Choice:     SlashConfirmationChoiceAlways,
	})
	if err != nil {
		t.Fatalf("ResolveSlashConfirmation: %v", err)
	}
	if outcome.Ticket != ticket || outcome.Choice != SlashConfirmationChoiceAlways || outcome.Canceled {
		t.Fatalf("outcome = %+v, want ticket=%+v choice=always canceled=false", outcome, ticket)
	}
	stored, ok := q.SlashConfirmationOutcome(ticket)
	if !ok || stored.Choice != SlashConfirmationChoiceAlways {
		t.Fatalf("stored outcome = (%+v, %v), want always outcome", stored, ok)
	}
	if _, ok := q.PendingSlashConfirmation("session-a"); ok {
		t.Fatal("resolved confirmation remains pending")
	}

	_, err = q.ResolveSlashConfirmation(context.Background(), SlashConfirmationResolution{
		SessionKey: "session-a",
		ID:         ticket.ID,
		Choice:     SlashConfirmationChoiceCancel,
	})
	if !errors.Is(err, ErrSlashConfirmationNotPending) {
		t.Fatalf("ResolveSlashConfirmation second call error = %v, want ErrSlashConfirmationNotPending", err)
	}

	for _, choice := range []SlashConfirmationChoice{
		SlashConfirmationChoiceOnce,
		SlashConfirmationChoiceCancel,
	} {
		ticket, err := q.RegisterSlashConfirmation("session-"+string(choice), SlashConfirmationRequest{Command: "reload-mcp"})
		if err != nil {
			t.Fatalf("RegisterSlashConfirmation %s: %v", choice, err)
		}
		outcome, err := q.ResolveSlashConfirmation(context.Background(), SlashConfirmationResolution{
			SessionKey: ticket.SessionKey,
			ID:         ticket.ID,
			Choice:     choice,
		})
		if err != nil {
			t.Fatalf("ResolveSlashConfirmation %s: %v", choice, err)
		}
		if outcome.Choice != choice || outcome.Canceled != (choice == SlashConfirmationChoiceCancel) {
			t.Fatalf("outcome for %s = %+v", choice, outcome)
		}
	}
}

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
