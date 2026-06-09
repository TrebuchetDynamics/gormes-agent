package slashconfirm

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestQueueRegisterSupersedesAndClearScoped(t *testing.T) {
	q := NewQueue()

	first, err := q.RegisterSlashConfirmation("session-a", Request{Command: "reload-mcp"})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation first: %v", err)
	}
	second, err := q.RegisterSlashConfirmation("session-a", Request{Command: "reload-mcp", Evidence: map[string]string{"source": "second"}})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation second: %v", err)
	}
	other, err := q.RegisterSlashConfirmation("session-b", Request{Command: "reload-mcp"})
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

func TestQueueRedactsSecretLikeRequestMetadata(t *testing.T) {
	q := NewQueue()
	ticket, err := q.RegisterSlashConfirmation("session-a", Request{
		Command:     "reload-mcp --token=plain-secret-token",
		Description: "reload with api_key=description-secret",
		Evidence:    map[string]string{"token": "evidence-secret", "source": "operator password=evidence-password"},
	})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation: %v", err)
	}
	pending, ok := q.PendingSlashConfirmation("session-a")
	if !ok {
		t.Fatal("pending confirmation missing")
	}
	outcome, err := q.ResolveSlashConfirmation(context.Background(), Resolution{SessionKey: "session-a", ID: ticket.ID, Choice: ChoiceOnce})
	if err != nil {
		t.Fatalf("ResolveSlashConfirmation: %v", err)
	}
	for label, req := range map[string]Request{"pending": pending.Request, "outcome": outcome.Request} {
		combined := req.Command + "\n" + req.Description
		for key, value := range req.Evidence {
			combined += "\n" + key + "=" + value
		}
		for _, forbidden := range []string{"plain-secret-token", "description-secret", "evidence-secret", "evidence-password", "--token", "api_key", "password="} {
			if strings.Contains(combined, forbidden) {
				t.Fatalf("%s request leaked secret-like metadata %q in:\n%s", label, forbidden, combined)
			}
		}
		if !strings.Contains(combined, "[redacted]") {
			t.Fatalf("%s request missing redacted evidence in:\n%s", label, combined)
		}
	}
}

func TestQueueNormalizesRequestMetadata(t *testing.T) {
	q := NewQueue()
	ticket, err := q.RegisterSlashConfirmation("session-a", Request{
		Command:     " reload-mcp ",
		Description: " reload servers ",
		Evidence:    map[string]string{" source ": " prompt "},
	})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation: %v", err)
	}
	outcome, err := q.ResolveSlashConfirmation(context.Background(), Resolution{SessionKey: "session-a", ID: ticket.ID, Choice: ChoiceOnce})
	if err != nil {
		t.Fatalf("ResolveSlashConfirmation: %v", err)
	}
	if outcome.Request.Command != "reload-mcp" || outcome.Request.Description != "reload servers" || outcome.Request.Evidence["source"] != "prompt" {
		t.Fatalf("outcome request = %+v, want trimmed metadata", outcome.Request)
	}
	if _, ok := outcome.Request.Evidence[" source "]; ok {
		t.Fatalf("outcome evidence kept untrimmed key: %+v", outcome.Request.Evidence)
	}
}

func TestQueueClonesRequestEvidence(t *testing.T) {
	q := NewQueue()
	evidence := map[string]string{"source": "original"}
	ticket, err := q.RegisterSlashConfirmation("session-a", Request{Command: "reload-mcp", Evidence: evidence})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation: %v", err)
	}
	evidence["source"] = "mutated"

	pending, ok := q.PendingSlashConfirmation("session-a")
	if !ok {
		t.Fatal("session-a pending confirmation missing")
	}
	pending.Request.Evidence["source"] = "caller-mutated"

	outcome, err := q.ResolveSlashConfirmation(context.Background(), Resolution{SessionKey: "session-a", ID: ticket.ID, Choice: ChoiceOnce})
	if err != nil {
		t.Fatalf("ResolveSlashConfirmation: %v", err)
	}
	if got := outcome.Request.Evidence["source"]; got != "original" {
		t.Fatalf("outcome evidence source = %q, want original", got)
	}
}

func TestQueueOutcomeLookupTrimsTicketSessionKey(t *testing.T) {
	q := NewQueue()
	ticket, err := q.RegisterSlashConfirmation("session-a", Request{Command: "reload-mcp"})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation: %v", err)
	}
	if _, err := q.ResolveSlashConfirmation(context.Background(), Resolution{SessionKey: "session-a", ID: ticket.ID, Choice: ChoiceOnce}); err != nil {
		t.Fatalf("ResolveSlashConfirmation: %v", err)
	}

	lookup := Ticket{SessionKey: " session-a ", ID: ticket.ID}
	outcome, ok := q.SlashConfirmationOutcome(lookup)
	if !ok {
		t.Fatalf("SlashConfirmationOutcome(%+v) missing; want canonical session lookup", lookup)
	}
	if outcome.Ticket.SessionKey != "session-a" {
		t.Fatalf("outcome ticket session = %q, want canonical session-a", outcome.Ticket.SessionKey)
	}
}

func TestQueueResolveAllowsNilContext(t *testing.T) {
	q := NewQueue()
	ticket, err := q.RegisterSlashConfirmation("session-a", Request{Command: "reload-mcp"})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation: %v", err)
	}
	outcome, err := q.ResolveSlashConfirmation(nil, Resolution{SessionKey: "session-a", ID: ticket.ID, Choice: ChoiceOnce})
	if err != nil {
		t.Fatalf("ResolveSlashConfirmation nil context: %v", err)
	}
	if outcome.Ticket != ticket || outcome.Choice != ChoiceOnce {
		t.Fatalf("outcome = %+v, want resolved ticket %+v", outcome, ticket)
	}
}

func TestQueueResolveHonorsCanceledContextWithoutMutating(t *testing.T) {
	q := NewQueue()
	ticket, err := q.RegisterSlashConfirmation("session-a", Request{Command: "reload-mcp"})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = q.ResolveSlashConfirmation(ctx, Resolution{SessionKey: "session-a", ID: ticket.ID, Choice: ChoiceOnce})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ResolveSlashConfirmation canceled error = %v, want context.Canceled", err)
	}
	if _, ok := q.PendingSlashConfirmation("session-a"); !ok {
		t.Fatal("canceled resolve cleared pending confirmation; want no mutation")
	}
	if _, ok := q.SlashConfirmationOutcome(ticket); ok {
		t.Fatal("canceled resolve recorded outcome; want no mutation")
	}
}

func TestQueueResolveOnce(t *testing.T) {
	q := NewQueue()
	ticket, err := q.RegisterSlashConfirmation("session-a", Request{Command: "reload-mcp"})
	if err != nil {
		t.Fatalf("RegisterSlashConfirmation: %v", err)
	}

	_, err = q.ResolveSlashConfirmation(context.Background(), Resolution{
		SessionKey: "session-a",
		ID:         ticket.ID + 1,
		Choice:     ChoiceOnce,
	})
	if !errors.Is(err, ErrIDMismatch) {
		t.Fatalf("ResolveSlashConfirmation wrong ID error = %v, want ErrIDMismatch", err)
	}
	if _, ok := q.PendingSlashConfirmation("session-a"); !ok {
		t.Fatal("wrong confirm ID cleared pending confirmation; want fail-closed preservation")
	}

	outcome, err := q.ResolveSlashConfirmation(context.Background(), Resolution{
		SessionKey: "session-a",
		ID:         ticket.ID,
		Choice:     ChoiceAlways,
	})
	if err != nil {
		t.Fatalf("ResolveSlashConfirmation: %v", err)
	}
	if outcome.Ticket != ticket || outcome.Choice != ChoiceAlways || outcome.Canceled {
		t.Fatalf("outcome = %+v, want ticket=%+v choice=always canceled=false", outcome, ticket)
	}
	stored, ok := q.SlashConfirmationOutcome(ticket)
	if !ok || stored.Choice != ChoiceAlways {
		t.Fatalf("stored outcome = (%+v, %v), want always outcome", stored, ok)
	}
	if _, ok := q.PendingSlashConfirmation("session-a"); ok {
		t.Fatal("resolved confirmation remains pending")
	}

	_, err = q.ResolveSlashConfirmation(context.Background(), Resolution{
		SessionKey: "session-a",
		ID:         ticket.ID,
		Choice:     ChoiceCancel,
	})
	if !errors.Is(err, ErrNotPending) {
		t.Fatalf("ResolveSlashConfirmation second call error = %v, want ErrNotPending", err)
	}

	for _, choice := range []Choice{
		ChoiceOnce,
		ChoiceCancel,
	} {
		ticket, err := q.RegisterSlashConfirmation("session-"+string(choice), Request{Command: "reload-mcp"})
		if err != nil {
			t.Fatalf("RegisterSlashConfirmation %s: %v", choice, err)
		}
		outcome, err := q.ResolveSlashConfirmation(context.Background(), Resolution{
			SessionKey: ticket.SessionKey,
			ID:         ticket.ID,
			Choice:     choice,
		})
		if err != nil {
			t.Fatalf("ResolveSlashConfirmation %s: %v", choice, err)
		}
		if outcome.Choice != choice || outcome.Canceled != (choice == ChoiceCancel) {
			t.Fatalf("outcome for %s = %+v", choice, outcome)
		}
	}
}
