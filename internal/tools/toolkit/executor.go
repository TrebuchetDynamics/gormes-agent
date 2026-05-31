package toolkit

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit/execution"

// ToolExecutor executes tools on behalf of an agent.
type ToolExecutor = execution.ToolExecutor

// ToolRequest is a single tool invocation request submitted to a ToolExecutor.
type ToolRequest = execution.ToolRequest

// ToolEvent is one observation from a tool invocation.
type ToolEvent = execution.ToolEvent

// InProcessToolExecutor runs tools directly against a Registry in the current
// process, honoring each tool's declared timeout.
type InProcessToolExecutor = execution.InProcessToolExecutor

// NewInProcessToolExecutor returns an in-process executor backed by reg.
func NewInProcessToolExecutor(reg *Registry) *InProcessToolExecutor {
	return execution.NewInProcessToolExecutor(reg)
}

var _ ToolExecutor = (*InProcessToolExecutor)(nil)
