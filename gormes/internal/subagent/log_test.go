package subagent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendRunLog_AppendsJSONLAndCreatesDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deep", "runs.jsonl")

	first := RunRecord{
		RunID:      "run-1",
		Status:     StatusCompleted,
		Summary:    "done",
		StartedAt:  time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 4, 21, 10, 0, 1, 0, time.UTC),
	}
	second := RunRecord{
		RunID:      "run-2",
		Status:     StatusFailed,
		Error:      "boom",
		StartedAt:  time.Date(2026, 4, 21, 10, 1, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 4, 21, 10, 1, 2, 0, time.UTC),
	}

	if err := AppendRunLog(path, first); err != nil {
		t.Fatalf("AppendRunLog(first): %v", err)
	}
	if err := AppendRunLog(path, second); err != nil {
		t.Fatalf("AppendRunLog(second): %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want 2", len(lines))
	}

	var got1, got2 RunRecord
	if err := json.Unmarshal([]byte(lines[0]), &got1); err != nil {
		t.Fatalf("unmarshal first: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &got2); err != nil {
		t.Fatalf("unmarshal second: %v", err)
	}

	if got1.RunID != first.RunID || got1.Status != first.Status || got1.Summary != first.Summary {
		t.Fatalf("first record = %#v, want %#v", got1, first)
	}
	if got1.StartedAt.IsZero() || got1.FinishedAt.IsZero() {
		t.Fatal("first record timestamps must be set")
	}
	if got2.RunID != second.RunID || got2.Status != second.Status || got2.Error != second.Error {
		t.Fatalf("second record = %#v, want %#v", got2, second)
	}
	if got2.StartedAt.IsZero() || got2.FinishedAt.IsZero() {
		t.Fatal("second record timestamps must be set")
	}
}

func TestAppendRunLog_EmptyPathIsNoop(t *testing.T) {
	if err := AppendRunLog("", RunRecord{RunID: "run-empty", Status: StatusCompleted}); err != nil {
		t.Fatalf("AppendRunLog empty path: %v", err)
	}
}
