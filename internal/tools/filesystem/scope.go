package filesystem

import filescope "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem/scope"

const ProfileWorkspaceScopeViolation = filescope.ProfileWorkspaceScopeViolation

const (
	ProfileWorkspaceAccessRead     = filescope.ProfileWorkspaceAccessRead
	ProfileWorkspaceAccessWrite    = filescope.ProfileWorkspaceAccessWrite
	ProfileWorkspaceAccessExecute  = filescope.ProfileWorkspaceAccessExecute
	ProfileWorkspaceAccessDelegate = filescope.ProfileWorkspaceAccessDelegate
)

type FilesystemScope = filescope.FilesystemScope
type PathCheckResult = filescope.PathCheckResult
type ProfileWorkspaceAccess = filescope.ProfileWorkspaceAccess
type ProfileWorkspaceScopeOptions = filescope.ProfileWorkspaceScopeOptions
type ProfileWorkspaceScope = filescope.ProfileWorkspaceScope

func NewFilesystemScope(cwd string, readPaths, writePaths []string) *FilesystemScope {
	return filescope.NewFilesystemScope(cwd, readPaths, writePaths)
}

func EvalPathOrExistingAncestor(path string) string {
	return filescope.EvalPathOrExistingAncestor(path)
}

func ValidateWorkspaceRealPath(root, abs string) error {
	return filescope.ValidateWorkspaceRealPath(root, abs)
}

func WorkspaceRel(root, path string) string {
	return filescope.WorkspaceRel(root, path)
}

func NewProfileWorkspaceScope(opts ProfileWorkspaceScopeOptions) (*ProfileWorkspaceScope, error) {
	return filescope.NewProfileWorkspaceScope(opts)
}

func NewFailClosedProfileWorkspaceScope(err error) *ProfileWorkspaceScope {
	return filescope.NewFailClosedProfileWorkspaceScope(err)
}

func NormalizeWorkspacePath(path, base string) (string, error) {
	return filescope.NormalizeWorkspacePath(path, base)
}
