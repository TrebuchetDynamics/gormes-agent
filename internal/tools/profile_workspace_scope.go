package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem"

const ProfileWorkspaceScopeViolation = filesystem.ProfileWorkspaceScopeViolation

type ProfileWorkspaceAccess = filesystem.ProfileWorkspaceAccess

const (
	ProfileWorkspaceAccessRead     = filesystem.ProfileWorkspaceAccessRead
	ProfileWorkspaceAccessWrite    = filesystem.ProfileWorkspaceAccessWrite
	ProfileWorkspaceAccessExecute  = filesystem.ProfileWorkspaceAccessExecute
	ProfileWorkspaceAccessDelegate = filesystem.ProfileWorkspaceAccessDelegate
)

type ProfileWorkspaceScopeOptions = filesystem.ProfileWorkspaceScopeOptions
type ProfileWorkspaceScope = filesystem.ProfileWorkspaceScope

func NewProfileWorkspaceScope(opts ProfileWorkspaceScopeOptions) (*ProfileWorkspaceScope, error) {
	return filesystem.NewProfileWorkspaceScope(opts)
}

func NewFailClosedProfileWorkspaceScope(err error) *ProfileWorkspaceScope {
	return filesystem.NewFailClosedProfileWorkspaceScope(err)
}

func normalizeWorkspacePath(path, base string) (string, error) {
	return filesystem.NormalizeWorkspacePath(path, base)
}
