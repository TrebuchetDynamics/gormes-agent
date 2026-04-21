// gormes/internal/subagent/errors.go
package subagent

import "errors"

var (
	// ErrMaxDepth is returned by SubagentManager.Spawn when the manager's
	// depth equals or exceeds MaxDepth.
	ErrMaxDepth = errors.New("subagent: max depth reached")

	// ErrSubagentNotFound is returned by SubagentManager.Interrupt when the
	// supplied *Subagent is not currently tracked by the manager.
	ErrSubagentNotFound = errors.New("subagent: not found")
)
