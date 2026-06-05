package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/process"

type InterruptTool = process.InterruptTool

func NewInterruptTool(fn func()) *InterruptTool {
	return process.NewInterruptTool(fn)
}
