package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	FileStateStatusStale       = "file_stale"
	FileStateStatusDeleted     = "file_deleted"
	FileStateStatusMissing     = "file_state_missing"
	FileStateStatusCWDMismatch = "file_state_cwd_mismatch"
)

// FileStateRegistry tracks the file snapshot a task last read or wrote.
type FileStateRegistry struct {
	mu      sync.Mutex
	entries map[fileStateKey]FileStateSnapshot
}

type fileStateKey struct {
	root   string
	taskID string
	path   string
}

// FileStateSnapshot is the redacted state evidence returned by file tools.
type FileStateSnapshot struct {
	Path          string `json:"path"`
	TaskID        string `json:"task_id"`
	CWD           string `json:"cwd"`
	ReadToken     string `json:"read_token"`
	Hash          string `json:"hash"`
	SizeBytes     int64  `json:"size_bytes"`
	MTimeUnixNano int64  `json:"mtime_unix_nano"`
}

// FileStateCheck describes a failed freshness check before a file mutation.
type FileStateCheck struct {
	Status   string
	Error    string
	Expected FileStateSnapshot
	Current  *FileStateSnapshot
}

// NewFileStateRegistry returns an empty file-state registry.
func NewFileStateRegistry() *FileStateRegistry {
	return &FileStateRegistry{entries: make(map[fileStateKey]FileStateSnapshot)}
}

// Record snapshots a resolved file path for a task.
func (r *FileStateRegistry) Record(root, taskID, cwd, rel, abs string) (FileStateSnapshot, error) {
	if r == nil {
		return FileStateSnapshot{}, nil
	}
	snapshot, err := newFileStateSnapshot(taskID, cwd, rel, abs)
	if err != nil {
		return FileStateSnapshot{}, err
	}
	key := fileStateKey{root: filepath.Clean(root), taskID: NormalizeFileTaskID(taskID), path: rel}
	r.mu.Lock()
	if r.entries == nil {
		r.entries = make(map[fileStateKey]FileStateSnapshot)
	}
	r.entries[key] = snapshot
	r.mu.Unlock()
	return snapshot, nil
}

// Check returns nil when the current file still matches the last recorded state.
func (r *FileStateRegistry) Check(root, taskID, cwd, rel, abs string) *FileStateCheck {
	if r == nil {
		return nil
	}
	key := fileStateKey{root: filepath.Clean(root), taskID: NormalizeFileTaskID(taskID), path: rel}
	r.mu.Lock()
	expected, ok := r.entries[key]
	r.mu.Unlock()
	if !ok {
		if _, err := os.Stat(abs); err == nil {
			return &FileStateCheck{
				Status: FileStateStatusMissing,
				Error:  fmt.Sprintf("%s: %s was not read by this task before editing", FileStateStatusMissing, rel),
			}
		}
		return nil
	}
	if expected.CWD != cwd {
		return &FileStateCheck{
			Status:   FileStateStatusCWDMismatch,
			Error:    fmt.Sprintf("%s: %s was read from cwd %q but edit resolved from cwd %q", FileStateStatusCWDMismatch, rel, expected.CWD, cwd),
			Expected: expected,
		}
	}
	current, err := newFileStateSnapshot(taskID, cwd, rel, abs)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileStateCheck{
				Status:   FileStateStatusDeleted,
				Error:    fmt.Sprintf("%s: %s was deleted after the last read", FileStateStatusDeleted, rel),
				Expected: expected,
			}
		}
		return &FileStateCheck{
			Status:   FileStateStatusStale,
			Error:    fmt.Sprintf("%s: cannot verify %s before editing: %v", FileStateStatusStale, rel, err),
			Expected: expected,
		}
	}
	if current.Hash != expected.Hash || current.SizeBytes != expected.SizeBytes || current.MTimeUnixNano != expected.MTimeUnixNano {
		return &FileStateCheck{
			Status:   FileStateStatusStale,
			Error:    fmt.Sprintf("%s: %s changed since the last read; re-read before editing", FileStateStatusStale, rel),
			Expected: expected,
			Current:  &current,
		}
	}
	return nil
}

func newFileStateSnapshot(taskID, cwd, rel, abs string) (FileStateSnapshot, error) {
	raw, err := os.ReadFile(abs)
	if err != nil {
		return FileStateSnapshot{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return FileStateSnapshot{}, err
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	tokenInput := fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%d", NormalizeFileTaskID(taskID), rel, hash, info.Size(), info.ModTime().UnixNano())
	tokenSum := sha256.Sum256([]byte(tokenInput))
	return FileStateSnapshot{
		Path:          rel,
		TaskID:        NormalizeFileTaskID(taskID),
		CWD:           cwd,
		ReadToken:     hex.EncodeToString(tokenSum[:])[:16],
		Hash:          hash,
		SizeBytes:     info.Size(),
		MTimeUnixNano: info.ModTime().UnixNano(),
	}, nil
}

// FileStatePayload returns transcript-safe file-state evidence.
func FileStatePayload(snapshot FileStateSnapshot) map[string]any {
	if snapshot.Path == "" {
		return nil
	}
	return map[string]any{
		"path":            snapshot.Path,
		"task_id":         snapshot.TaskID,
		"cwd":             snapshot.CWD,
		"read_token":      snapshot.ReadToken,
		"hash":            snapshot.Hash,
		"size_bytes":      snapshot.SizeBytes,
		"mtime_unix_nano": snapshot.MTimeUnixNano,
	}
}

// FileStateErrorPayload returns transcript-safe stale-file evidence.
func FileStateErrorPayload(path string, check *FileStateCheck) map[string]any {
	payload := map[string]any{
		"path":   path,
		"status": check.Status,
		"error":  check.Error,
	}
	if expected := FileStatePayload(check.Expected); expected != nil {
		payload["file_state"] = expected
	}
	if check.Current != nil {
		payload["current_file_state"] = FileStatePayload(*check.Current)
	}
	return payload
}

// NormalizeFileTaskID returns the stable task identity used for file-state keys.
func NormalizeFileTaskID(taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return "default"
	}
	return taskID
}
