package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety"

type RefuseAction = safety.RefuseAction
type ReActDecision = safety.ReActDecision
type ReActSafetyAdapter = safety.ReActSafetyAdapter
type RefusalEvidence = safety.RefusalEvidence

func NewReActSafetyAdapter(planGate PlanGate, toolGate ToolGate) *ReActSafetyAdapter {
	return safety.NewReActSafetyAdapter(planGate, toolGate)
}
