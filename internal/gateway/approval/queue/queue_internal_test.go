package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/approval/choice"
)

func TestResolveGatewayApprovalRejectsOutOfOrderTicketID(t *testing.T) {
	q := NewGatewayApprovalQueue()
	first, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "first"})
	if err != nil {
		t.Fatalf("SubmitGatewayApproval first: %v", err)
	}
	second, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "second"})
	if err != nil {
		t.Fatalf("SubmitGatewayApproval second: %v", err)
	}

	err = q.ResolveGatewayApproval(context.Background(), choice.Resolution{SessionKey: "session-a", TicketID: second.ID, Choice: choice.ChoiceOnce})
	if !errors.Is(err, ErrGatewayApprovalNotPending) {
		t.Fatalf("ResolveGatewayApproval out-of-order ticket err = %v, want ErrGatewayApprovalNotPending", err)
	}
	if _, ok := q.GatewayApprovalOutcome(first); ok {
		t.Fatal("out-of-order resolve recorded first approval outcome")
	}
	if _, ok := q.GatewayApprovalOutcome(second); ok {
		t.Fatal("out-of-order resolve recorded second approval outcome")
	}
	if !q.HasBlockingApproval("session-a") {
		t.Fatal("out-of-order resolve removed pending approvals")
	}
}

func TestResolveGatewayApprovalHonorsCanceledContextAfterInitialCheck(t *testing.T) {
	q := NewGatewayApprovalQueue()
	ticket, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "danger"})
	if err != nil {
		t.Fatalf("SubmitGatewayApproval: %v", err)
	}

	ctx := &cancelAfterFirstErrContext{}
	if err := q.ResolveGatewayApproval(ctx, choice.Resolution{SessionKey: "session-a", Choice: choice.ChoiceOnce}); !errors.Is(err, context.Canceled) {
		t.Fatalf("ResolveGatewayApproval error = %v, want context.Canceled", err)
	}
	if _, ok := q.GatewayApprovalOutcome(ticket); ok {
		t.Fatal("ResolveGatewayApproval recorded outcome after context cancellation")
	}
	if !q.HasBlockingApproval("session-a") {
		t.Fatal("ResolveGatewayApproval removed pending approval after context cancellation")
	}
}

type cancelAfterFirstErrContext struct{ calls int }

func (c *cancelAfterFirstErrContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *cancelAfterFirstErrContext) Done() <-chan struct{}       { return nil }
func (c *cancelAfterFirstErrContext) Value(any) any               { return nil }
func (c *cancelAfterFirstErrContext) Err() error {
	c.calls++
	if c.calls == 1 {
		return nil
	}
	return context.Canceled
}

func TestGatewayApprovalQueuePrunesOldestStoredOutcomes(t *testing.T) {
	q := NewGatewayApprovalQueue()
	var first GatewayApprovalTicket
	for i := 0; i < maxStoredGatewayApprovalOutcomes+1; i++ {
		ticket, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "danger"})
		if err != nil {
			t.Fatalf("SubmitGatewayApproval %d: %v", i, err)
		}
		if i == 0 {
			first = ticket
		}
		if err := q.ResolveGatewayApproval(context.Background(), choice.Resolution{SessionKey: "session-a", Choice: choice.ChoiceOnce}); err != nil {
			t.Fatalf("ResolveGatewayApproval %d: %v", i, err)
		}
	}

	if _, ok := q.GatewayApprovalOutcome(first); ok {
		t.Fatalf("oldest outcome was retained after exceeding maxStoredGatewayApprovalOutcomes=%d", maxStoredGatewayApprovalOutcomes)
	}
	latest := GatewayApprovalTicket{SessionKey: "session-a", ID: uint64(maxStoredGatewayApprovalOutcomes + 1)}
	if _, ok := q.GatewayApprovalOutcome(latest); !ok {
		t.Fatal("latest outcome was pruned; want newest outcomes retained")
	}
}

func TestSubmitGatewayApprovalDoesNotReuseStoredOutcomeIDAfterCounterWrap(t *testing.T) {
	q := NewGatewayApprovalQueue()
	first, err := q.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "danger"})
	if err != nil {
		t.Fatalf("SubmitGatewayApproval first: %v", err)
	}
	if err := q.ResolveGatewayApproval(context.Background(), choice.Resolution{SessionKey: "session-a", Choice: choice.ChoiceOnce}); err != nil {
		t.Fatalf("ResolveGatewayApproval first: %v", err)
	}
	q.nextID = ^uint64(0)

	second, err := q.SubmitGatewayApproval("session-b", GatewayApprovalRequest{Command: "danger"})
	if err != nil {
		t.Fatalf("SubmitGatewayApproval after wrap: %v", err)
	}
	if second.ID == 0 {
		t.Fatalf("SubmitGatewayApproval returned zero ticket ID after counter wrap: %+v", second)
	}
	if second.ID == first.ID {
		t.Fatalf("SubmitGatewayApproval reused stored outcome ID after wrap: first=%+v second=%+v", first, second)
	}
	if err := q.ResolveGatewayApproval(context.Background(), choice.Resolution{SessionKey: "session-b", Choice: choice.ChoiceOnce}); err != nil {
		t.Fatalf("ResolveGatewayApproval second: %v", err)
	}
	if _, ok := q.GatewayApprovalOutcome(first); !ok {
		t.Fatal("stored first outcome was lost after wrapped ticket resolution")
	}
}
