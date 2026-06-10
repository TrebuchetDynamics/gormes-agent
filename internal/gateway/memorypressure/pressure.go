package memorypressure

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultWarnRSSMB     = 1024
	DefaultCriticalRSSMB = 1536
	BytesPerMegabyte     = 1024 * 1024
)

type PressureStatus string

const (
	StatusOK          PressureStatus = "ok"
	StatusWarn        PressureStatus = "warn"
	StatusCritical    PressureStatus = "critical"
	StatusUnavailable PressureStatus = "unavailable"
)

type Action string

const (
	ActionNone    Action = "none"
	ActionRestart Action = "restart_requested"
)

type Policy struct {
	WarnRSSMB      int
	CriticalRSSMB  int
	CriticalAction Action
}

func DefaultPolicy() Policy {
	return Policy{
		WarnRSSMB:      DefaultWarnRSSMB,
		CriticalRSSMB:  DefaultCriticalRSSMB,
		CriticalAction: ActionRestart,
	}
}

func NormalizePolicy(policy Policy) Policy {
	if policy.WarnRSSMB <= 0 {
		policy.WarnRSSMB = DefaultWarnRSSMB
	}
	if policy.CriticalRSSMB <= 0 {
		policy.CriticalRSSMB = DefaultCriticalRSSMB
	}
	if policy.CriticalRSSMB < policy.WarnRSSMB {
		policy.CriticalRSSMB = policy.WarnRSSMB
	}
	switch policy.CriticalAction {
	case ActionRestart, ActionNone:
	case "":
		policy.CriticalAction = ActionRestart
	default:
		policy.CriticalAction = ActionRestart
	}
	return policy
}

type Owner struct {
	PID       int
	StartTime int64
}

type Sample struct {
	RSSBytes       uint64
	Uptime         time.Duration
	GoRoutines     int
	GCCollections  uint32
	SampleDuration time.Duration
}

type Evidence struct {
	Status          PressureStatus `json:"status,omitempty"`
	RSSMB           int            `json:"rss_mb,omitempty"`
	WarnRSSMB       int            `json:"warn_rss_mb,omitempty"`
	CriticalRSSMB   int            `json:"critical_rss_mb,omitempty"`
	UptimeSeconds   int64          `json:"uptime_seconds,omitempty"`
	GoRoutines      int            `json:"goroutines,omitempty"`
	GCCollections   uint32         `json:"gc_collections,omitempty"`
	Action          Action         `json:"action,omitempty"`
	TargetPID       int            `json:"target_pid,omitempty"`
	TargetStartTime int64          `json:"target_start_time,omitempty"`
	Evidence        []string       `json:"evidence,omitempty"`
	Message         string         `json:"message,omitempty"`
	CheckedAt       string         `json:"checked_at,omitempty"`
	Redacted        bool           `json:"redacted"`
}

func Evaluate(sample Sample, policy Policy, owner Owner, now time.Time) Evidence {
	policy = NormalizePolicy(policy)
	rssMB := int(sample.RSSBytes / BytesPerMegabyte)
	evidence := Evidence{
		Status:        StatusOK,
		RSSMB:         rssMB,
		WarnRSSMB:     policy.WarnRSSMB,
		CriticalRSSMB: policy.CriticalRSSMB,
		UptimeSeconds: int64(sample.Uptime.Seconds()),
		GoRoutines:    sample.GoRoutines,
		GCCollections: sample.GCCollections,
		Action:        ActionNone,
		CheckedAt:     now.UTC().Format(time.RFC3339Nano),
		Redacted:      true,
		Evidence:      []string{"memory_pressure_ok"},
		Message:       "gateway RSS is below warning threshold",
	}
	switch {
	case rssMB >= policy.CriticalRSSMB:
		evidence.Status = StatusCritical
		evidence.Action = policy.CriticalAction
		evidence.Evidence = []string{"memory_pressure_critical"}
		evidence.Message = "gateway RSS is above critical threshold"
		if evidence.Action == ActionRestart {
			if owner.PID > 0 {
				evidence.TargetPID = owner.PID
			}
			if owner.StartTime > 0 {
				evidence.TargetStartTime = owner.StartTime
			}
			evidence.Evidence = append(evidence.Evidence, "memory_pressure_restart_requested")
		}
	case rssMB >= policy.WarnRSSMB:
		evidence.Status = StatusWarn
		evidence.Evidence = []string{"memory_pressure_warn"}
		evidence.Message = "gateway RSS is above warning threshold"
	}
	return evidence
}

func Format(evidence Evidence) string {
	if evidence.Status == "" {
		return ""
	}
	parts := []string{
		fmt.Sprintf("memory_pressure: %s", formatValue(string(evidence.Status))),
		fmt.Sprintf("rss=%dMB", nonNegativeInt(evidence.RSSMB)),
		fmt.Sprintf("warn=%dMB", nonNegativeInt(evidence.WarnRSSMB)),
		fmt.Sprintf("critical=%dMB", nonNegativeInt(evidence.CriticalRSSMB)),
	}
	if uptime := nonNegativeInt64(evidence.UptimeSeconds); uptime > 0 {
		parts = append(parts, fmt.Sprintf("uptime=%ds", uptime))
	}
	if goroutines := nonNegativeInt(evidence.GoRoutines); goroutines > 0 {
		parts = append(parts, fmt.Sprintf("goroutines=%d", goroutines))
	}
	if evidence.GCCollections > 0 {
		parts = append(parts, fmt.Sprintf("gc=%d", evidence.GCCollections))
	}
	if evidence.Action != "" && evidence.Action != ActionNone {
		parts = append(parts, "action="+formatValue(string(evidence.Action)))
	}
	if evidence.TargetPID > 0 {
		parts = append(parts, fmt.Sprintf("target_pid=%d", evidence.TargetPID))
	}
	if evidence.TargetStartTime > 0 {
		parts = append(parts, fmt.Sprintf("target_start_time=%d", evidence.TargetStartTime))
	}
	if len(evidence.Evidence) > 0 {
		items := make([]string, 0, len(evidence.Evidence))
		for _, item := range evidence.Evidence {
			if item = formatValue(item); item != "" {
				items = append(items, item)
			}
		}
		parts = append(parts, "evidence="+strings.Join(items, ","))
	}
	if evidence.Message != "" {
		parts = append(parts, "message="+strconv.Quote(formatValue(evidence.Message)))
	}
	return strings.Join(parts, " ")
}

func formatValue(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func nonNegativeInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func nonNegativeInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
