package transcript

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
)

func TestTrajectoryWriterConvertsScratchpadTags(t *testing.T) {
	dir := t.TempDir()
	samplesPath := filepath.Join(dir, "trajectory_samples.jsonl")
	failedPath := filepath.Join(dir, "failed_trajectories.jsonl")
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	writer := NewTrajectoryWriter(TrajectoryWriterOptions{
		SamplesPath: samplesPath,
		FailedPath:  failedPath,
		Now:         func() time.Time { return now },
	})

	evidence := writer.Write(TrajectoryWriteInput{
		Model:     "gpt-5.2",
		Completed: true,
		Conversations: []TrajectoryTurn{
			{From: "human", Value: "show your reasoning"},
			{From: "gpt", Value: "before <REASONING_SCRATCHPAD>private plan</REASONING_SCRATCHPAD> after"},
		},
	})

	if evidence.Code != TrajectoryWriteCompleted {
		t.Fatalf("evidence.Code = %q, want %q: %+v", evidence.Code, TrajectoryWriteCompleted, evidence)
	}
	if evidence.Path != audit.RedactText(samplesPath) || !evidence.Completed || !evidence.Redacted {
		t.Fatalf("evidence = %+v, want completed sample write at %q with redaction", evidence, audit.RedactText(samplesPath))
	}
	if _, err := os.Stat(failedPath); !os.IsNotExist(err) {
		t.Fatalf("failed trajectory path exists or stat failed: %v", err)
	}

	entry := readTrajectoryEntry(t, samplesPath)
	if entry.Timestamp != now.Format(time.RFC3339Nano) || entry.Model != "gpt-5.2" || !entry.Completed {
		t.Fatalf("entry metadata = %+v, want injected timestamp/model/completed", entry)
	}
	got := entry.Conversations[1].Value
	if !strings.Contains(got, "<think>private plan</think>") {
		t.Fatalf("assistant value = %q, want scratchpad converted to think tags", got)
	}
	if strings.Contains(got, "REASONING_SCRATCHPAD") {
		t.Fatalf("assistant value leaked scratchpad tag: %q", got)
	}
}

func TestTrajectoryWriterIncompleteScratchpadRoutesFailed(t *testing.T) {
	dir := t.TempDir()
	samplesPath := filepath.Join(dir, "trajectory_samples.jsonl")
	failedPath := filepath.Join(dir, "failed_trajectories.jsonl")
	writer := NewTrajectoryWriter(TrajectoryWriterOptions{
		SamplesPath: samplesPath,
		FailedPath:  failedPath,
		Now:         func() time.Time { return time.Date(2026, 5, 6, 12, 5, 0, 0, time.UTC) },
	})

	evidence := writer.Write(TrajectoryWriteInput{
		Model:     "gpt-5.2",
		Completed: true,
		Conversations: []TrajectoryTurn{
			{From: "human", Value: "finish this"},
			{From: "gpt", Value: "visible <REASONING_SCRATCHPAD>unfinished"},
		},
	})

	if evidence.Code != TrajectoryWriteCompleted {
		t.Fatalf("evidence.Code = %q, want %q: %+v", evidence.Code, TrajectoryWriteCompleted, evidence)
	}
	if evidence.Path != audit.RedactText(failedPath) || evidence.Completed {
		t.Fatalf("evidence = %+v, want failed trajectory path %q and completed=false", evidence, audit.RedactText(failedPath))
	}
	if _, err := os.Stat(samplesPath); !os.IsNotExist(err) {
		t.Fatalf("sample trajectory path exists or stat failed: %v", err)
	}

	entry := readTrajectoryEntry(t, failedPath)
	if entry.Completed {
		t.Fatalf("entry.Completed = true, want false for incomplete scratchpad")
	}
	if got := entry.Conversations[1].Value; !strings.Contains(got, "visible <think>unfinished") {
		t.Fatalf("assistant value = %q, want open scratchpad converted before failed write", got)
	}
}

func TestTrajectoryWriterAppendsJSONLMetadata(t *testing.T) {
	dir := t.TempDir()
	samplesPath := filepath.Join(dir, "trajectory_samples.jsonl")
	timestamps := []time.Time{
		time.Date(2026, 5, 6, 12, 6, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 12, 7, 0, 0, time.UTC),
	}
	var tick int
	writer := NewTrajectoryWriter(TrajectoryWriterOptions{
		SamplesPath: samplesPath,
		FailedPath:  filepath.Join(dir, "failed_trajectories.jsonl"),
		Now: func() time.Time {
			defer func() { tick++ }()
			return timestamps[tick]
		},
	})

	for _, turn := range []TrajectoryTurn{
		{From: "human", Value: "first"},
		{From: "gpt", Value: "second"},
	} {
		evidence := writer.Write(TrajectoryWriteInput{
			Model:         "gpt-5.2",
			Completed:     true,
			Conversations: []TrajectoryTurn{turn},
		})
		if evidence.Code != TrajectoryWriteCompleted {
			t.Fatalf("Write(%q) evidence = %+v, want completed", turn.Value, evidence)
		}
	}

	entries := readTrajectoryEntries(t, samplesPath)
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Timestamp != timestamps[0].Format(time.RFC3339Nano) ||
		entries[1].Timestamp != timestamps[1].Format(time.RFC3339Nano) {
		t.Fatalf("timestamps = %q/%q, want injected timestamps", entries[0].Timestamp, entries[1].Timestamp)
	}
	if entries[0].Conversations[0].Value != "first" || entries[1].Conversations[0].Value != "second" {
		t.Fatalf("entries = %+v, want append-only first then second", entries)
	}
}

func TestTrajectoryWriterRedactsSecrets(t *testing.T) {
	dir := t.TempDir()
	samplesPath := filepath.Join(dir, "trajectory_samples.jsonl")
	auditRecorder := &recordingTrajectoryAuditRecorder{}
	writer := NewTrajectoryWriter(TrajectoryWriterOptions{
		SamplesPath: samplesPath,
		FailedPath:  filepath.Join(dir, "failed_trajectories.jsonl"),
		Now:         func() time.Time { return time.Date(2026, 5, 6, 12, 10, 0, 0, time.UTC) },
		Audit:       auditRecorder,
	})

	evidence := writer.Write(TrajectoryWriteInput{
		SessionID: "sess-redaction",
		Model:     "gpt-5.2",
		Completed: true,
		Conversations: []TrajectoryTurn{
			{From: "human", Value: "use sk-secret-123456 and Authorization: Bearer provider-token"},
			{From: "gpt", Value: "saved xoxb-slack-secret at /home/alice/.gormes/config.toml"},
		},
	})

	if evidence.Code != TrajectoryWriteCompleted || !evidence.Redacted {
		t.Fatalf("evidence = %+v, want redacted successful write", evidence)
	}
	raw, err := os.ReadFile(samplesPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", samplesPath, err)
	}
	rendered := string(raw)
	for _, leaked := range []string{
		"sk-secret-123456",
		"Bearer provider-token",
		"xoxb-slack-secret",
		"/home/alice/.gormes/config.toml",
	} {
		if strings.Contains(rendered, leaked) {
			t.Fatalf("trajectory JSONL leaked %q in:\n%s", leaked, rendered)
		}
	}
	for _, want := range []string{"[redacted]", "[redacted-home]"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("trajectory JSONL missing %q in:\n%s", want, rendered)
		}
	}

	if len(auditRecorder.records) != 1 {
		t.Fatalf("audit records = %d, want 1", len(auditRecorder.records))
	}
	rec := auditRecorder.records[0]
	if rec.Source != "trajectory_writer" || rec.SessionID != "sess-redaction" || rec.Status != "completed" {
		t.Fatalf("audit record = %+v, want trajectory writer completed evidence", rec)
	}
	if strings.Contains(string(rec.Args)+rec.Error, "sk-secret-123456") ||
		strings.Contains(string(rec.Args)+rec.Error, "/home/alice/.gormes/config.toml") {
		t.Fatalf("audit record leaked secret/home path: %+v", rec)
	}
}

func TestTrajectoryWriterAppendFailureNonfatal(t *testing.T) {
	dir := t.TempDir()
	writer := NewTrajectoryWriter(TrajectoryWriterOptions{
		SamplesPath: dir,
		FailedPath:  filepath.Join(t.TempDir(), "failed_trajectories.jsonl"),
		Now:         func() time.Time { return time.Date(2026, 5, 6, 12, 20, 0, 0, time.UTC) },
	})

	evidence := writer.Write(TrajectoryWriteInput{
		Model:         "gpt-5.2",
		Completed:     true,
		Conversations: []TrajectoryTurn{{From: "human", Value: "hello"}},
	})

	if evidence.Code != TrajectoryWriteFailed {
		t.Fatalf("evidence.Code = %q, want %q: %+v", evidence.Code, TrajectoryWriteFailed, evidence)
	}
	if !evidence.Completed || !evidence.Redacted {
		t.Fatalf("evidence = %+v, want completed response path preserved and redacted evidence", evidence)
	}
	if evidence.Error == "" {
		t.Fatalf("evidence.Error is empty, want nonfatal write failure detail")
	}
}

type recordingTrajectoryAuditRecorder struct {
	records []audit.Record
}

func (r *recordingTrajectoryAuditRecorder) Record(rec audit.Record) error {
	r.records = append(r.records, rec)
	return nil
}

type trajectoryEntryForTest struct {
	Conversations []TrajectoryTurn `json:"conversations"`
	Timestamp     string           `json:"timestamp"`
	Model         string           `json:"model"`
	Completed     bool             `json:"completed"`
}

func readTrajectoryEntry(t *testing.T, path string) trajectoryEntryForTest {
	t.Helper()
	entries := readTrajectoryEntries(t, path)
	if len(entries) != 1 {
		raw, _ := os.ReadFile(path)
		t.Fatalf("line count = %d, want 1\n%s", len(entries), raw)
	}
	return entries[0]
}

func readTrajectoryEntries(t *testing.T, path string) []trajectoryEntryForTest {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	entries := make([]trajectoryEntryForTest, 0, len(lines))
	for _, line := range lines {
		var entry trajectoryEntryForTest
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("Unmarshal trajectory line: %v\n%s", err, line)
		}
		entries = append(entries, entry)
	}
	return entries
}
