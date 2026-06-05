package skills

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/runtime"
)

type Snapshot = runtime.Snapshot

type InvalidSkill = runtime.InvalidSkill

type Store = runtime.Store

type Runtime = runtime.Runtime

func NewStore(root string, maxBytes int) *Store { return runtime.NewStore(root, maxBytes) }

func NewRuntime(root string, maxBytes, selectionCap int, usageLogPath string) *Runtime {
	return runtime.NewRuntime(root, maxBytes, selectionCap, usageLogPath)
}
