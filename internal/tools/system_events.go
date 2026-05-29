package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
)

const (
	SystemEventCodeEnqueued    = "system_event_enqueued"
	SystemEventCodeHeartbeat   = "system_heartbeat_updated"
	SystemEventCodePresence    = "system_presence_updated"
	SystemEventCodeUnavailable = "system_unavailable"

	SystemEventModeNextHeartbeat SystemEventMode = "next-heartbeat"
	SystemEventModeNow           SystemEventMode = "now"

	defaultSystemEventQueueLimit = 256
)

type SystemEventMode string

type SystemEventsOptions struct {
	StatePath  string
	AuditPath  string
	QueueLimit int
	Now        func() time.Time
}

type SystemEventsManager struct {
	statePath  string
	auditPath  string
	queueLimit int
	now        func() time.Time
}

type SystemEventRequest struct {
	Text string          `json:"text"`
	Mode SystemEventMode `json:"mode"`
}

type SystemEvent struct {
	ID         string          `json:"id"`
	Text       string          `json:"text"`
	Mode       SystemEventMode `json:"mode"`
	EnqueuedAt time.Time       `json:"enqueued_at"`
}

type SystemHeartbeatState struct {
	Enabled    bool      `json:"enabled"`
	LastBeatAt time.Time `json:"last_beat_at,omitempty"`
}

type SystemPresenceEntry struct {
	Component  string    `json:"component"`
	Status     string    `json:"status"`
	LastSeenAt time.Time `json:"last_seen_at"`
	Reason     string    `json:"reason,omitempty"`
}

type SystemPresenceUpdate struct {
	Component string `json:"component"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
}

type SystemEventsSnapshot struct {
	Events    []SystemEvent          `json:"events"`
	Heartbeat SystemHeartbeatState   `json:"heartbeat"`
	Presence  []SystemPresenceEntry  `json:"presence"`
	Degraded  []SystemDegradedStatus `json:"degraded,omitempty"`
}

type SystemDegradedStatus struct {
	Code    string `json:"code"`
	Reason  string `json:"reason"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message,omitempty"`
}

type SystemEventResult struct {
	OK        bool                   `json:"ok"`
	Code      string                 `json:"code"`
	Event     SystemEvent            `json:"event,omitempty"`
	Heartbeat SystemHeartbeatResult  `json:"heartbeat"`
	Degraded  []SystemDegradedStatus `json:"degraded,omitempty"`
}

type SystemHeartbeatResult struct {
	Enabled    bool      `json:"enabled"`
	Triggered  bool      `json:"triggered,omitempty"`
	LastBeatAt time.Time `json:"last_beat_at,omitempty"`
}

type SystemPresenceResult struct {
	OK       bool                   `json:"ok"`
	Code     string                 `json:"code"`
	Entry    SystemPresenceEntry    `json:"entry,omitempty"`
	Degraded []SystemDegradedStatus `json:"degraded,omitempty"`
}

type systemEventsState struct {
	Events    []SystemEvent         `json:"events"`
	Heartbeat SystemHeartbeatState  `json:"heartbeat"`
	Presence  []SystemPresenceEntry `json:"presence,omitempty"`
}

func NewSystemEventsManager(opts SystemEventsOptions) SystemEventsManager {
	limit := opts.QueueLimit
	if limit <= 0 {
		limit = defaultSystemEventQueueLimit
	}
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return SystemEventsManager{
		statePath:  strings.TrimSpace(opts.StatePath),
		auditPath:  strings.TrimSpace(opts.AuditPath),
		queueLimit: limit,
		now:        now,
	}
}

func (m SystemEventsManager) EnqueueEvent(ctx context.Context, req SystemEventRequest) SystemEventResult {
	if err := ctx.Err(); err != nil {
		return systemUnavailable("context_cancelled", "", err)
	}
	state, degraded := m.loadState()
	if len(degraded) > 0 {
		return SystemEventResult{OK: false, Code: SystemEventCodeUnavailable, Degraded: degraded}
	}
	if len(state.Events) >= m.queueLimit {
		return SystemEventResult{
			OK:   false,
			Code: SystemEventCodeUnavailable,
			Degraded: []SystemDegradedStatus{{
				Code:    SystemEventCodeUnavailable,
				Reason:  "event_queue_full",
				Path:    m.statePath,
				Message: fmt.Sprintf("system event queue has %d entries (limit %d)", len(state.Events), m.queueLimit),
			}},
		}
	}

	at := m.now().UTC()
	mode := normalizeSystemEventMode(req.Mode)
	event := SystemEvent{
		ID:         "system-event-" + at.Format("20060102T150405.000000000Z"),
		Text:       strings.TrimSpace(req.Text),
		Mode:       mode,
		EnqueuedAt: at,
	}
	auditArgs, _ := json.Marshal(map[string]any{
		"text": event.Text,
		"mode": string(event.Mode),
	})
	if err := audit.NewJSONLWriter(m.auditPath).Record(audit.Record{
		Timestamp: event.EnqueuedAt,
		Source:    "system",
		Tool:      "system_event",
		Args:      auditArgs,
		Status:    "completed",
	}); err != nil {
		return SystemEventResult{
			OK:    false,
			Code:  SystemEventCodeUnavailable,
			Event: event,
			Degraded: []SystemDegradedStatus{{
				Code:    SystemEventCodeUnavailable,
				Reason:  "audit_ledger_unavailable",
				Path:    m.auditPath,
				Message: err.Error(),
			}},
		}
	}

	state.Events = append(state.Events, event)
	result := SystemEventResult{
		OK:    true,
		Code:  SystemEventCodeEnqueued,
		Event: event,
		Heartbeat: SystemHeartbeatResult{
			Enabled: state.Heartbeat.Enabled,
		},
	}
	if mode == SystemEventModeNow {
		if !state.Heartbeat.Enabled {
			result.OK = false
			result.Code = SystemEventCodeUnavailable
			result.Degraded = append(result.Degraded, SystemDegradedStatus{
				Code:    SystemEventCodeUnavailable,
				Reason:  "heartbeat_disabled",
				Path:    m.statePath,
				Message: "heartbeat is disabled",
			})
		} else {
			state.Heartbeat.LastBeatAt = at
			result.Heartbeat.Triggered = true
			result.Heartbeat.LastBeatAt = at
		}
	}
	if degraded := m.saveState(state); len(degraded) > 0 {
		result.OK = false
		result.Code = SystemEventCodeUnavailable
		result.Degraded = append(result.Degraded, degraded...)
	}
	return result
}

func (m SystemEventsManager) Snapshot(ctx context.Context) (SystemEventsSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return SystemEventsSnapshot{}, err
	}
	state, degraded := m.loadState()
	return SystemEventsSnapshot{
		Events:    append([]SystemEvent(nil), state.Events...),
		Heartbeat: state.Heartbeat,
		Presence:  sortedPresence(state.Presence),
		Degraded:  degraded,
	}, nil
}

func (m SystemEventsManager) SetHeartbeat(ctx context.Context, enabled bool) SystemEventResult {
	if err := ctx.Err(); err != nil {
		return systemUnavailable("context_cancelled", "", err)
	}
	state, degraded := m.loadState()
	if len(degraded) > 0 {
		return SystemEventResult{OK: false, Code: SystemEventCodeUnavailable, Degraded: degraded}
	}
	state.Heartbeat.Enabled = enabled
	if degraded := m.saveState(state); len(degraded) > 0 {
		return SystemEventResult{OK: false, Code: SystemEventCodeUnavailable, Degraded: degraded}
	}
	return SystemEventResult{
		OK:   true,
		Code: SystemEventCodeHeartbeat,
		Heartbeat: SystemHeartbeatResult{
			Enabled:    state.Heartbeat.Enabled,
			LastBeatAt: state.Heartbeat.LastBeatAt,
		},
	}
}

func (m SystemEventsManager) RecordHeartbeat(ctx context.Context, reason string) SystemEventResult {
	if err := ctx.Err(); err != nil {
		return systemUnavailable("context_cancelled", "", err)
	}
	state, degraded := m.loadState()
	if len(degraded) > 0 {
		return SystemEventResult{OK: false, Code: SystemEventCodeUnavailable, Degraded: degraded}
	}
	if !state.Heartbeat.Enabled {
		return SystemEventResult{
			OK:   false,
			Code: SystemEventCodeUnavailable,
			Heartbeat: SystemHeartbeatResult{
				Enabled: false,
			},
			Degraded: []SystemDegradedStatus{{
				Code:    SystemEventCodeUnavailable,
				Reason:  "heartbeat_disabled",
				Path:    m.statePath,
				Message: "heartbeat is disabled",
			}},
		}
	}
	now := m.now().UTC()
	state.Heartbeat.LastBeatAt = now
	state.Presence = upsertPresence(state.Presence, SystemPresenceEntry{
		Component:  "heartbeat",
		Status:     "active",
		LastSeenAt: now,
		Reason:     strings.TrimSpace(reason),
	})
	if degraded := m.saveState(state); len(degraded) > 0 {
		return SystemEventResult{OK: false, Code: SystemEventCodeUnavailable, Degraded: degraded}
	}
	return SystemEventResult{
		OK:   true,
		Code: SystemEventCodeHeartbeat,
		Heartbeat: SystemHeartbeatResult{
			Enabled:    true,
			Triggered:  true,
			LastBeatAt: now,
		},
	}
}

func (m SystemEventsManager) UpdatePresence(ctx context.Context, update SystemPresenceUpdate) SystemPresenceResult {
	if err := ctx.Err(); err != nil {
		return SystemPresenceResult{OK: false, Code: SystemEventCodeUnavailable, Degraded: systemUnavailable("context_cancelled", "", err).Degraded}
	}
	state, degraded := m.loadState()
	if len(degraded) > 0 {
		return SystemPresenceResult{OK: false, Code: SystemEventCodeUnavailable, Degraded: degraded}
	}
	component := strings.TrimSpace(update.Component)
	if component == "" {
		component = "gormes"
	}
	status := strings.TrimSpace(update.Status)
	if status == "" {
		status = "active"
	}
	entry := SystemPresenceEntry{
		Component:  component,
		Status:     status,
		LastSeenAt: m.now().UTC(),
		Reason:     strings.TrimSpace(update.Reason),
	}
	state.Presence = upsertPresence(state.Presence, entry)
	if degraded := m.saveState(state); len(degraded) > 0 {
		return SystemPresenceResult{OK: false, Code: SystemEventCodeUnavailable, Entry: entry, Degraded: degraded}
	}
	return SystemPresenceResult{OK: true, Code: SystemEventCodePresence, Entry: entry}
}

func (m SystemEventsManager) loadState() (systemEventsState, []SystemDegradedStatus) {
	state := defaultSystemEventsState()
	if m.statePath == "" {
		return state, nil
	}
	raw, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, []SystemDegradedStatus{{
			Code:    SystemEventCodeUnavailable,
			Reason:  "state_unavailable",
			Path:    m.statePath,
			Message: err.Error(),
		}}
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return defaultSystemEventsState(), []SystemDegradedStatus{{
			Code:    SystemEventCodeUnavailable,
			Reason:  "state_unavailable",
			Path:    m.statePath,
			Message: err.Error(),
		}}
	}
	if !state.Heartbeat.Enabled && state.Heartbeat.LastBeatAt.IsZero() {
		// Preserve explicit disabled zero state. Missing heartbeat fields are
		// normalized by older-state migration below.
		return state, nil
	}
	if !state.Heartbeat.Enabled && len(raw) > 0 && !strings.Contains(string(raw), `"enabled"`) {
		state.Heartbeat.Enabled = true
	}
	return state, nil
}

func (m SystemEventsManager) saveState(state systemEventsState) []SystemDegradedStatus {
	if m.statePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.statePath), 0o755); err != nil {
		return []SystemDegradedStatus{{
			Code:    SystemEventCodeUnavailable,
			Reason:  "state_unavailable",
			Path:    m.statePath,
			Message: err.Error(),
		}}
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return []SystemDegradedStatus{{
			Code:    SystemEventCodeUnavailable,
			Reason:  "state_unavailable",
			Path:    m.statePath,
			Message: err.Error(),
		}}
	}
	if err := os.WriteFile(m.statePath, append(raw, '\n'), 0o600); err != nil {
		return []SystemDegradedStatus{{
			Code:    SystemEventCodeUnavailable,
			Reason:  "state_unavailable",
			Path:    m.statePath,
			Message: err.Error(),
		}}
	}
	return nil
}

func defaultSystemEventsState() systemEventsState {
	return systemEventsState{
		Heartbeat: SystemHeartbeatState{Enabled: true},
	}
}

func normalizeSystemEventMode(mode SystemEventMode) SystemEventMode {
	switch mode {
	case SystemEventModeNow:
		return SystemEventModeNow
	default:
		return SystemEventModeNextHeartbeat
	}
}

func sortedPresence(entries []SystemPresenceEntry) []SystemPresenceEntry {
	out := append([]SystemPresenceEntry(nil), entries...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Component != out[j].Component {
			return out[i].Component < out[j].Component
		}
		return out[i].LastSeenAt.Before(out[j].LastSeenAt)
	})
	return out
}

func FormatSystemStatus(snapshot SystemEventsSnapshot, auditPath string) string {
	if len(snapshot.Degraded) > 0 {
		item := snapshot.Degraded[0]
		return fmt.Sprintf("system: %s reason=%s path=%s audit=%s", SystemEventCodeUnavailable, item.Reason, item.Path, auditPath)
	}
	heartbeat := "disabled"
	if snapshot.Heartbeat.Enabled {
		heartbeat = "enabled"
	}
	line := fmt.Sprintf("system: heartbeat=%s queued_events=%d presence=%d audit=%s", heartbeat, len(snapshot.Events), len(snapshot.Presence), auditPath)
	if !snapshot.Heartbeat.LastBeatAt.IsZero() {
		line += " last_beat_at=" + snapshot.Heartbeat.LastBeatAt.Format(time.RFC3339)
	}
	return line
}

func upsertPresence(entries []SystemPresenceEntry, entry SystemPresenceEntry) []SystemPresenceEntry {
	out := append([]SystemPresenceEntry(nil), entries...)
	for i := range out {
		if out[i].Component == entry.Component {
			out[i] = entry
			return out
		}
	}
	return append(out, entry)
}

func systemUnavailable(reason, path string, err error) SystemEventResult {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return SystemEventResult{
		OK:   false,
		Code: SystemEventCodeUnavailable,
		Degraded: []SystemDegradedStatus{{
			Code:    SystemEventCodeUnavailable,
			Reason:  reason,
			Path:    path,
			Message: message,
		}},
	}
}
