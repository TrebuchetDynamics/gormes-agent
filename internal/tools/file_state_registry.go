package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem"

const (
	fileStateStatusStale       = filesystem.FileStateStatusStale
	fileStateStatusDeleted     = filesystem.FileStateStatusDeleted
	fileStateStatusMissing     = filesystem.FileStateStatusMissing
	fileStateStatusCWDMismatch = filesystem.FileStateStatusCWDMismatch
)

var defaultFileStateRegistry = NewFileStateRegistry()

// FileStateRegistry tracks the file snapshot a task last read or wrote.
type FileStateRegistry = filesystem.FileStateRegistry

// FileStateSnapshot is the redacted state evidence returned by file tools.
type FileStateSnapshot = filesystem.FileStateSnapshot

type fileStateCheck = filesystem.FileStateCheck

// NewFileStateRegistry returns an empty file-state registry.
func NewFileStateRegistry() *FileStateRegistry {
	return filesystem.NewFileStateRegistry()
}

func fileStatePayload(snapshot FileStateSnapshot) map[string]any {
	return filesystem.FileStatePayload(snapshot)
}

func fileStateErrorPayload(path string, check *fileStateCheck) map[string]any {
	return filesystem.FileStateErrorPayload(path, check)
}

func normalizeFileTaskID(taskID string) string {
	return filesystem.NormalizeFileTaskID(taskID)
}
