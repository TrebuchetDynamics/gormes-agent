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

func TestTrajectoryWriterAuditRecordRedactsEvidence(t *testing.T) {
	rec := TrajectoryWriteAuditRecord(TrajectoryWriteAuditInput{
		Timestamp: time.Date(2026, 5, 6, 12, 15, 0, 0, time.UTC),
		SessionID: "sess-trajectory",
		Model:     "gpt-5.2",
		Path:      "/home/alice/.gormes/trajectory_samples.jsonl",
		Code:      "trajectory_write_failed",
		Completed: true,
		Redacted:  true,
		Error:     "append failed for /home/alice/.gormes/trajectory_samples.jsonl with Bearer provider-token",
	})

	if rec.Source != "trajectory_writer" || rec.Tool != "trajectory_write" || rec.Status != "failed" {
		t.Fatalf("record envelope = %+v, want trajectory_writer/trajectory_write/failed", rec)
	}
	if rec.SessionID != "sess-trajectory" || rec.Timestamp.IsZero() {
		t.Fatalf("record session/timestamp = %+v, want populated", rec)
	}

	path := filepath.Join(t.TempDir(), "audit.jsonl")
	if err := NewJSONLWriter(path).Record(rec); err != nil {
		t.Fatalf("Record: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	rendered := string(raw)
	for _, leaked := range []string{
		"/home/alice/.gormes/trajectory_samples.jsonl",
		"Bearer provider-token",
	} {
		if strings.Contains(rendered, leaked) {
			t.Fatalf("trajectory audit leaked %q in:\n%s", leaked, rendered)
		}
	}
	for _, want := range []string{"[redacted-home]", "Bearer [redacted]", "trajectory_write_failed"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("trajectory audit missing %q in:\n%s", want, rendered)
		}
	}

	var got Record
	if err := json.Unmarshal(bytes.TrimSpace(raw), &got); err != nil {
		t.Fatalf("Unmarshal audit record: %v\n%s", err, raw)
	}
	var args map[string]any
	if err := json.Unmarshal(got.Args, &args); err != nil {
		t.Fatalf("Unmarshal audit args: %v\n%s", err, got.Args)
	}
	if args["code"] != "trajectory_write_failed" || args["completed"] != true || args["redacted"] != true {
		t.Fatalf("audit args = %+v, want code/completed/redacted evidence", args)
	}
}

func TestJSONLWriterRedactsCommonSecretShapes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	rec := Record{
		Source: "kernel",
		Tool:   "terminal",
		Args: json.RawMessage(`{
			"command": "printf $OPENAI_API_KEY",
			"env": {"OPENAI_API_KEY": "sk-test-abcdefghijklmnopqrstuvwxyz"},
			"url": "postgres://user:pass@example.test/db"
		}`),
		Status: "failed",
		Error:  "upstream rejected Authorization: Bearer token-secret-1234567890 and AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	if err := NewJSONLWriter(path).Record(rec); err != nil {
		t.Fatalf("Record: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	rendered := string(raw)
	for _, leaked := range []string{
		"sk-test-abcdefghijklmnopqrstuvwxyz",
		"postgres://user:pass",
		"token-secret-1234567890",
		"wJalrXUtnFEMI",
	} {
		if strings.Contains(rendered, leaked) {
			t.Fatalf("audit log leaked %q in:\n%s", leaked, rendered)
		}
	}
	if !strings.Contains(rendered, "[redacted]") {
		t.Fatalf("audit log missing redacted marker:\n%s", rendered)
	}
}
