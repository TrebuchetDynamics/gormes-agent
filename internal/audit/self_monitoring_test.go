package audit

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func TestSelfMonitoringAuditRecordUsesExistingEnvelopeAndRedactsPayload(t *testing.T) {
	event := telemetry.Event{
		Name:          "gormes.goncho.reasoning_trace",
		UpstreamEvent: "reasoning.trace",
		Source:        "honcho",
		SessionID:     "sess-1",
		TraceID:       "trace-1",
		TreeNodeID:    "node-2",
		ParentID:      "node-1",
		Level:         2,
		EventType:     "agent.iteration",
		Timestamp:     time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC),
		PayloadSummary: map[string]string{
			"prompt":        "raw prompt with sk-secret",
			"authorization": "Bearer provider-token",
			"safe":          "summary",
		},
	}

	rec := SelfMonitoringAuditRecord(event)
	if rec.Source != "self_monitoring" {
		t.Fatalf("Source = %q, want self_monitoring", rec.Source)
	}
	if rec.SessionID != "sess-1" || rec.Tool != "gormes.goncho.reasoning_trace" || rec.Status != "completed" {
		t.Fatalf("audit envelope = %+v, want session/tool/completed", rec)
	}
	if rec.Timestamp != event.Timestamp {
		t.Fatalf("Timestamp = %v, want %v", rec.Timestamp, event.Timestamp)
	}

	var args map[string]any
	if err := json.Unmarshal(rec.Args, &args); err != nil {
		t.Fatalf("Unmarshal Args: %v\n%s", err, rec.Args)
	}
	if args["trace_id"] != "trace-1" || args["tree_node_id"] != "node-2" || args["parent_id"] != "node-1" {
		t.Fatalf("trace args = %+v, want trace/tree/parent ids", args)
	}
	for _, forbidden := range []string{"raw prompt", "sk-secret", "Bearer provider-token"} {
		if strings.Contains(string(rec.Args), forbidden) {
			t.Fatalf("audit args leaked %q in %s", forbidden, rec.Args)
		}
	}
}
