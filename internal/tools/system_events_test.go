package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
)

func TestSystemEventsEnqueueAuditAndHeartbeat(t *testing.T) {
	now := time.Date(2026, 5, 4, 9, 30, 0, 0, time.UTC)
	statePath := filepath.Join(t.TempDir(), "system", "state.json")
	auditPath := filepath.Join(t.TempDir(), "tools", "audit.jsonl")
	manager := NewSystemEventsManager(SystemEventsOptions{
		StatePath: statePath,
		AuditPath: auditPath,
		Now: func() time.Time {
			return now
		},
	})

	result := manager.EnqueueEvent(context.Background(), SystemEventRequest{
		Text: "gateway restart",
		Mode: SystemEventModeNow,
	})
	if !result.OK {
		t.Fatalf("OK = false, result=%+v", result)
	}
	if result.Code != SystemEventCodeEnqueued {
		t.Fatalf("Code = %q, want %q", result.Code, SystemEventCodeEnqueued)
	}
	if result.Event.Text != "gateway restart" || !result.Event.EnqueuedAt.Equal(now) {
		t.Fatalf("event = %+v, want text and timestamp", result.Event)
	}
	if !result.Heartbeat.Triggered || !result.Heartbeat.LastBeatAt.Equal(now) {
		t.Fatalf("heartbeat = %+v, want triggered at fixed time", result.Heartbeat)
	}

	snapshot, err := manager.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snapshot.Events) != 1 || snapshot.Events[0].Text != "gateway restart" {
		t.Fatalf("events = %+v, want one queued gateway restart", snapshot.Events)
	}
	if !snapshot.Heartbeat.Enabled || !snapshot.Heartbeat.LastBeatAt.Equal(now) {
		t.Fatalf("snapshot heartbeat = %+v, want enabled with last beat", snapshot.Heartbeat)
	}

	raw, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("ReadFile audit: %v", err)
	}
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	if len(lines) != 1 {
		t.Fatalf("audit lines = %d, want 1\n%s", len(lines), raw)
	}
	var record audit.Record
	if err := json.Unmarshal(lines[0], &record); err != nil {
		t.Fatalf("audit record JSON: %v\n%s", err, lines[0])
	}
	if record.Source != "system" || record.Tool != "system_event" || record.Status != "completed" {
		t.Fatalf("audit envelope = %+v, want system/system_event completed", record)
	}
	for _, want := range []string{`"text":"gateway restart"`, `"mode":"now"`} {
		if !strings.Contains(string(record.Args), want) {
			t.Fatalf("audit args missing %s: %s", want, record.Args)
		}
	}
}

func TestSystemEventsHeartbeatControlsAndDisabledWakeDegrades(t *testing.T) {
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	statePath := filepath.Join(t.TempDir(), "system", "state.json")
	auditPath := filepath.Join(t.TempDir(), "tools", "audit.jsonl")
	manager := NewSystemEventsManager(SystemEventsOptions{
		StatePath: statePath,
		AuditPath: auditPath,
		Now: func() time.Time {
			return now
		},
	})

	disabled := manager.SetHeartbeat(context.Background(), false)
	if disabled.OK != true || disabled.Heartbeat.Enabled {
		t.Fatalf("disable result = %+v, want ok disabled heartbeat", disabled)
	}

	result := manager.EnqueueEvent(context.Background(), SystemEventRequest{
		Text: "gateway restart",
		Mode: SystemEventModeNow,
	})
	if result.OK {
		t.Fatalf("OK = true, want degraded disabled heartbeat result")
	}
	if result.Code != SystemEventCodeUnavailable {
		t.Fatalf("Code = %q, want %q", result.Code, SystemEventCodeUnavailable)
	}
	if !systemDegradedReason(result.Degraded, "heartbeat_disabled") {
		t.Fatalf("degraded = %+v, want heartbeat_disabled", result.Degraded)
	}
	if result.Heartbeat.Triggered {
		t.Fatalf("Heartbeat triggered while disabled: %+v", result.Heartbeat)
	}

	snapshot, err := manager.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snapshot.Events) != 1 {
		t.Fatalf("events = %d, want disabled wake to still enqueue the event", len(snapshot.Events))
	}
	if snapshot.Heartbeat.Enabled {
		t.Fatalf("heartbeat enabled = true, want disabled")
	}

	now = now.Add(5 * time.Minute)
	enabled := manager.SetHeartbeat(context.Background(), true)
	if !enabled.OK || !enabled.Heartbeat.Enabled {
		t.Fatalf("enable result = %+v, want ok enabled heartbeat", enabled)
	}
	beat := manager.RecordHeartbeat(context.Background(), "manual")
	if !beat.OK || !beat.Heartbeat.Triggered || !beat.Heartbeat.LastBeatAt.Equal(now) {
		t.Fatalf("beat result = %+v, want heartbeat at updated time", beat)
	}
}

func TestSystemEventsPresenceListsLastSeen(t *testing.T) {
	now := time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC)
	manager := NewSystemEventsManager(SystemEventsOptions{
		StatePath: filepath.Join(t.TempDir(), "system", "state.json"),
		AuditPath: filepath.Join(t.TempDir(), "tools", "audit.jsonl"),
		Now: func() time.Time {
			return now
		},
	})

	gateway := manager.UpdatePresence(context.Background(), SystemPresenceUpdate{
		Component: "gateway",
		Status:    "running",
		Reason:    "boot",
	})
	if !gateway.OK {
		t.Fatalf("gateway presence result = %+v, want ok", gateway)
	}
	now = now.Add(time.Minute)
	worker := manager.UpdatePresence(context.Background(), SystemPresenceUpdate{
		Component: "worker",
		Status:    "idle",
		Reason:    "heartbeat",
	})
	if !worker.OK {
		t.Fatalf("worker presence result = %+v, want ok", worker)
	}

	snapshot, err := manager.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snapshot.Presence) != 2 {
		t.Fatalf("presence = %+v, want two entries", snapshot.Presence)
	}
	if snapshot.Presence[0].Component != "gateway" || snapshot.Presence[0].Status != "running" || snapshot.Presence[0].Reason != "boot" {
		t.Fatalf("gateway presence = %+v", snapshot.Presence[0])
	}
	if snapshot.Presence[1].Component != "worker" || !snapshot.Presence[1].LastSeenAt.Equal(now) {
		t.Fatalf("worker presence = %+v, want last seen at updated time", snapshot.Presence[1])
	}
}

func TestSystemEventsUnavailableEvidence(t *testing.T) {
	t.Run("queue full", func(t *testing.T) {
		manager := NewSystemEventsManager(SystemEventsOptions{
			StatePath:  filepath.Join(t.TempDir(), "system", "state.json"),
			AuditPath:  filepath.Join(t.TempDir(), "tools", "audit.jsonl"),
			QueueLimit: 1,
		})
		first := manager.EnqueueEvent(context.Background(), SystemEventRequest{Text: "one"})
		if !first.OK {
			t.Fatalf("first enqueue = %+v, want ok", first)
		}
		second := manager.EnqueueEvent(context.Background(), SystemEventRequest{Text: "two"})
		if second.OK || second.Code != SystemEventCodeUnavailable || !systemDegradedReason(second.Degraded, "event_queue_full") {
			t.Fatalf("second enqueue = %+v, want queue-full unavailable evidence", second)
		}
	})

	t.Run("audit ledger unavailable", func(t *testing.T) {
		auditPath := t.TempDir()
		manager := NewSystemEventsManager(SystemEventsOptions{
			StatePath: filepath.Join(t.TempDir(), "system", "state.json"),
			AuditPath: auditPath,
		})
		result := manager.EnqueueEvent(context.Background(), SystemEventRequest{Text: "gateway restart"})
		if result.OK || result.Code != SystemEventCodeUnavailable || !systemDegradedReason(result.Degraded, "audit_ledger_unavailable") {
			t.Fatalf("result = %+v, want audit unavailable evidence", result)
		}
		if len(result.Degraded) == 0 || result.Degraded[0].Path != auditPath {
			t.Fatalf("degraded = %+v, want audit path evidence", result.Degraded)
		}
	})
}

func systemDegradedReason(items []SystemDegradedStatus, reason string) bool {
	for _, item := range items {
		if item.Reason == reason {
			return true
		}
	}
	return false
}
