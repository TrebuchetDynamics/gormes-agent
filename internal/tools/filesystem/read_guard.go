package filesystem

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem/readguard"

const (
	FileReadDedupStatusUnchanged     = readguard.FileReadDedupStatusUnchanged
	FileReadStatusDedupCacheDisabled = readguard.FileReadStatusDedupCacheDisabled
	FileReadStatusGuardUnavailable   = readguard.FileReadStatusGuardUnavailable
	FileReadStatusDedupStubBlocked   = readguard.FileReadStatusDedupStubBlocked
	FileReadDedupStatusMessage       = readguard.FileReadDedupStatusMessage
	FileReadDedupStubBlockedMessage  = readguard.FileReadDedupStubBlockedMessage
)

var ErrFileReadGuardStatusContent = readguard.ErrFileReadGuardStatusContent

type FileReadGuardOptions = readguard.FileReadGuardOptions
type FileReadGuard = readguard.FileReadGuard
type FileReadResult = readguard.FileReadResult
type FileReadEvidence = readguard.FileReadEvidence

func NewFileReadGuard(opts FileReadGuardOptions) *FileReadGuard {
	return readguard.NewFileReadGuard(opts)
}

func IsFileReadGuardStatusText(content []byte) bool {
	return readguard.IsFileReadGuardStatusText(content)
}
