package react

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety/plangate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety/toolgate"
)

func TestSafetyLoop_FullTurnSafe(t *testing.T) {
	adapter := newDefaultTestAdapter()

	plan := plangate.PlannedActions{
		Calls: []plangate.PlannedCall{
			{ToolName: "echo", TrustClassRequired: []string{"operator"}},
		},
	}
	planDecision, _ := adapter.CheckPlan(context.Background(), plan)
	if planDecision.Action != "proceed" {
		t.Fatal("safe plan should proceed")
	}

	toolDecision, _ := adapter.CheckTool(context.Background(), toolgate.CallRequest{
		ToolName: "echo", Arguments: json.RawMessage(`{}`), CallerRole: "operator",
	})
	if toolDecision.Action != "execute" {
		t.Fatal("safe tool should execute")
	}
}

func TestSafetyLoop_UnsafePlanCaught(t *testing.T) {
	adapter := newDefaultTestAdapter()

	plan := plangate.PlannedActions{
		Calls: []plangate.PlannedCall{
			{ToolName: "dangerous", TrustClassRequired: []string{"not-allowed"}},
		},
	}
	decision, _ := adapter.CheckPlan(context.Background(), plan)
	if decision.Action != "refuse" {
		t.Fatal("unsafe plan should be refused before any tool executes")
	}
	if decision.Refusal == nil {
		t.Fatal("refusal missing reason")
	}
}

func TestSafetyLoop_ToolDriftCaught(t *testing.T) {
	adapter := newDefaultTestAdapter()

	plan := plangate.PlannedActions{
		Calls: []plangate.PlannedCall{
			{ToolName: "echo", TrustClassRequired: []string{"operator"}},
			{ToolName: "gateway_restart", TrustClassRequired: []string{"operator"}},
		},
	}
	planDecision, _ := adapter.CheckPlan(context.Background(), plan)
	if planDecision.Action != "proceed" {
		t.Fatal("plan should pass (plan gate doesn't check system-only)")
	}

	toolDecision, _ := adapter.CheckTool(context.Background(), toolgate.CallRequest{
		ToolName: "gateway_restart", Arguments: json.RawMessage(`{}`), CallerRole: "operator",
	})
	if toolDecision.Action != "refuse" {
		t.Fatal("system-only tool should be caught by tool gate even if plan gate passed")
	}
}

func TestSafetyLoop_Recovery(t *testing.T) {
	adapter := newDefaultTestAdapter()

	evilDecision, _ := adapter.CheckTool(context.Background(), toolgate.CallRequest{
		ToolName: "gateway_restart", Arguments: json.RawMessage(`{}`), CallerRole: "operator",
	})
	if evilDecision.Action != "refuse" {
		t.Fatal("evil tool should be refused")
	}
	if !evilDecision.ContinueTurn {
		t.Fatal("turn should continue after refusal for recovery")
	}

	safeDecision, _ := adapter.CheckTool(context.Background(), toolgate.CallRequest{
		ToolName: "echo", Arguments: json.RawMessage(`{"text":"ok"}`), CallerRole: "operator",
	})
	if safeDecision.Action != "execute" {
		t.Fatal("safe tool should execute after previous refusal")
	}
}
