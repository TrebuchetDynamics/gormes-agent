package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem"

const (
	AtomicWriteFailed        = filesystem.AtomicWriteFailed
	AtomicWriteSymlinkEscape = filesystem.AtomicWriteSymlinkEscape
)

type AtomicReplaceOptions = filesystem.AtomicReplaceOptions
type AtomicReplaceResult = filesystem.AtomicReplaceResult
type AtomicReplaceError = filesystem.AtomicReplaceError

func AtomicReplace(tmpPath, targetPath string, opts AtomicReplaceOptions) (AtomicReplaceResult, error) {
	return filesystem.AtomicReplace(tmpPath, targetPath, opts)
}

func AtomicWrite(targetPath string, data []byte) error {
	return filesystem.AtomicWrite(targetPath, data)
}
