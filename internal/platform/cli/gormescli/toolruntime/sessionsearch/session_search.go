package sessionsearch

import (
	"database/sql"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/sessionsearch"
)

type SessionSearchDirectory = sessionsearch.SessionSearchDirectory

// RegisterSessionSearchTool registers the session_search adapter without
// exposing the concrete tool package to cmd/gormes.
func RegisterSessionSearchTool(reg *tools.Registry, db *sql.DB, sessions SessionSearchDirectory) {
	reg.MustRegister(sessionsearch.NewSessionSearchTool(sessionsearch.SessionSearchToolConfig{
		DB:       db,
		Sessions: sessions,
	}))
}
