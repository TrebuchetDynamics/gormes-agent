package diagnostics

import (
	"strings"
	"testing"
	"time"
)

type capturedPasteEvent struct {
	Name   string
	Fields map[string]any
}

type fakePasteLogger struct {
	events []capturedPasteEvent
}

func (l *fakePasteLogger) Log(name string, fields map[string]any) {
	copyFields := make(map[string]any, len(fields))
	for k, v := range fields {
		copyFields[k] = v
	}
	l.events = append(l.events, capturedPasteEvent{Name: name, Fields: copyFields})
}

func TestSlowBracketedPasteDiagnostic_BelowThresholdNoLog(t *testing.T) {
	logger := &fakePasteLogger{}
	rec := NewSlowBracketedPasteDiagnostic(logger, 0)

	rec.Record(SlowBracketedPasteSample{
		Duration:        499 * time.Millisecond,
		PastedContent:   "secret token",
		PastedFilePaths: []string{"/tmp/secret.txt"},
	})

	if len(logger.events) != 0 {
		t.Fatalf("logger.events = %d, want 0 (below default 500ms threshold)", len(logger.events))
	}
}

func TestSlowBracketedPasteDiagnostic_AtOrAboveThresholdLogsEvidence(t *testing.T) {
	logger := &fakePasteLogger{}
	rec := NewSlowBracketedPasteDiagnostic(logger, 0)

	rec.Record(SlowBracketedPasteSample{Duration: 500 * time.Millisecond})
	rec.Record(SlowBracketedPasteSample{Duration: 750 * time.Millisecond})

	if len(logger.events) != 2 {
		t.Fatalf("logger.events len = %d, want 2", len(logger.events))
	}
	for _, ev := range logger.events {
		if ev.Name != "paste_handler_slow" {
			t.Fatalf("event name = %q, want paste_handler_slow", ev.Name)
		}
		if _, ok := ev.Fields["duration_ms"]; !ok {
			t.Fatalf("event fields = %+v, want duration_ms key", ev.Fields)
		}
		if got, ok := ev.Fields["threshold_ms"]; !ok || got != int64(500) {
			t.Fatalf("event fields = %+v, want threshold_ms=500", ev.Fields)
		}
	}
	if got := logger.events[0].Fields["duration_ms"]; got != int64(500) {
		t.Fatalf("event[0].duration_ms = %v, want 500", got)
	}
	if got := logger.events[1].Fields["duration_ms"]; got != int64(750) {
		t.Fatalf("event[1].duration_ms = %v, want 750", got)
	}
}

func TestSlowBracketedPasteDiagnostic_RedactsPasteContent(t *testing.T) {
	logger := &fakePasteLogger{}
	rec := NewSlowBracketedPasteDiagnostic(logger, 0)

	rec.Record(SlowBracketedPasteSample{
		Duration:        750 * time.Millisecond,
		PastedContent:   "MY_API_KEY=sk-9f...",
		PastedFilePaths: []string{"/Users/alice/secret.png", "/tmp/fingerprint.txt"},
	})

	if len(logger.events) != 1 {
		t.Fatalf("logger.events len = %d, want 1", len(logger.events))
	}

	for key, value := range logger.events[0].Fields {
		text := stringerOf(value)
		if strings.Contains(text, "MY_API_KEY") || strings.Contains(text, "sk-9f") {
			t.Fatalf("field %q leaked pasted content: %v", key, value)
		}
		if strings.Contains(text, "secret.png") || strings.Contains(text, "fingerprint.txt") {
			t.Fatalf("field %q leaked pasted file path: %v", key, value)
		}
	}
}

func TestSlowBracketedPasteDiagnostic_CustomThreshold(t *testing.T) {
	logger := &fakePasteLogger{}
	rec := NewSlowBracketedPasteDiagnostic(logger, 200*time.Millisecond)

	rec.Record(SlowBracketedPasteSample{Duration: 199 * time.Millisecond})
	if len(logger.events) != 0 {
		t.Fatalf("custom threshold not honored: %d events at 199ms", len(logger.events))
	}

	rec.Record(SlowBracketedPasteSample{Duration: 200 * time.Millisecond})
	if len(logger.events) != 1 {
		t.Fatalf("logger.events len = %d, want 1 at 200ms with custom threshold", len(logger.events))
	}
	if got, ok := logger.events[0].Fields["threshold_ms"]; !ok || got != int64(200) {
		t.Fatalf("threshold_ms = %v, want 200 (custom)", got)
	}

	defaultRec := NewSlowBracketedPasteDiagnostic(&fakePasteLogger{}, 0)
	if defaultRec.Threshold() != 500*time.Millisecond {
		t.Fatalf("default threshold = %v, want 500ms when none injected", defaultRec.Threshold())
	}
}

func stringerOf(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, ",")
	default:
		return ""
	}
}
