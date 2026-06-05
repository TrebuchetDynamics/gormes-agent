package filesystem

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem/atomicio"

const (
	AtomicWriteFailed        = atomicio.AtomicWriteFailed
	AtomicWriteSymlinkEscape = atomicio.AtomicWriteSymlinkEscape
)

type AtomicReplaceOptions = atomicio.AtomicReplaceOptions
type AtomicReplaceResult = atomicio.AtomicReplaceResult
type AtomicReplaceError = atomicio.AtomicReplaceError

func AtomicReplace(tmpPath, targetPath string, opts AtomicReplaceOptions) (AtomicReplaceResult, error) {
	return atomicio.AtomicReplace(tmpPath, targetPath, opts)
}

func AtomicWrite(targetPath string, data []byte) error {
	return atomicio.AtomicWrite(targetPath, data)
}
