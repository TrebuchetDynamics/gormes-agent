package kanban

import (
	"path/filepath"
	"strings"
)

func kanbanWorkspaceRootForDBPath(dbPath string) string {
	return filepath.Join(kanbanBoardRootForDBPath(dbPath), "workspaces")
}

func kanbanWorkerLogRootForDBPath(dbPath string) string {
	return filepath.Join(kanbanBoardRootForDBPath(dbPath), "logs")
}

func kanbanBoardRootForDBPath(dbPath string) string {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return filepath.Join(".", "kanban")
	}

	clean := filepath.Clean(dbPath)
	boardDir := filepath.Dir(clean)
	if filepath.Base(clean) == "kanban.db" &&
		filepath.Base(filepath.Dir(boardDir)) == "boards" &&
		ValidateBoardSlug(filepath.Base(boardDir)) == nil {
		return boardDir
	}
	return filepath.Join(filepath.Dir(clean), "kanban")
}
