package approval

import (
	"context"
	"errors"
	"testing"
)

func TestApprovalQueueSubmitAndHasBlocking(t *testing.T) {
	q := NewGatewayApprovalQueue()

	ticket, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{
		Command:     "git reset --hard",
		Description: "destructive git reset",
		PatternKeys: []string{
			"git reset --hard",
		},
		Evidence: map[string]string{"detector": "dangerous"},
	})
	if err != nil {
		t.Fatalf("SubmitGatewayApproval: %v", err)
	}
	if ticket.SessionKey != "session-a" || ticket.ID == 0 {
		t.Fatalf("ticket = %+v, want session-a with nonzero ID", ticket)
	}
	if !q.HasBlockingApproval("session-a") {
		t.Fatal("HasBlockingApproval(session-a) = false, want true")
	}
	if q.HasBlockingApproval("session-b") {
		t.Fatal("HasBlockingApproval(session-b) = true, want false")
	}
}

func TestApprovalQueueClonesRequestEvidence(t *testing.T) {
	q := NewGatewayApprovalQueue()
	evidence := map[string]string{"detector": "original"}
	ticket, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "danger", Evidence: evidence})
	if err != nil {
		t.Fatalf("SubmitGatewayApproval: %v", err)
	}
	evidence["detector"] = "mutated"

	if err := q.ResolveGatewayApproval(context.Background(), Resolution{SessionKey: "session-a", Choice: ChoiceOnce}); err != nil {
		t.Fatalf("ResolveGatewayApproval: %v", err)
	}
	outcome, ok := q.GatewayApprovalOutcome(ticket)
	if !ok {
		t.Fatal("approval outcome missing")
	}
	outcome.Request.Evidence["detector"] = "caller-mutated"

	stored, ok := q.GatewayApprovalOutcome(ticket)
	if !ok {
		t.Fatal("approval outcome missing on second read")
	}
	if got := stored.Request.Evidence["detector"]; got != "original" {
		t.Fatalf("stored evidence detector = %q, want original", got)
	}
}

func TestApprovalQueueResolveOldestFIFO(t *testing.T) {
	q := NewGatewayApprovalQueue()
	first, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "first"})
	if err != nil {
		t.Fatalf("Submit first: %v", err)
	}
	second, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "second"})
	if err != nil {
		t.Fatalf("Submit second: %v", err)
	}

	err = q.ResolveGatewayApproval(context.Background(), Resolution{
		SessionKey: "session-a",
		Choice:     ChoiceSession,
	})
	if err != nil {
		t.Fatalf("ResolveGatewayApproval: %v", err)
	}

	firstOutcome, ok := q.GatewayApprovalOutcome(first)
	if !ok {
		t.Fatal("first outcome missing")
	}
	if firstOutcome.Request.Command != "first" || firstOutcome.Choice != ChoiceSession || firstOutcome.Canceled {
		t.Fatalf("first outcome = %+v, want session approval for first request", firstOutcome)
	}
	if _, ok := q.GatewayApprovalOutcome(second); ok {
		t.Fatal("second request resolved too early; want FIFO single resolution")
	}
	if !q.HasBlockingApproval("session-a") {
		t.Fatal("HasBlockingApproval(session-a) = false after resolving one of two, want true")
	}
}

func TestApprovalQueueResolveAll(t *testing.T) {
	q := NewGatewayApprovalQueue()
	first, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "first"})
	if err != nil {
		t.Fatalf("Submit first: %v", err)
	}
	second, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "second"})
	if err != nil {
		t.Fatalf("Submit second: %v", err)
	}
	other, err := q.SubmitGatewayApproval("session-b", GatewayApprovalRequest{Command: "other"})
	if err != nil {
		t.Fatalf("Submit other: %v", err)
	}

	count, err := q.ResolveAllGatewayApprovals("session-a", ChoiceAlways)
	if err != nil {
		t.Fatalf("ResolveAllGatewayApprovals: %v", err)
	}
	if count != 2 {
		t.Fatalf("resolved count = %d, want 2", count)
	}
	for _, ticket := range []GatewayApprovalTicket{first, second} {
		outcome, ok := q.GatewayApprovalOutcome(ticket)
		if !ok {
			t.Fatalf("outcome for ticket %+v missing", ticket)
		}
		if outcome.Choice != ChoiceAlways || outcome.Canceled {
			t.Fatalf("outcome = %+v, want always approval", outcome)
		}
	}
	if q.HasBlockingApproval("session-a") {
		t.Fatal("session-a still blocking after resolve all")
	}
	if !q.HasBlockingApproval("session-b") {
		t.Fatal("session-b not blocking; resolve all must not affect other sessions")
	}
	if _, ok := q.GatewayApprovalOutcome(other); ok {
		t.Fatal("session-b outcome exists; resolve all touched the wrong session")
	}
}

func TestApprovalQueueClearSessionCancelsPending(t *testing.T) {
	q := NewGatewayApprovalQueue()
	first, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "first"})
	if err != nil {
		t.Fatalf("Submit first: %v", err)
	}
	second, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "second"})
	if err != nil {
		t.Fatalf("Submit second: %v", err)
	}

	count := q.ClearGatewayApprovalSession("session-a")
	if count != 2 {
		t.Fatalf("clear count = %d, want 2", count)
	}
	if q.HasBlockingApproval("session-a") {
		t.Fatal("session-a still blocking after clear")
	}
	for _, ticket := range []GatewayApprovalTicket{first, second} {
		outcome, ok := q.GatewayApprovalOutcome(ticket)
		if !ok {
			t.Fatalf("outcome for ticket %+v missing", ticket)
		}
		if outcome.Choice != ChoiceDeny || !outcome.Canceled {
			t.Fatalf("outcome = %+v, want canceled deny", outcome)
		}
	}
}

func TestApprovalQueueInvalidChoiceAndEmptySessionFailClosed(t *testing.T) {
	q := NewGatewayApprovalQueue()
	if _, err := q.SubmitGatewayApproval("", GatewayApprovalRequest{Command: "missing session"}); !errors.Is(err, ErrGatewayApprovalEmptySession) {
		t.Fatalf("SubmitGatewayApproval empty session error = %v, want ErrGatewayApprovalEmptySession", err)
	}
	if err := q.ResolveGatewayApproval(context.Background(), Resolution{SessionKey: "session-a", Choice: Choice("bad")}); !errors.Is(err, ErrGatewayApprovalInvalidChoice) {
		t.Fatalf("ResolveGatewayApproval invalid choice error = %v, want ErrGatewayApprovalInvalidChoice", err)
	}
	if _, err := q.ResolveAllGatewayApprovals("session-a", Choice("bad")); !errors.Is(err, ErrGatewayApprovalInvalidChoice) {
		t.Fatalf("ResolveAllGatewayApprovals invalid choice error = %v, want ErrGatewayApprovalInvalidChoice", err)
	}
	if err := q.ResolveGatewayApproval(context.Background(), Resolution{SessionKey: "", Choice: ChoiceOnce}); !errors.Is(err, ErrGatewayApprovalEmptySession) {
		t.Fatalf("ResolveGatewayApproval empty session error = %v, want ErrGatewayApprovalEmptySession", err)
	}
	if err := q.ResolveGatewayApproval(context.Background(), Resolution{SessionKey: "session-a", Choice: ChoiceOnce}); !errors.Is(err, ErrGatewayApprovalNotPending) {
		t.Fatalf("ResolveGatewayApproval missing pending error = %v, want ErrGatewayApprovalNotPending", err)
	}
	if count, err := q.ResolveAllGatewayApprovals("session-a", ChoiceOnce); err != nil || count != 0 {
		t.Fatalf("ResolveAllGatewayApprovals missing pending = (%d, %v), want (0, nil)", count, err)
	}
	if count := q.ClearGatewayApprovalSession(""); count != 0 {
		t.Fatalf("ClearGatewayApprovalSession empty session = %d, want 0", count)
	}
}
