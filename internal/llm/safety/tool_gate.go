package safety

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety/toolgate"

type ToolCallRequest = toolgate.CallRequest

type ToolGateDecision = toolgate.Decision

type ToolGate = toolgate.Gate

type DefaultToolGate = toolgate.DefaultGate

type ToolRule = toolgate.Rule

func NewDefaultToolGate() *DefaultToolGate {
	return toolgate.NewDefaultGate()
}

type ToolIntentRule = toolgate.IntentRule

type ToolPermissionRule = toolgate.PermissionRule
