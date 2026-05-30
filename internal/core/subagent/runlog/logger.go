package runlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Record is one subagent completion entry written as JSONL.
type Record struct {
	ID         string    `json:"id"`
	ParentID   string    `json:"parent_id"`
	Depth      int       `json:"depth"`
	Goal       string    `json:"goal"`
	Status     string    `json:"status"`
	ExitReason string    `json:"exit_reason"`
	DurationMs int64     `json:"duration_ms"`
	Iterations int       `json:"iterations"`
	Error      string    `json:"error,omitempty"`
	FinishedAt time.Time `json:"finished_at"`
}

// Logger appends subagent completion records to a JSONL path.
type Logger struct {
	path string
	mu   sync.Mutex
}

func New(path string) *Logger {
	if path == "" {
		return nil
	}
	return &Logger{path: path}
}

func (l *Logger) Append(record Record) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err
}
