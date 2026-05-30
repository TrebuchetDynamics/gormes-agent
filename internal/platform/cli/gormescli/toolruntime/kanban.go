package toolruntime

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	kanbantools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/kanban"
)

// RegisterKanbanTools registers the gated Kanban toolset without exposing the
// concrete tool adapter package to cmd/gormes.
func RegisterKanbanTools(reg *tools.Registry) {
	for _, tool := range kanbantools.NewTools(kanbantools.ConfigFromEnv()) {
		reg.MustRegister(tool)
	}
}
