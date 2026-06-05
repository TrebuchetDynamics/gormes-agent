package gormescli

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/toolruntime"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// RegisterKanbanTools registers the gated Kanban toolset without exposing the
// concrete tool adapter package to cmd/gormes.
func RegisterKanbanTools(reg *tools.Registry) {
	toolruntime.RegisterKanbanTools(reg)
}
