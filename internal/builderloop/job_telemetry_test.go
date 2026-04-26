package builderloop

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLedgerEvent_JobTelemetryFieldsRoundTrip(t *testing.T) {
	eventTime := time.Date(2026, 4, 26, 4, 0, 0, 0, time.UTC)
	event := LedgerEvent{
		TS:          eventTime,
		RunID:       "run-123",
		Event:       "job_finished",
		Worker:      2,
		Task:        "4/4.A/Azure Foundry",
		Status:      "failed",
		JobID:       "run-123/post-verify/1/2",
		JobKind:     "post_verify_command",
		Attempt:     1,
		Command:     "go test ./...",
		Dir:         "/repo",
		StartedAt:   eventTime.Format(time.RFC3339Nano),
		DurationMS:  1234,
		ExitError:   "exit status 1",
		StdoutTail:  "stdout tail",
		StderrTail:  "stderr tail",
		StdoutBytes: 4096,
		StderrBytes: 128,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got LedgerEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got.JobID != event.JobID ||
		got.JobKind != event.JobKind ||
		got.Attempt != event.Attempt ||
		got.Command != event.Command ||
		got.Dir != event.Dir ||
		got.StartedAt != event.StartedAt ||
		got.DurationMS != event.DurationMS ||
		got.ExitError != event.ExitError ||
		got.StdoutTail != event.StdoutTail ||
		got.StderrTail != event.StderrTail ||
		got.StdoutBytes != event.StdoutBytes ||
		got.StderrBytes != event.StderrBytes {
		t.Fatalf("job telemetry round trip mismatch:\n got: %+v\nwant: %+v", got, event)
	}
}
