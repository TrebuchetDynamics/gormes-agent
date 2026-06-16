package audit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJSONLWriterAppendsStableSchemaInOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tools", "audit.jsonl")
	writer := NewJSONLWriter(path)

	first := Record{
		Timestamp:       time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC),
		Source:          "kernel",
		SessionID:       "sess_123",
		Tool:            "echo",
		Args:            json.RawMessage(`{"text":"hi"}`),
		DurationMs:      14,
		Status:          "completed",
		ResultSizeBytes: 13,
		Error:           "",
	}
	second := Record{
		Timestamp:       time.Date(2026, 4, 22, 10, 0, 1, 0, time.UTC),
		Source:          "delegate_task",
		AgentID:         "sa_456",
		Tool:            "web_search",
		Args:            json.RawMessage(`{"query":"gormes"}`),
		DurationMs:      29,
		Status:          "failed",
		ResultSizeBytes: 0,
		Error:           "synthetic failure",
	}

	if err := writer.Record(first); err != nil {
		t.Fatalf("Record(first): %v", err)
	}
	if err := writer.Record(second); err != nil {
		t.Fatalf("Record(second): %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q): %v", path, err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("audit log mode = %o, want 0600", got)
	}

	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2\nraw=%s", len(lines), raw)
	}

	var gotFirst, gotSecond Record
	if err := json.Unmarshal(lines[0], &gotFirst); err != nil {
		t.Fatalf("Unmarshal(first): %v\nline=%s", err, lines[0])
	}
	if err := json.Unmarshal(lines[1], &gotSecond); err != nil {
		t.Fatalf("Unmarshal(second): %v\nline=%s", err, lines[1])
	}

	if gotFirst.Timestamp != first.Timestamp {
		t.Fatalf("first timestamp = %v, want %v", gotFirst.Timestamp, first.Timestamp)
	}
	if gotFirst.Source != "kernel" {
		t.Fatalf("first source = %q, want %q", gotFirst.Source, "kernel")
	}
	if gotFirst.SessionID != "sess_123" {
		t.Fatalf("first session_id = %q, want %q", gotFirst.SessionID, "sess_123")
	}
	if gotFirst.Tool != "echo" {
		t.Fatalf("first tool = %q, want %q", gotFirst.Tool, "echo")
	}
	if string(gotFirst.Args) != `{"text":"hi"}` {
		t.Fatalf("first args = %s, want %s", gotFirst.Args, `{"text":"hi"}`)
	}
	if gotFirst.DurationMs != 14 {
		t.Fatalf("first duration_ms = %d, want 14", gotFirst.DurationMs)
	}
	if gotFirst.Status != "completed" {
		t.Fatalf("first status = %q, want %q", gotFirst.Status, "completed")
	}
	if gotFirst.ResultSizeBytes != 13 {
		t.Fatalf("first result_size_bytes = %d, want 13", gotFirst.ResultSizeBytes)
	}
	if gotFirst.Error != "" {
		t.Fatalf("first error = %q, want empty", gotFirst.Error)
	}

	if gotSecond.Timestamp != second.Timestamp {
		t.Fatalf("second timestamp = %v, want %v", gotSecond.Timestamp, second.Timestamp)
	}
	if gotSecond.Source != "delegate_task" {
		t.Fatalf("second source = %q, want %q", gotSecond.Source, "delegate_task")
	}
	if gotSecond.AgentID != "sa_456" {
		t.Fatalf("second agent_id = %q, want %q", gotSecond.AgentID, "sa_456")
	}
	if gotSecond.Tool != "web_search" {
		t.Fatalf("second tool = %q, want %q", gotSecond.Tool, "web_search")
	}
	if string(gotSecond.Args) != `{"query":"gormes"}` {
		t.Fatalf("second args = %s, want %s", gotSecond.Args, `{"query":"gormes"}`)
	}
	if gotSecond.DurationMs != 29 {
		t.Fatalf("second duration_ms = %d, want 29", gotSecond.DurationMs)
	}
	if gotSecond.Status != "failed" {
		t.Fatalf("second status = %q, want %q", gotSecond.Status, "failed")
	}
	if gotSecond.ResultSizeBytes != 0 {
		t.Fatalf("second result_size_bytes = %d, want 0", gotSecond.ResultSizeBytes)
	}
	if gotSecond.Error != "synthetic failure" {
		t.Fatalf("second error = %q, want %q", gotSecond.Error, "synthetic failure")
	}
}

func TestJSONLWriterTightensExistingWorldReadableLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tools", "audit.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile fixture: %v", err)
	}

	writer := NewJSONLWriter(path)
	if err := writer.Record(Record{Tool: "echo", Args: json.RawMessage(`{}`)}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q): %v", path, err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("audit log mode = %o, want 0600", got)
	}
}

func TestJSONLWriterSanitizesOversizedAndSensitiveArgs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tools", "audit.jsonl")
	writer := NewJSONLWriter(path)
	longURL := "https://r.jina.ai/http://" + strings.Repeat("r.jina.ai/http://", 500) + "https://www.reddit.com/r/openclaw/s/2AXfTcVSes"
	args, err := json.Marshal(map[string]any{
		"url":     longURL,
		"api_key": "sk-secret-123",
		"nested": map[string]any{
			"Authorization": "Bearer secret-token",
		},
		"command": strings.Repeat("curl ", 1000),
	})
	if err != nil {
		t.Fatalf("Marshal args: %v", err)
	}

	if err := writer.Record(Record{
		Timestamp: time.Date(2026, 5, 1, 23, 10, 0, 0, time.UTC),
		Source:    "kernel",
		SessionID: "sess_reddit",
		Tool:      "web_extract",
		Args:      args,
		Status:    "failed",
		Error:     "provider returned secret-token in a very long error: " + strings.Repeat("detail ", 500),
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if len(raw) > 12_000 {
		t.Fatalf("audit line len = %d, want bounded line <= 12000", len(raw))
	}
	rendered := string(raw)
	for _, leaked := range []string{
		"sk-secret-123",
		"Bearer secret-token",
		strings.Repeat("r.jina.ai/http://", 20),
	} {
		if strings.Contains(rendered, leaked) {
			t.Fatalf("audit log leaked %q in:\n%s", leaked, rendered)
		}
	}
	for _, want := range []string{"[redacted]", "[truncated", "openclaw/s/2AXfTcVSes"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("audit log missing %q in:\n%s", want, rendered)
		}
	}

	var got Record
	if err := json.Unmarshal(bytes.TrimSpace(raw), &got); err != nil {
		t.Fatalf("Unmarshal sanitized audit line: %v\n%s", err, raw)
	}
	if !json.Valid(got.Args) {
		t.Fatalf("sanitized args are invalid JSON: %s", got.Args)
	}
	if strings.Contains(got.Error, "secret-token") || len(got.Error) > 600 {
		t.Fatalf("sanitized error = %q, want redacted bounded error", got.Error)
	}
}
