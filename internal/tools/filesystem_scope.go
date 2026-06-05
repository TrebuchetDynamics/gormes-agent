package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem"

type FilesystemScope = filesystem.FilesystemScope
type PathCheckResult = filesystem.PathCheckResult

func NewFilesystemScope(cwd string, readPaths, writePaths []string) *FilesystemScope {
	return filesystem.NewFilesystemScope(cwd, readPaths, writePaths)
}

func evalPathOrExistingAncestor(path string) string {
	return filesystem.EvalPathOrExistingAncestor(path)
}
