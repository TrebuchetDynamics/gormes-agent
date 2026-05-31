package process

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/process/interrupt"

// InterruptTool requests interruption of the current agent turn.
type InterruptTool = interrupt.Tool

func NewInterruptTool(fn func()) *InterruptTool {
	return interrupt.NewTool(fn)
}
