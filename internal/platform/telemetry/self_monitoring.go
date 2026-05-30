package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

type DivergenceClassification string

const (
	DivergenceLocal         DivergenceClassification = "local"
	DivergenceOwnedExcluded DivergenceClassification = "owned_excluded"
)

type DivergenceEvidence struct {
	Classification DivergenceClassification `json:"classification"`
	Rationale      string                   `json:"rationale,omitempty"`
	Replacement    string                   `json:"replacement,omitempty"`
}

type EventMatrixEntry struct {
	UpstreamEvent      string             `json:"upstream_event"`
	LocalEvent         string             `json:"local_event"`
	Source             string             `json:"source"`
	Category           string             `json:"category"`
	HostedExporterOnly bool               `json:"hosted_exporter_only,omitempty"`
	Divergence         DivergenceEvidence `json:"divergence"`
}

type Event struct {
	Name             string              `json:"name"`
	UpstreamEvent    string              `json:"upstream_event,omitempty"`
	Source           string              `json:"source,omitempty"`
	Category         string              `json:"category,omitempty"`
	Timestamp        time.Time           `json:"timestamp"`
	SessionID        string              `json:"session_id,omitempty"`
	AgentID          string              `json:"agent_id,omitempty"`
	PeerID           string              `json:"peer_id,omitempty"`
	WorkspaceID      string              `json:"workspace_id,omitempty"`
	RunID            string              `json:"run_id,omitempty"`
	TraceID          string              `json:"trace_id,omitempty"`
	TreeNodeID       string              `json:"tree_node_id,omitempty"`
	ParentID         string              `json:"parent_id,omitempty"`
	Level            int                 `json:"level,omitempty"`
	EventType        string              `json:"event_type,omitempty"`
	Provider         string              `json:"provider,omitempty"`
	Model            string              `json:"model,omitempty"`
	TaskType         string              `json:"task_type,omitempty"`
	ReasoningEffort  string              `json:"reasoning_effort,omitempty"`
	DurationMs       int64               `json:"duration_ms,omitempty"`
	TokensIn         int                 `json:"tokens_in,omitempty"`
	TokensOut        int                 `json:"tokens_out,omitempty"`
	CacheReadTokens  int                 `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int                 `json:"cache_write_tokens,omitempty"`
	ReasoningTokens  int                 `json:"reasoning_tokens,omitempty"`
	RequestCount     int                 `json:"request_count,omitempty"`
	ToolCalls        int                 `json:"tool_calls,omitempty"`
	ToolErrors       int                 `json:"tool_errors,omitempty"`
	QueueItems       int                 `json:"queue_items,omitempty"`
	PayloadSummary   map[string]string   `json:"payload_summary,omitempty"`
	Divergence       *DivergenceEvidence `json:"divergence,omitempty"`
}

type TelemetrySink interface {
	RecordTelemetry(context.Context, Event) error
}

type AuditSink interface {
	RecordSelfMonitoringAudit(context.Context, Event) error
}

type InsightsRecorder interface {
	RecordSelfMonitoringUsage(context.Context, UsageEvidence) error
}

type SelfMonitoringBridge struct {
	TelemetrySink    TelemetrySink
	AuditSink        AuditSink
	InsightsRecorder InsightsRecorder
	Now              func() time.Time
}

type DegradedEvidence struct {
	Sink    string `json:"sink"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type SelfMonitoringEvidence struct {
	EventName     string             `json:"event_name"`
	UpstreamEvent string             `json:"upstream_event,omitempty"`
	Recorded      []string           `json:"recorded,omitempty"`
	Degraded      []DegradedEvidence `json:"degraded,omitempty"`
}

type UsageEvidence struct {
	SessionID        string    `json:"session_id,omitempty"`
	Provider         string    `json:"provider,omitempty"`
	Model            string    `json:"model,omitempty"`
	InputTokens      int       `json:"input_tokens,omitempty"`
	OutputTokens     int       `json:"output_tokens,omitempty"`
	CacheReadTokens  int       `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int       `json:"cache_write_tokens,omitempty"`
	ReasoningTokens  int       `json:"reasoning_tokens,omitempty"`
	RequestCount     int       `json:"request_count,omitempty"`
	ToolCalls        int       `json:"tool_calls,omitempty"`
	ToolErrors       int       `json:"tool_errors,omitempty"`
	EstimatedCostUSD float64   `json:"estimated_cost_usd,omitempty"`
	FinishedAt       time.Time `json:"finished_at"`
}

type ReasoningTraceInput struct {
	TraceID         string
	TreeNodeID      string
	ParentID        string
	Level           int
	EventType       string
	TaskType        string
	Provider        string
	Model           string
	ReasoningEffort string
	StartedAt       time.Time
	FinishedAt      time.Time
	InputTokens     int
	OutputTokens    int
	ToolCalls       []string
	Payload         map[string]any
}

type ReasoningTraceRecord struct {
	TraceID         string              `json:"trace_id"`
	TreeNodeID      string              `json:"tree_node_id"`
	ParentID        string              `json:"parent_id,omitempty"`
	Level           int                 `json:"level"`
	EventType       string              `json:"event_type"`
	TaskType        string              `json:"task_type,omitempty"`
	Provider        string              `json:"provider,omitempty"`
	Model           string              `json:"model,omitempty"`
	ReasoningEffort string              `json:"reasoning_effort,omitempty"`
	Timestamp       time.Time           `json:"timestamp"`
	DurationMs      int64               `json:"duration_ms,omitempty"`
	InputTokens     int                 `json:"input_tokens,omitempty"`
	OutputTokens    int                 `json:"output_tokens,omitempty"`
	ToolCalls       []string            `json:"tool_calls,omitempty"`
	PayloadSummary  map[string]string   `json:"payload_summary"`
	Redactions      []RedactionEvidence `json:"redactions,omitempty"`
}

type RedactionEvidence struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

var selfMonitoringEventMatrix = []EventMatrixEntry{
	localMatrixEntry("hermes.turn.started", "gormes.turn.started", "hermes", "turn"),
	localMatrixEntry("hermes.turn.completed", "gormes.turn.completed", "hermes", "turn"),
	localMatrixEntry("hermes.provider.usage", "gormes.provider.usage", "hermes", "provider"),
	localMatrixEntry("hermes.provider.account_usage", "gormes.provider.account_usage", "hermes", "provider"),
	localMatrixEntry("hermes.tool.called", "gormes.tool.called", "hermes", "tool"),
	localMatrixEntry("hermes.tool.completed", "gormes.tool.completed", "hermes", "tool"),
	localMatrixEntry("representation.completed", "gormes.goncho.representation.completed", "honcho", "representation"),
	localMatrixEntry("dream.run", "gormes.goncho.dream.run", "honcho", "dream"),
	localMatrixEntry("dream.specialist", "gormes.goncho.dream.specialist", "honcho", "dream"),
	localMatrixEntry("dialectic.completed", "gormes.goncho.dialectic.completed", "honcho", "dialectic"),
	localMatrixEntry("agent.iteration", "gormes.goncho.agent.iteration", "honcho", "agent"),
	localMatrixEntry("agent.tool.conclusions.created", "gormes.goncho.agent.tool.conclusions.created", "honcho", "agent"),
	localMatrixEntry("agent.tool.conclusions.deleted", "gormes.goncho.agent.tool.conclusions.deleted", "honcho", "agent"),
	localMatrixEntry("agent.tool.peer_card.updated", "gormes.goncho.agent.tool.peer_card.updated", "honcho", "agent"),
	localMatrixEntry("agent.tool.summary.created", "gormes.goncho.agent.tool.summary.created", "honcho", "agent"),
	localMatrixEntry("deletion.completed", "gormes.goncho.deletion.completed", "honcho", "deletion"),
	localMatrixEntry("reconciliation.sync_vectors.completed", "gormes.goncho.reconciliation.sync_vectors.completed", "honcho", "reconciliation"),
	localMatrixEntry("reconciliation.cleanup_stale_items.completed", "gormes.goncho.reconciliation.cleanup_stale_items.completed", "honcho", "reconciliation"),
	localMatrixEntry("reasoning.trace", "gormes.goncho.reasoning_trace", "honcho", "reasoning"),
	exporterMatrixEntry("honcho.prometheus.metrics"),
	exporterMatrixEntry("honcho.sentry.trace"),
	exporterMatrixEntry("honcho.cloudevents.http"),
}

func TelemetryEventMatrix() []EventMatrixEntry {
	out := make([]EventMatrixEntry, len(selfMonitoringEventMatrix))
	copy(out, selfMonitoringEventMatrix)
	return out
}

func LookupTelemetryEvent(upstream string) (EventMatrixEntry, bool) {
	needle := strings.ToLower(strings.TrimSpace(upstream))
	for _, entry := range selfMonitoringEventMatrix {
		if strings.ToLower(entry.UpstreamEvent) == needle {
			return entry, true
		}
	}
	return EventMatrixEntry{}, false
}

func (b SelfMonitoringBridge) Record(ctx context.Context, event Event) SelfMonitoringEvidence {
	event = NormalizeEvent(event, b.now())
	evidence := SelfMonitoringEvidence{EventName: event.Name, UpstreamEvent: event.UpstreamEvent}
	if b.TelemetrySink != nil {
		if err := b.TelemetrySink.RecordTelemetry(ctx, event); err != nil {
			evidence.Degraded = append(evidence.Degraded, degraded("telemetry", err))
		} else {
			evidence.Recorded = append(evidence.Recorded, "telemetry")
		}
	}
	if b.AuditSink != nil {
		if err := b.AuditSink.RecordSelfMonitoringAudit(ctx, event); err != nil {
			evidence.Degraded = append(evidence.Degraded, degraded("audit", err))
		} else {
			evidence.Recorded = append(evidence.Recorded, "audit")
		}
	}
	return evidence
}

func (b SelfMonitoringBridge) RecordUsage(ctx context.Context, usage UsageEvidence) SelfMonitoringEvidence {
	evidence := b.Record(ctx, usage.Event())
	if b.InsightsRecorder != nil {
		if err := b.InsightsRecorder.RecordSelfMonitoringUsage(ctx, usage); err != nil {
			evidence.Degraded = append(evidence.Degraded, degraded("insights", err))
		} else {
			evidence.Recorded = append(evidence.Recorded, "insights")
		}
	}
	return evidence
}

func (b SelfMonitoringBridge) RecordReasoningTrace(ctx context.Context, trace ReasoningTraceRecord) SelfMonitoringEvidence {
	return b.Record(ctx, trace.TelemetryEvent())
}

func NormalizeEvent(event Event, now time.Time) Event {
	event.Name = strings.TrimSpace(event.Name)
	event.UpstreamEvent = strings.TrimSpace(event.UpstreamEvent)
	if entry, ok := LookupTelemetryEvent(firstNonEmpty(event.UpstreamEvent, event.Name)); ok {
		if event.Name == "" {
			event.Name = entry.LocalEvent
		}
		if event.UpstreamEvent == "" {
			event.UpstreamEvent = entry.UpstreamEvent
		}
		if strings.TrimSpace(event.Source) == "" {
			event.Source = entry.Source
		}
		if strings.TrimSpace(event.Category) == "" {
			event.Category = entry.Category
		}
		if event.Divergence == nil && entry.Divergence.Classification != "" {
			divergence := entry.Divergence
			event.Divergence = &divergence
		}
	}
	if event.Name == "" && event.UpstreamEvent != "" {
		event.Name = "gormes.telemetry." + strings.ReplaceAll(strings.ToLower(event.UpstreamEvent), " ", "_")
	}
	if event.Timestamp.IsZero() {
		if now.IsZero() {
			now = time.Now().UTC()
		}
		event.Timestamp = now.UTC()
	} else {
		event.Timestamp = event.Timestamp.UTC()
	}
	event.PayloadSummary = SanitizePayloadSummary(event.PayloadSummary)
	return event
}

func (u UsageEvidence) Event() Event {
	entry, _ := LookupTelemetryEvent("hermes.provider.usage")
	summary := map[string]string{
		"provider":           strings.TrimSpace(u.Provider),
		"model":              strings.TrimSpace(u.Model),
		"input_tokens":       strconv.Itoa(nonNegative(u.InputTokens)),
		"output_tokens":      strconv.Itoa(nonNegative(u.OutputTokens)),
		"cache_read_tokens":  strconv.Itoa(nonNegative(u.CacheReadTokens)),
		"cache_write_tokens": strconv.Itoa(nonNegative(u.CacheWriteTokens)),
		"reasoning_tokens":   strconv.Itoa(nonNegative(u.ReasoningTokens)),
		"request_count":      strconv.Itoa(nonNegative(u.RequestCount)),
		"tool_calls":         strconv.Itoa(nonNegative(u.ToolCalls)),
		"tool_errors":        strconv.Itoa(nonNegative(u.ToolErrors)),
	}
	if u.EstimatedCostUSD > 0 {
		summary["estimated_cost_usd"] = fmt.Sprintf("%.6f", u.EstimatedCostUSD)
	}
	finished := u.FinishedAt
	if !finished.IsZero() {
		finished = finished.UTC()
	}
	return Event{
		Name:             entry.LocalEvent,
		UpstreamEvent:    entry.UpstreamEvent,
		Source:           entry.Source,
		Category:         entry.Category,
		Timestamp:        finished,
		SessionID:        strings.TrimSpace(u.SessionID),
		Provider:         strings.TrimSpace(u.Provider),
		Model:            strings.TrimSpace(u.Model),
		TokensIn:         nonNegative(u.InputTokens),
		TokensOut:        nonNegative(u.OutputTokens),
		CacheReadTokens:  nonNegative(u.CacheReadTokens),
		CacheWriteTokens: nonNegative(u.CacheWriteTokens),
		ReasoningTokens:  nonNegative(u.ReasoningTokens),
		RequestCount:     nonNegative(u.RequestCount),
		ToolCalls:        nonNegative(u.ToolCalls),
		ToolErrors:       nonNegative(u.ToolErrors),
		PayloadSummary:   summary,
	}
}

func NewReasoningTraceRecord(input ReasoningTraceInput) ReasoningTraceRecord {
	started := input.StartedAt
	if started.IsZero() {
		started = time.Now().UTC()
	} else {
		started = started.UTC()
	}
	finished := input.FinishedAt
	var durationMS int64
	if !finished.IsZero() {
		finished = finished.UTC()
		if !finished.Before(started) {
			durationMS = int64(finished.Sub(started) / time.Millisecond)
		}
	}
	summary, redactions := SummarizePayload(input.Payload)
	return ReasoningTraceRecord{
		TraceID:         strings.TrimSpace(input.TraceID),
		TreeNodeID:      strings.TrimSpace(input.TreeNodeID),
		ParentID:        strings.TrimSpace(input.ParentID),
		Level:           input.Level,
		EventType:       strings.TrimSpace(input.EventType),
		TaskType:        strings.TrimSpace(input.TaskType),
		Provider:        strings.TrimSpace(input.Provider),
		Model:           strings.TrimSpace(input.Model),
		ReasoningEffort: strings.TrimSpace(input.ReasoningEffort),
		Timestamp:       started,
		DurationMs:      durationMS,
		InputTokens:     nonNegative(input.InputTokens),
		OutputTokens:    nonNegative(input.OutputTokens),
		ToolCalls:       append([]string(nil), input.ToolCalls...),
		PayloadSummary:  summary,
		Redactions:      redactions,
	}
}

func (r ReasoningTraceRecord) TelemetryEvent() Event {
	entry, _ := LookupTelemetryEvent("reasoning.trace")
	return NormalizeEvent(Event{
		Name:            entry.LocalEvent,
		UpstreamEvent:   entry.UpstreamEvent,
		Source:          entry.Source,
		Category:        entry.Category,
		Timestamp:       r.Timestamp,
		TraceID:         r.TraceID,
		TreeNodeID:      r.TreeNodeID,
		ParentID:        r.ParentID,
		Level:           r.Level,
		EventType:       r.EventType,
		Provider:        r.Provider,
		Model:           r.Model,
		TaskType:        r.TaskType,
		ReasoningEffort: r.ReasoningEffort,
		DurationMs:      r.DurationMs,
		TokensIn:        r.InputTokens,
		TokensOut:       r.OutputTokens,
		ToolCalls:       len(r.ToolCalls),
		PayloadSummary:  r.PayloadSummary,
	}, r.Timestamp)
}

func SummarizePayload(payload map[string]any) (map[string]string, []RedactionEvidence) {
	if len(payload) == 0 {
		return map[string]string{}, nil
	}
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := map[string]string{"payload_keys": strings.Join(keys, ",")}
	var redactions []RedactionEvidence
	for _, key := range keys {
		value := payload[key]
		field := strings.TrimSpace(key)
		if field == "" {
			continue
		}
		if sensitiveField(field) {
			out[field] = "<redacted>"
			redactions = append(redactions, RedactionEvidence{Field: field, Reason: "sensitive field redacted"})
			continue
		}
		raw := stablePayloadString(value)
		if summarizeField(field) || sensitiveValue(raw) || complexPayload(value) {
			out[field+"_sha256"] = hashString(raw)
			out[field+"_bytes"] = strconv.Itoa(len(raw))
			reason := "raw payload summarized"
			if sensitiveValue(raw) {
				reason = "payload value contained secret-like material"
			}
			redactions = append(redactions, RedactionEvidence{Field: field, Reason: reason})
			continue
		}
		out[field] = strings.TrimSpace(raw)
	}
	return SanitizePayloadSummary(out), redactions
}

func SanitizePayloadSummary(summary map[string]string) map[string]string {
	if len(summary) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(summary))
	for key, value := range summary {
		field := strings.TrimSpace(key)
		if field == "" {
			continue
		}
		raw := strings.TrimSpace(value)
		if sensitiveField(field) {
			out[field] = "<redacted>"
			continue
		}
		if summarizeField(field) || sensitiveValue(raw) {
			out[field+"_sha256"] = hashString(raw)
			out[field+"_bytes"] = strconv.Itoa(len(raw))
			continue
		}
		out[field] = raw
	}
	return out
}

func localMatrixEntry(upstream, local, source, category string) EventMatrixEntry {
	return EventMatrixEntry{
		UpstreamEvent: upstream,
		LocalEvent:    local,
		Source:        source,
		Category:      category,
		Divergence: DivergenceEvidence{
			Classification: DivergenceLocal,
			Replacement:    "redacted local telemetry, audit, and insights evidence",
		},
	}
}

func exporterMatrixEntry(upstream string) EventMatrixEntry {
	return EventMatrixEntry{
		UpstreamEvent:      upstream,
		LocalEvent:         "gormes.telemetry.exporter.excluded",
		Source:             "honcho",
		Category:           "exporter",
		HostedExporterOnly: true,
		Divergence: DivergenceEvidence{
			Classification: DivergenceOwnedExcluded,
			Rationale:      "hosted Prometheus, Sentry, and CloudEvents HTTP exporters are excluded from local Gormes telemetry until a dedicated exporter row adds them",
			Replacement:    "local deterministic redacted telemetry event plus audit/status evidence",
		},
	}
}

func degraded(sink string, err error) DegradedEvidence {
	return DegradedEvidence{
		Sink:    sink,
		Reason:  "self_monitoring_sink_failed",
		Message: err.Error(),
	}
}

func (b SelfMonitoringBridge) now() time.Time {
	if b.Now == nil {
		return time.Now().UTC()
	}
	return b.Now().UTC()
}

func firstNonEmpty(values ...string) string {
	return textvalue.FirstNonEmptyTrimmed(values...)
}

func stablePayloadString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(raw)
	}
}

func complexPayload(value any) bool {
	switch value.(type) {
	case nil, string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return false
	default:
		return true
	}
}

func sensitiveField(field string) bool {
	normalized := strings.ToLower(strings.TrimSpace(field))
	for _, marker := range []string{"authorization", "api_key", "apikey", "secret", "token", "password", "credential"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func summarizeField(field string) bool {
	normalized := strings.ToLower(strings.TrimSpace(field))
	switch normalized {
	case "prompt", "messages", "message", "input", "output", "response", "content", "thinking_content", "reasoning_content":
		return true
	default:
		return false
	}
}

func sensitiveValue(value string) bool {
	normalized := strings.ToLower(value)
	for _, marker := range []string{"sk-", "bearer ", "provider-token", "api_key", "secret", "token"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func nonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
