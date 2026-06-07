package safety

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety/react"

type RefuseAction = react.RefuseAction

type ReActDecision = react.Decision

type ReActSafetyAdapter = react.SafetyAdapter

func NewReActSafetyAdapter(planGate PlanGate, toolGate ToolGate) *ReActSafetyAdapter {
	return react.NewSafetyAdapter(planGate, toolGate)
}

type RefusalEvidence = react.RefusalEvidence
