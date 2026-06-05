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

type Status string

const (
	StatusOK          Status = "ok"
	StatusWarn        Status = "warn"
	StatusCritical    Status = "critical"
	StatusUnavailable Status = "unavailable"
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
	if policy.CriticalAction == "" {
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
	Status          Status   `json:"status,omitempty"`
	RSSMB           int      `json:"rss_mb,omitempty"`
	WarnRSSMB       int      `json:"warn_rss_mb,omitempty"`
	CriticalRSSMB   int      `json:"critical_rss_mb,omitempty"`
	UptimeSeconds   int64    `json:"uptime_seconds,omitempty"`
	GoRoutines      int      `json:"goroutines,omitempty"`
	GCCollections   uint32   `json:"gc_collections,omitempty"`
	Action          Action   `json:"action,omitempty"`
	TargetPID       int      `json:"target_pid,omitempty"`
	TargetStartTime int64    `json:"target_start_time,omitempty"`
	Evidence        []string `json:"evidence,omitempty"`
	Message         string   `json:"message,omitempty"`
	CheckedAt       string   `json:"checked_at,omitempty"`
	Redacted        bool     `json:"redacted"`
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
			evidence.TargetPID = owner.PID
			evidence.TargetStartTime = owner.StartTime
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
		fmt.Sprintf("memory_pressure: %s", evidence.Status),
		fmt.Sprintf("rss=%dMB", evidence.RSSMB),
		fmt.Sprintf("warn=%dMB", evidence.WarnRSSMB),
		fmt.Sprintf("critical=%dMB", evidence.CriticalRSSMB),
	}
	if evidence.UptimeSeconds > 0 {
		parts = append(parts, fmt.Sprintf("uptime=%ds", evidence.UptimeSeconds))
	}
	if evidence.GoRoutines > 0 {
		parts = append(parts, fmt.Sprintf("goroutines=%d", evidence.GoRoutines))
	}
	if evidence.GCCollections > 0 {
		parts = append(parts, fmt.Sprintf("gc=%d", evidence.GCCollections))
	}
	if evidence.Action != "" && evidence.Action != ActionNone {
		parts = append(parts, "action="+string(evidence.Action))
	}
	if evidence.TargetPID > 0 {
		parts = append(parts, fmt.Sprintf("target_pid=%d", evidence.TargetPID))
	}
	if evidence.TargetStartTime > 0 {
		parts = append(parts, fmt.Sprintf("target_start_time=%d", evidence.TargetStartTime))
	}
	if len(evidence.Evidence) > 0 {
		parts = append(parts, "evidence="+strings.Join(evidence.Evidence, ","))
	}
	if evidence.Message != "" {
		parts = append(parts, "message="+strconv.Quote(evidence.Message))
	}
	return strings.Join(parts, " ")
}
