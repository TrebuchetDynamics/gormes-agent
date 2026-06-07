// internal/core/subagent/blocked.go
package lifecycle

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/lifecycle/limits"
	"github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/lifecycle/toolaccess"
)

const (
	// MaxDepth bounds the subagent depth tree. Parent depth=0; a Spawn at
	// depth >= MaxDepth returns ErrMaxDepth. Default policy: parent → child OK,
	// grandchild rejected.
	MaxDepth = limits.MaxDepth

	// DefaultMaxConcurrent is SpawnBatch's default semaphore size when the
	// caller passes maxConcurrent <= 0.
	DefaultMaxConcurrent = limits.DefaultMaxConcurrent

	// DefaultMaxIterations is the per-subagent iteration budget applied at
	// Spawn time when SubagentConfig.MaxIterations <= 0. The StubRunner
	// ignores this; LLMRunner (2.E.7) will honour it.
	DefaultMaxIterations = limits.DefaultMaxIterations
)

// BlockedTools is the forward-looking list of tool names that subagents
// must not be allowed to invoke. Of these names, only delegate_task exists
// in the current Gormes tool surface; the others are placeholders for
// tools that will be added in later phases.
var BlockedTools = toolaccess.BlockedTools

func BlockedToolRequest(enabled []string) string {
	return toolaccess.BlockedToolRequest(enabled)
}

func ToolAllowlisted(enabled []string, name string) bool {
	return toolaccess.ToolAllowlisted(enabled, name)
}
