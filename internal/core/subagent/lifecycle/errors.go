// internal/core/subagent/errors.go
package lifecycle

import "github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/lifecycle/sentinel"

var (
	// ErrMaxDepth is returned by SubagentManager.Spawn when the manager's
	// depth equals or exceeds MaxDepth.
	ErrMaxDepth = sentinel.ErrMaxDepth

	// ErrSubagentNotFound is returned by SubagentManager.Interrupt when the
	// supplied *Subagent is not currently tracked by the manager.
	ErrSubagentNotFound = sentinel.ErrSubagentNotFound
)
