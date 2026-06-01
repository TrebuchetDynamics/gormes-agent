package ledgerlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type AutoloopLedgerEvent struct {
	TS     time.Time `json:"ts"`
	RunID  string    `json:"run_id,omitempty"`
	Event  string    `json:"event"`
	Worker int       `json:"worker,omitempty"`
	Task   string    `json:"task,omitempty"`
	Branch string    `json:"branch,omitempty"`
	Commit string    `json:"commit,omitempty"`
	Status string    `json:"status,omitempty"`
	Detail string    `json:"detail,omitempty"`

	JobID       string `json:"job_id,omitempty"`
	JobKind     string `json:"job_kind,omitempty"`
	Attempt     int    `json:"attempt,omitempty"`
	Command     string `json:"command,omitempty"`
	Dir         string `json:"dir,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	DurationMS  int64  `json:"duration_ms,omitempty"`
	ExitError   string `json:"exit_error,omitempty"`
	StdoutTail  string `json:"stdout_tail,omitempty"`
	StderrTail  string `json:"stderr_tail,omitempty"`
	StdoutBytes int    `json:"stdout_bytes,omitempty"`
	StderrBytes int    `json:"stderr_bytes,omitempty"`
}

func AppendAutoloopLedgerEvent(path string, event AutoloopLedgerEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}
