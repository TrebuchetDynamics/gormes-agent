package llm

import (
	"context"
	"testing"
)

func TestPlanGate_AllowSafePlan(t *testing.T) {
	gate := NewDefaultPlanGate()

	plan := PlannedActions{
		Calls: []PlannedCall{
			{ToolName: "echo", TrustClassRequired: []string{"operator"}},
			{ToolName: "now", TrustClassRequired: []string{"operator"}},
		},
	}

	decision, err := gate.CheckPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("safe plan was refused: %s", decision.Reason)
	}
	if len(decision.BlockedCalls) != 0 {
		t.Fatalf("blocked calls in safe plan: %v", decision.BlockedCalls)
	}
}

func TestPlanGate_RefuseUnsafePlan(t *testing.T) {
	gate := NewDefaultPlanGate()

	plan := PlannedActions{
		Calls: []PlannedCall{
			{ToolName: "dangerous_tool", TrustClassRequired: []string{"not-a-real-class"}},
		},
	}

	decision, err := gate.CheckPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Allowed {
		t.Fatal("unsafe plan was allowed")
	}
	if len(decision.BlockedCalls) != 1 {
		t.Fatalf("blocked calls = %d, want 1", len(decision.BlockedCalls))
	}
	if decision.BlockedCalls[0] != "dangerous_tool" {
		t.Fatalf("blocked call = %q, want dangerous_tool", decision.BlockedCalls[0])
	}
}

func TestPlanGate_RefuseSingleTool(t *testing.T) {
	gate := NewDefaultPlanGate()

	plan := PlannedActions{
		Calls: []PlannedCall{
			{ToolName: "safe_tool", TrustClassRequired: []string{"operator"}},
			{ToolName: "bad_tool", TrustClassRequired: []string{"not-allowed"}},
			{ToolName: "also_safe", TrustClassRequired: []string{"operator"}},
		},
	}

	decision, err := gate.CheckPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Allowed {
		t.Fatal("plan with one unsafe tool was allowed")
	}
	if len(decision.BlockedCalls) != 1 {
		t.Fatalf("blocked calls = %d, want 1", len(decision.BlockedCalls))
	}
	if decision.BlockedCalls[0] != "bad_tool" {
		t.Fatalf("blocked call = %q, want bad_tool", decision.BlockedCalls[0])
	}
}

func TestPlanGate_EmptyPlan(t *testing.T) {
	gate := NewDefaultPlanGate()

	decision, err := gate.CheckPlan(context.Background(), PlannedActions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !decision.Allowed {
		t.Fatal("empty plan was refused")
	}
}

func TestPlanGate_CustomRule(t *testing.T) {
	gate := NewDefaultPlanGate()
	gate.AddRule(blockToolRule{"rm_tool"})

	plan := PlannedActions{
		Calls: []PlannedCall{
			{ToolName: "rm_tool", TrustClassRequired: []string{"operator"}},
		},
	}

	decision, err := gate.CheckPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Allowed {
		t.Fatal("plan with blocked tool was allowed by custom rule")
	}
}

func TestPlanGate_MultipleRules(t *testing.T) {
	gate := NewDefaultPlanGate()
	gate.AddRule(blockToolRule{"tool_a"})

	plan := PlannedActions{
		Calls: []PlannedCall{
			{ToolName: "tool_a", TrustClassRequired: []string{"operator"}},
		},
	}

	decision, err := gate.CheckPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Allowed {
		t.Fatal("plan was allowed when custom rule should have blocked it")
	}
}

func TestPlanGate_ReasonMessage(t *testing.T) {
	gate := NewDefaultPlanGate()

	plan := PlannedActions{
		Calls: []PlannedCall{
			{ToolName: "evil_tool", TrustClassRequired: []string{"nonexistent"}},
		},
	}

	decision, _ := gate.CheckPlan(context.Background(), plan)
	if decision.Reason == "" {
		t.Fatal("blocked plan has empty reason message")
	}
}

type blockToolRule struct {
	toolName string
}

func (r blockToolRule) Check(ctx context.Context, call PlannedCall) (bool, string) {
	if call.ToolName == r.toolName {
		return false, "blocked by test rule"
	}
	return true, ""
}
