package sessiondb

import (
	"database/sql"
	"strings"

	appsession "github.com/TrebuchetDynamics/gormes-agent/internal/app/session"
)

func OpenDirectory() (*sql.DB, error) {
	return appsession.OpenSessionDirectoryDB()
}

func IsMemoryDatabaseNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "memory database not found")
}
