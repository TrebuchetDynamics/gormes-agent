package filesystem

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem/snapshot"

type WorkspaceSnapshot = snapshot.WorkspaceSnapshot

func TakeWorkspaceSnapshot(root string) (*WorkspaceSnapshot, error) {
	return snapshot.TakeWorkspaceSnapshot(root)
}
