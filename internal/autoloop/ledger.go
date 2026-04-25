package autoloop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type LedgerEvent struct {
	TS     time.Time `json:"ts"`
	RunID  string    `json:"run_id,omitempty"`
	Event  string    `json:"event"`
	Worker int       `json:"worker,omitempty"`
	Task   string    `json:"task,omitempty"`
	Branch string    `json:"branch,omitempty"`
	Commit string    `json:"commit,omitempty"`
	Status string    `json:"status,omitempty"`
	Detail string    `json:"detail,omitempty"`
}

func AppendLedgerEvent(path string, event LedgerEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(event)
}
