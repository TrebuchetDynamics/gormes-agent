package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety"

type ToolCallRequest = safety.ToolCallRequest
type ToolGateDecision = safety.ToolGateDecision
type ToolGate = safety.ToolGate
type DefaultToolGate = safety.DefaultToolGate
type ToolRule = safety.ToolRule
type ToolIntentRule = safety.ToolIntentRule
type ToolPermissionRule = safety.ToolPermissionRule

func NewDefaultToolGate() *DefaultToolGate {
	return safety.NewDefaultToolGate()
}
