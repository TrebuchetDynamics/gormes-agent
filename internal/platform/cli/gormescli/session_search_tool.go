package gormescli

import (
	"database/sql"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/toolruntime"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type SessionSearchDirectory = toolruntime.SessionSearchDirectory

// RegisterSessionSearchTool registers the session_search adapter without
// exposing the concrete tool package to cmd/gormes.
func RegisterSessionSearchTool(reg *tools.Registry, db *sql.DB, sessions SessionSearchDirectory) {
	toolruntime.RegisterSessionSearchTool(reg, db, sessions)
}
