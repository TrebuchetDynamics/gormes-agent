package gateway

import (
	"context"
	"testing"
)

func TestGatewayApprovalQueueCompatibilityWrapper(t *testing.T) {
	queue := NewGatewayApprovalQueue()
	ticket, err := queue.SubmitGatewayApproval("session-a", GatewayApprovalRequest{Command: "danger"})
	if err != nil {
		t.Fatalf("SubmitGatewayApproval: %v", err)
	}
	if err := queue.ResolveGatewayApproval(context.Background(), ApprovalResolution{SessionKey: "session-a", Choice: ApprovalChoiceOnce}); err != nil {
		t.Fatalf("ResolveGatewayApproval: %v", err)
	}
	outcome, ok := queue.GatewayApprovalOutcome(ticket)
	if !ok || outcome.Choice != ApprovalChoiceOnce || outcome.Request.Command != "danger" {
		t.Fatalf("GatewayApprovalOutcome = %+v, %v; want once danger outcome", outcome, ok)
	}
}
