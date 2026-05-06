package tools

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
	fileStateStatusStale       = "file_stale"
	fileStateStatusDeleted     = "file_deleted"
	fileStateStatusMissing     = "file_state_missing"
	fileStateStatusCWDMismatch = "file_state_cwd_mismatch"
)

var defaultFileStateRegistry = NewFileStateRegistry()

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

type fileStateCheck struct {
	Status   string
	Error    string
	Expected FileStateSnapshot
	Current  *FileStateSnapshot
}

// NewFileStateRegistry returns an empty file-state registry.
func NewFileStateRegistry() *FileStateRegistry {
	return &FileStateRegistry{entries: make(map[fileStateKey]FileStateSnapshot)}
}

func (r *FileStateRegistry) record(root, taskID, cwd, rel, abs string) (FileStateSnapshot, error) {
	if r == nil {
		return FileStateSnapshot{}, nil
	}
	snapshot, err := newFileStateSnapshot(taskID, cwd, rel, abs)
	if err != nil {
		return FileStateSnapshot{}, err
	}
	key := fileStateKey{root: filepath.Clean(root), taskID: normalizeFileTaskID(taskID), path: rel}
	r.mu.Lock()
	if r.entries == nil {
		r.entries = make(map[fileStateKey]FileStateSnapshot)
	}
	r.entries[key] = snapshot
	r.mu.Unlock()
	return snapshot, nil
}

func (r *FileStateRegistry) check(root, taskID, cwd, rel, abs string) *fileStateCheck {
	if r == nil {
		return nil
	}
	key := fileStateKey{root: filepath.Clean(root), taskID: normalizeFileTaskID(taskID), path: rel}
	r.mu.Lock()
	expected, ok := r.entries[key]
	r.mu.Unlock()
	if !ok {
		if _, err := os.Stat(abs); err == nil {
			return &fileStateCheck{
				Status: fileStateStatusMissing,
				Error:  fmt.Sprintf("%s: %s was not read by this task before editing", fileStateStatusMissing, rel),
			}
		}
		return nil
	}
	if expected.CWD != cwd {
		return &fileStateCheck{
			Status:   fileStateStatusCWDMismatch,
			Error:    fmt.Sprintf("%s: %s was read from cwd %q but edit resolved from cwd %q", fileStateStatusCWDMismatch, rel, expected.CWD, cwd),
			Expected: expected,
		}
	}
	current, err := newFileStateSnapshot(taskID, cwd, rel, abs)
	if err != nil {
		if os.IsNotExist(err) {
			return &fileStateCheck{
				Status:   fileStateStatusDeleted,
				Error:    fmt.Sprintf("%s: %s was deleted after the last read", fileStateStatusDeleted, rel),
				Expected: expected,
			}
		}
		return &fileStateCheck{
			Status:   fileStateStatusStale,
			Error:    fmt.Sprintf("%s: cannot verify %s before editing: %v", fileStateStatusStale, rel, err),
			Expected: expected,
		}
	}
	if current.Hash != expected.Hash || current.SizeBytes != expected.SizeBytes || current.MTimeUnixNano != expected.MTimeUnixNano {
		return &fileStateCheck{
			Status:   fileStateStatusStale,
			Error:    fmt.Sprintf("%s: %s changed since the last read; re-read before editing", fileStateStatusStale, rel),
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
	tokenInput := fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%d", normalizeFileTaskID(taskID), rel, hash, info.Size(), info.ModTime().UnixNano())
	tokenSum := sha256.Sum256([]byte(tokenInput))
	return FileStateSnapshot{
		Path:          rel,
		TaskID:        normalizeFileTaskID(taskID),
		CWD:           cwd,
		ReadToken:     hex.EncodeToString(tokenSum[:])[:16],
		Hash:          hash,
		SizeBytes:     info.Size(),
		MTimeUnixNano: info.ModTime().UnixNano(),
	}, nil
}

func fileStatePayload(snapshot FileStateSnapshot) map[string]any {
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

func fileStateErrorPayload(path string, check *fileStateCheck) map[string]any {
	payload := map[string]any{
		"path":   path,
		"status": check.Status,
		"error":  check.Error,
	}
	if expected := fileStatePayload(check.Expected); expected != nil {
		payload["file_state"] = expected
	}
	if check.Current != nil {
		payload["current_file_state"] = fileStatePayload(*check.Current)
	}
	return payload
}

func normalizeFileTaskID(taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return "default"
	}
	return taskID
}
