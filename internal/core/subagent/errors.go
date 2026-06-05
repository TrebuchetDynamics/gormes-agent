// internal/core/subagent/errors.go
package subagent

import "github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/lifecycle"

var (
	// ErrMaxDepth is returned by SubagentManager.Spawn when the manager's
	// depth equals or exceeds MaxDepth.
	ErrMaxDepth = lifecycle.ErrMaxDepth

	// ErrSubagentNotFound is returned by SubagentManager.Interrupt when the
	// supplied *Subagent is not currently tracked by the manager.
	ErrSubagentNotFound = lifecycle.ErrSubagentNotFound
)
