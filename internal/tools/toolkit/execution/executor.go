package execution

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit/core"
)

// ToolExecutor executes tools on behalf of an agent.
type ToolExecutor interface {
	Execute(ctx context.Context, req ToolRequest) (<-chan ToolEvent, error)
}

// ToolRequest is a single tool invocation request submitted to a ToolExecutor.
type ToolRequest struct {
	AgentID  string
	ToolName string
	Input    json.RawMessage
	Metadata map[string]string
}

// ToolEvent is one observation from a tool invocation.
type ToolEvent struct {
	Type   string
	Output json.RawMessage
	Err    error
}

// InProcessToolExecutor runs tools directly against a Registry in the current
// process, honoring each tool's declared timeout.
type InProcessToolExecutor struct {
	registry *core.Registry
}

// NewInProcessToolExecutor returns an in-process executor backed by reg.
func NewInProcessToolExecutor(reg *core.Registry) *InProcessToolExecutor {
	return &InProcessToolExecutor{registry: reg}
}

// Execute looks up the requested tool and streams started→output→completed, or
// started→failed when the tool returns an error.
func (e *InProcessToolExecutor) Execute(ctx context.Context, req ToolRequest) (<-chan ToolEvent, error) {
	if e == nil || e.registry == nil {
		return nil, fmt.Errorf("tools: nil tool registry")
	}
	tool, ok := e.registry.Get(req.ToolName)
	if !ok {
		return nil, fmt.Errorf("%w: %s", core.ErrUnknownTool, req.ToolName)
	}

	ch := make(chan ToolEvent, 4)
	go func() {
		defer close(ch)

		ch <- ToolEvent{Type: "started"}

		execCtx := ctx
		if timeout := tool.Timeout(); timeout > 0 {
			var cancel context.CancelFunc
			execCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		out, err := tool.Execute(execCtx, req.Input)
		if err != nil {
			ch <- ToolEvent{Type: "failed", Err: err}
			return
		}

		ch <- ToolEvent{Type: "output", Output: out}
		ch <- ToolEvent{Type: "completed"}
	}()

	return ch, nil
}
