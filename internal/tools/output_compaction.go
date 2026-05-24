package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/compact"

// AutoOutputCompaction returns the standard auto compaction config used by
// shell-like built-in tools.
func AutoOutputCompaction() compact.Config {
	return compact.Config{Mode: compact.ModeAuto}
}
