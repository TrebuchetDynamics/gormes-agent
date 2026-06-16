package react

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety/plangate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety/toolgate"
)

func TestSafetyAdapter_CheckPlanRefuse(t *testing.T) {
	adapter := newDefaultTestAdapter()

	plan := plangate.PlannedActions{
		Calls: []plangate.PlannedCall{
			{ToolName: "evil", TrustClassRequired: []string{"not-allowed"}},
		},
	}

	decision, err := adapter.CheckPlan(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != "refuse" {
		t.Fatalf("action = %s, want refuse", decision.Action)
	}
	if decision.Refusal == nil {
		t.Fatal("refusal is nil")
	}
	if !decision.ContinueTurn {
		t.Fatal("turn should continue after refusal")
	}
}

func TestSafetyAdapter_CheckPlanProceed(t *testing.T) {
	adapter := newDefaultTestAdapter()

	plan := plangate.PlannedActions{
		Calls: []plangate.PlannedCall{
			{ToolName: "echo", TrustClassRequired: []string{"operator"}},
		},
	}

	decision, _ := adapter.CheckPlan(context.Background(), plan)
	if decision.Action != "proceed" {
		t.Fatalf("action = %s, want proceed", decision.Action)
	}
}

func TestSafetyAdapter_CheckToolRefuse(t *testing.T) {
	adapter := newDefaultTestAdapter()

	call := toolgate.CallRequest{
		ToolName:   "gateway_restart",
		Arguments:  json.RawMessage(`{}`),
		CallerRole: "operator",
	}

	decision, _ := adapter.CheckTool(context.Background(), call)
	if decision.Action != "refuse" {
		t.Fatalf("action = %s, want refuse", decision.Action)
	}
}

func TestSafetyAdapter_CheckToolExecute(t *testing.T) {
	adapter := newDefaultTestAdapter()

	call := toolgate.CallRequest{
		ToolName:   "echo",
		Arguments:  json.RawMessage(`{}`),
		CallerRole: "operator",
	}

	decision, _ := adapter.CheckTool(context.Background(), call)
	if decision.Action != "execute" {
		t.Fatalf("action = %s, want execute", decision.Action)
	}
}

func TestRefusalEvidence(t *testing.T) {
	r := &RefuseAction{ToolName: "dangerous", Reason: "blocked by policy"}
	raw, err := r.MarshalEvidence()
	if err != nil {
		t.Fatal(err)
	}
	var ev RefusalEvidence
	json.Unmarshal(raw, &ev)
	if ev.ToolName != "dangerous" || ev.Action != "refused" {
		t.Fatal("refusal evidence mismatch")
	}
}
