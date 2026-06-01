package toolruntime

import (
	"database/sql"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/toolruntime/delegation"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/toolruntime/kanban"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/toolruntime/sessionsearch"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type DelegationToolOptions = delegation.DelegationToolOptions
type SessionSearchDirectory = sessionsearch.SessionSearchDirectory

// RegisterDelegationTool registers the delegate tool when delegation is enabled.
func RegisterDelegationTool(opts DelegationToolOptions) {
	delegation.RegisterDelegationTool(opts)
}

// RegisterKanbanTools registers the gated Kanban toolset without exposing the
// concrete tool adapter package to cmd/gormes.
func RegisterKanbanTools(reg *tools.Registry) {
	kanban.RegisterKanbanTools(reg)
}

// RegisterSessionSearchTool registers the session_search adapter without
// exposing the concrete tool package to cmd/gormes.
func RegisterSessionSearchTool(reg *tools.Registry, db *sql.DB, sessions SessionSearchDirectory) {
	sessionsearch.RegisterSessionSearchTool(reg, db, sessions)
}
