package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem"

type WorkspaceSnapshot = filesystem.WorkspaceSnapshot

func TakeWorkspaceSnapshot(root string) (*WorkspaceSnapshot, error) {
	return filesystem.TakeWorkspaceSnapshot(root)
}
