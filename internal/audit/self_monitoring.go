package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

type SelfMonitoringRecorder struct {
	Recorder Recorder
}

func NewSelfMonitoringRecorder(recorder Recorder) SelfMonitoringRecorder {
	return SelfMonitoringRecorder{Recorder: recorder}
}

func (r SelfMonitoringRecorder) RecordSelfMonitoringAudit(_ context.Context, event telemetry.Event) error {
	if r.Recorder == nil {
		return nil
	}
	return r.Recorder.Record(SelfMonitoringAuditRecord(event))
}

func SelfMonitoringAuditRecord(event telemetry.Event) Record {
	event = telemetry.NormalizeEvent(event, time.Time{})
	args, err := json.Marshal(map[string]any{
		"name":            event.Name,
		"upstream_event":  event.UpstreamEvent,
		"source":          event.Source,
		"category":        event.Category,
		"workspace_id":    event.WorkspaceID,
		"run_id":          event.RunID,
		"trace_id":        event.TraceID,
		"tree_node_id":    event.TreeNodeID,
		"parent_id":       event.ParentID,
		"level":           event.Level,
		"event_type":      event.EventType,
		"provider":        event.Provider,
		"model":           event.Model,
		"duration_ms":     event.DurationMs,
		"tokens_in":       event.TokensIn,
		"tokens_out":      event.TokensOut,
		"payload_summary": telemetry.SanitizePayloadSummary(event.PayloadSummary),
		"divergence":      event.Divergence,
	})
	if err != nil {
		args = json.RawMessage(`null`)
	}
	return Record{
		Timestamp:  event.Timestamp,
		Source:     "self_monitoring",
		SessionID:  event.SessionID,
		AgentID:    event.AgentID,
		Tool:       event.Name,
		Args:       args,
		DurationMs: event.DurationMs,
		Status:     "completed",
	}
}
