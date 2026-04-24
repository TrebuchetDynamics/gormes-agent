package autoloop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendLedgerEventWritesJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "ledger.jsonl")
	event := LedgerEvent{
		TS:     time.Unix(123, 0).UTC(),
		Event:  "claim",
		Worker: 2,
		Task:   "task-10",
		Status: "started",
	}

	if err := AppendLedgerEvent(path, event); err != nil {
		t.Fatalf("AppendLedgerEvent() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Fatal("ledger event missing trailing newline")
	}

	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("ledger line count = %d, want 1", len(lines))
	}

	var got LedgerEvent
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !got.TS.Equal(event.TS) {
		t.Fatalf("TS = %v, want %v", got.TS, event.TS)
	}
	if got.Event != event.Event {
		t.Fatalf("Event = %q, want %q", got.Event, event.Event)
	}
	if got.Worker != event.Worker {
		t.Fatalf("Worker = %d, want %d", got.Worker, event.Worker)
	}
	if got.Task != event.Task {
		t.Fatalf("Task = %q, want %q", got.Task, event.Task)
	}
	if got.Status != event.Status {
		t.Fatalf("Status = %q, want %q", got.Status, event.Status)
	}
}
