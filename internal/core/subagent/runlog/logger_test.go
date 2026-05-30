package runlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoggerAppendWritesJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "runs.jsonl")
	logger := New(path)
	if logger == nil {
		t.Fatal("New returned nil")
	}

	finishedAt := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	if err := logger.Append(Record{
		ID:         "sa_1",
		ParentID:   "parent",
		Depth:      1,
		Goal:       "log me",
		Status:     "completed",
		ExitReason: "scripted",
		DurationMs: 42,
		Iterations: 3,
		FinishedAt: finishedAt,
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var got Record
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal: %v\nraw=%s", err, raw)
	}
	if got.ID != "sa_1" || got.Status != "completed" || got.DurationMs != 42 || !got.FinishedAt.Equal(finishedAt) {
		t.Fatalf("record = %#v", got)
	}
}

func TestNewEmptyPathReturnsNil(t *testing.T) {
	if got := New(""); got != nil {
		t.Fatalf("New empty = %#v, want nil", got)
	}
}
