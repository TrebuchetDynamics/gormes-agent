package goncho

import (
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

type TelemetryMetrics struct {
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheWriteTokens    int
	ReasoningTokens     int
	QueueItemsProcessed int
	ToolCallsCount      int
	ToolErrors          int
	DurationMs          int64
}

type TelemetryEventInput struct {
	Type       string
	Workspace  string
	SessionKey string
	Peer       string
	RunID      string
	AgentID    string
	ResourceID string
	Iteration  int
	Timestamp  time.Time
	Metrics    TelemetryMetrics
	Payload    map[string]any
}

func NewTelemetryEvent(input TelemetryEventInput) telemetry.Event {
	entry, ok := telemetry.LookupTelemetryEvent(input.Type)
	if !ok {
		entry = telemetry.EventMatrixEntry{
			UpstreamEvent: strings.TrimSpace(input.Type),
			LocalEvent:    "gormes.goncho." + strings.ReplaceAll(strings.ToLower(strings.TrimSpace(input.Type)), " ", "_"),
			Source:        "honcho",
			Category:      "unknown",
			Divergence: telemetry.DivergenceEvidence{
				Classification: telemetry.DivergenceLocal,
				Replacement:    "redacted local telemetry, audit, and insights evidence",
			},
		}
	}
	summary, _ := telemetry.SummarizePayload(input.Payload)
	event := telemetry.Event{
		Name:             entry.LocalEvent,
		UpstreamEvent:    entry.UpstreamEvent,
		Source:           entry.Source,
		Category:         entry.Category,
		Timestamp:        input.Timestamp,
		SessionID:        strings.TrimSpace(input.SessionKey),
		AgentID:          strings.TrimSpace(input.AgentID),
		PeerID:           strings.TrimSpace(input.Peer),
		WorkspaceID:      strings.TrimSpace(input.Workspace),
		RunID:            strings.TrimSpace(input.RunID),
		EventType:        strings.TrimSpace(input.Type),
		TokensIn:         nonNegativeTelemetryMetric(input.Metrics.InputTokens),
		TokensOut:        nonNegativeTelemetryMetric(input.Metrics.OutputTokens),
		CacheReadTokens:  nonNegativeTelemetryMetric(input.Metrics.CacheReadTokens),
		CacheWriteTokens: nonNegativeTelemetryMetric(input.Metrics.CacheWriteTokens),
		ReasoningTokens:  nonNegativeTelemetryMetric(input.Metrics.ReasoningTokens),
		ToolCalls:        nonNegativeTelemetryMetric(input.Metrics.ToolCallsCount),
		ToolErrors:       nonNegativeTelemetryMetric(input.Metrics.ToolErrors),
		QueueItems:       nonNegativeTelemetryMetric(input.Metrics.QueueItemsProcessed),
		DurationMs:       input.Metrics.DurationMs,
		PayloadSummary:   summary,
	}
	if entry.Divergence.Classification != "" {
		divergence := entry.Divergence
		event.Divergence = &divergence
	}
	return telemetry.NormalizeEvent(event, input.Timestamp)
}

func NewReasoningTraceRecord(input telemetry.ReasoningTraceInput) telemetry.ReasoningTraceRecord {
	return telemetry.NewReasoningTraceRecord(input)
}

func nonNegativeTelemetryMetric(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
