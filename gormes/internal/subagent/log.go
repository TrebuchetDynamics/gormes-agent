package subagent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunRecord is one append-only JSONL entry for a delegated run.
type RunRecord struct {
	RunID        string       `json:"run_id"`
	Status       ResultStatus `json:"status"`
	Summary      string       `json:"summary,omitempty"`
	Error        string       `json:"error,omitempty"`
	FinishReason string       `json:"finish_reason,omitempty"`
	ToolCalls    []string     `json:"tool_calls,omitempty"`
	StartedAt    time.Time    `json:"started_at"`
	FinishedAt   time.Time    `json:"finished_at"`
}

// AppendRunLog appends one JSONL record, creating parent directories if needed.
func AppendRunLog(path string, rec RunRecord) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(rec)
}
