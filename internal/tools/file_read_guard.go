package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem"

const (
	FileReadDedupStatusUnchanged     = filesystem.FileReadDedupStatusUnchanged
	FileReadStatusDedupCacheDisabled = filesystem.FileReadStatusDedupCacheDisabled
	FileReadStatusGuardUnavailable   = filesystem.FileReadStatusGuardUnavailable
	FileReadStatusDedupStubBlocked   = filesystem.FileReadStatusDedupStubBlocked
)

const fileReadDedupStatusMessage = filesystem.FileReadDedupStatusMessage
const fileReadDedupStubBlockedMessage = filesystem.FileReadDedupStubBlockedMessage

var ErrFileReadGuardStatusContent = filesystem.ErrFileReadGuardStatusContent

type FileReadGuardOptions = filesystem.FileReadGuardOptions
type FileReadGuard = filesystem.FileReadGuard
type FileReadResult = filesystem.FileReadResult
type FileReadEvidence = filesystem.FileReadEvidence

func NewFileReadGuard(opts FileReadGuardOptions) *FileReadGuard {
	return filesystem.NewFileReadGuard(opts)
}

func isFileReadGuardStatusText(content []byte) bool {
	return filesystem.IsFileReadGuardStatusText(content)
}
