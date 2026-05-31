package filesystem

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem/state"

const (
	FileStateStatusStale       = state.FileStateStatusStale
	FileStateStatusDeleted     = state.FileStateStatusDeleted
	FileStateStatusMissing     = state.FileStateStatusMissing
	FileStateStatusCWDMismatch = state.FileStateStatusCWDMismatch
)

// FileStateRegistry tracks the file snapshot a task last read or wrote.
type FileStateRegistry = state.FileStateRegistry

// FileStateSnapshot is the redacted state evidence returned by file tools.
type FileStateSnapshot = state.FileStateSnapshot

// FileStateCheck describes a failed freshness check before a file mutation.
type FileStateCheck = state.FileStateCheck

// NewFileStateRegistry returns an empty file-state registry.
func NewFileStateRegistry() *FileStateRegistry {
	return state.NewFileStateRegistry()
}

func FileStatePayload(snapshot FileStateSnapshot) map[string]any {
	return state.FileStatePayload(snapshot)
}

func FileStateErrorPayload(path string, check *FileStateCheck) map[string]any {
	return state.FileStateErrorPayload(path, check)
}

func NormalizeFileTaskID(taskID string) string {
	return state.NormalizeFileTaskID(taskID)
}
