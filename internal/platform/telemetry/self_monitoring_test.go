package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSelfMonitoringBridgeRecordsInjectedSinksAndDegradesNonfatally(t *testing.T) {
	now := time.Date(2026, 4, 29, 12, 30, 0, 0, time.UTC)
	telemetrySink := &recordingSelfMonitoringTelemetrySink{}
	auditSink := &recordingSelfMonitoringAuditSink{err: errors.New("audit unavailable")}
	insightsRecorder := &recordingSelfMonitoringInsightsRecorder{}
	bridge := SelfMonitoringBridge{
		TelemetrySink:    telemetrySink,
		AuditSink:        auditSink,
		InsightsRecorder: insightsRecorder,
		Now:              func() time.Time { return now },
	}

	evidence := bridge.RecordUsage(context.Background(), UsageEvidence{
		SessionID:        "sess-1",
		Provider:         "openai-codex",
		Model:            "gpt-5.5",
		InputTokens:      100,
		CacheReadTokens:  10,
		CacheWriteTokens: 5,
		OutputTokens:     40,
		ReasoningTokens:  7,
		RequestCount:     2,
		ToolCalls:        3,
		ToolErrors:       1,
		EstimatedCostUSD: 0.012345,
		FinishedAt:       now,
	})

	if evidence.EventName != "gormes.provider.usage" {
		t.Fatalf("EventName = %q, want gormes.provider.usage", evidence.EventName)
	}
	if len(telemetrySink.events) != 1 {
		t.Fatalf("telemetry event count = %d, want 1", len(telemetrySink.events))
	}
	if len(auditSink.events) != 1 {
		t.Fatalf("audit event count = %d, want 1", len(auditSink.events))
	}
	if len(insightsRecorder.usage) != 1 {
		t.Fatalf("insights usage count = %d, want 1", len(insightsRecorder.usage))
	}
	if len(evidence.Recorded) != 2 {
		t.Fatalf("recorded sinks = %v, want telemetry and insights only", evidence.Recorded)
	}
	if len(evidence.Degraded) != 1 || evidence.Degraded[0].Sink != "audit" {
		t.Fatalf("degraded evidence = %+v, want one audit degradation", evidence.Degraded)
	}
	if telemetrySink.events[0].Timestamp != now {
		t.Fatalf("event timestamp = %v, want bridge clock %v", telemetrySink.events[0].Timestamp, now)
	}
	if got := telemetrySink.events[0].PayloadSummary["provider"]; got != "openai-codex" {
		t.Fatalf("payload provider = %q, want openai-codex", got)
	}
}

func TestTelemetryEventMatrixMapsLocalEventsAndHostedExporterDivergence(t *testing.T) {
	cases := []struct {
		upstream       string
		local          string
		source         string
		classification DivergenceClassification
		hostedOnly     bool
	}{
		{"hermes.provider.usage", "gormes.provider.usage", "hermes", DivergenceLocal, false},
		{"hermes.tool.completed", "gormes.tool.completed", "hermes", DivergenceLocal, false},
		{"representation.completed", "gormes.goncho.representation.completed", "honcho", DivergenceLocal, false},
		{"dream.run", "gormes.goncho.dream.run", "honcho", DivergenceLocal, false},
		{"agent.iteration", "gormes.goncho.agent.iteration", "honcho", DivergenceLocal, false},
		{"reconciliation.sync_vectors.completed", "gormes.goncho.reconciliation.sync_vectors.completed", "honcho", DivergenceLocal, false},
		{"reasoning.trace", "gormes.goncho.reasoning_trace", "honcho", DivergenceLocal, false},
		{"honcho.prometheus.metrics", "gormes.telemetry.exporter.excluded", "honcho", DivergenceOwnedExcluded, true},
		{"honcho.sentry.trace", "gormes.telemetry.exporter.excluded", "honcho", DivergenceOwnedExcluded, true},
		{"honcho.cloudevents.http", "gormes.telemetry.exporter.excluded", "honcho", DivergenceOwnedExcluded, true},
	}

	for _, tc := range cases {
		t.Run(tc.upstream, func(t *testing.T) {
			entry, ok := LookupTelemetryEvent(tc.upstream)
			if !ok {
				t.Fatalf("LookupTelemetryEvent(%q) not found", tc.upstream)
			}
			if entry.LocalEvent != tc.local || entry.Source != tc.source {
				t.Fatalf("entry = %+v, want local/source %q/%q", entry, tc.local, tc.source)
			}
			if entry.Divergence.Classification != tc.classification {
				t.Fatalf("classification = %q, want %q", entry.Divergence.Classification, tc.classification)
			}
			if entry.HostedExporterOnly != tc.hostedOnly {
				t.Fatalf("HostedExporterOnly = %v, want %v", entry.HostedExporterOnly, tc.hostedOnly)
			}
		})
	}
}

func TestReasoningTraceRecordPreservesTreeShapeAndRedactsPayload(t *testing.T) {
	started := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	finished := started.Add(250 * time.Millisecond)

	record := NewReasoningTraceRecord(ReasoningTraceInput{
		TraceID:         "trace-1",
		TreeNodeID:      "node-2",
		ParentID:        "node-1",
		Level:           2,
		EventType:       "agent.iteration",
		TaskType:        "dialectic_chat",
		Provider:        "openai",
		Model:           "gpt-5.5",
		ReasoningEffort: "high",
		StartedAt:       started,
		FinishedAt:      finished,
		InputTokens:     120,
		OutputTokens:    45,
		ToolCalls:       []string{"honcho_search"},
		Payload: map[string]any{
			"prompt":        "raw prompt with sk-secret and bearer token",
			"authorization": "Bearer provider-token",
			"response":      "assistant response should be summarized, not copied",
		},
	})

	if record.TraceID != "trace-1" || record.TreeNodeID != "node-2" || record.ParentID != "node-1" || record.Level != 2 {
		t.Fatalf("tree fields = %+v, want trace-1/node-2/node-1/2", record)
	}
	if record.EventType != "agent.iteration" || record.DurationMs != 250 {
		t.Fatalf("event/duration = %q/%d, want agent.iteration/250", record.EventType, record.DurationMs)
	}
	if record.PayloadSummary["prompt_sha256"] == "" || record.PayloadSummary["response_sha256"] == "" {
		t.Fatalf("payload summary missing prompt/response hashes: %+v", record.PayloadSummary)
	}
	if len(record.Redactions) == 0 {
		t.Fatalf("redactions empty, want prompt/secret evidence")
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Marshal(record): %v", err)
	}
	for _, forbidden := range []string{"raw prompt", "sk-secret", "Bearer provider-token", "assistant response should"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("record leaked %q in %s", forbidden, raw)
		}
	}

	event := record.TelemetryEvent()
	if event.Name != "gormes.goncho.reasoning_trace" || event.UpstreamEvent != "reasoning.trace" {
		t.Fatalf("trace event names = %q/%q, want gormes.goncho.reasoning_trace/reasoning.trace", event.Name, event.UpstreamEvent)
	}
	if event.TraceID != "trace-1" || event.TreeNodeID != "node-2" || event.ParentID != "node-1" || event.Level != 2 {
		t.Fatalf("trace event tree fields = %+v", event)
	}
}

type recordingSelfMonitoringTelemetrySink struct {
	events []Event
	err    error
}

func (s *recordingSelfMonitoringTelemetrySink) RecordTelemetry(_ context.Context, event Event) error {
	s.events = append(s.events, event)
	return s.err
}

type recordingSelfMonitoringAuditSink struct {
	events []Event
	err    error
}

func (s *recordingSelfMonitoringAuditSink) RecordSelfMonitoringAudit(_ context.Context, event Event) error {
	s.events = append(s.events, event)
	return s.err
}

type recordingSelfMonitoringInsightsRecorder struct {
	usage []UsageEvidence
	err   error
}

func (r *recordingSelfMonitoringInsightsRecorder) RecordSelfMonitoringUsage(_ context.Context, usage UsageEvidence) error {
	r.usage = append(r.usage, usage)
	return r.err
}
