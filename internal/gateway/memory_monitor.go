package gateway

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultMemoryMonitorInterval = 5 * time.Minute
	defaultMemoryWarnRSSMB       = 1024
	defaultMemoryCriticalRSSMB   = 1536
	bytesPerMegabyte             = 1024 * 1024
)

type MemoryPressureStatus string

const (
	MemoryPressureOK          MemoryPressureStatus = "ok"
	MemoryPressureWarn        MemoryPressureStatus = "warn"
	MemoryPressureCritical    MemoryPressureStatus = "critical"
	MemoryPressureUnavailable MemoryPressureStatus = "unavailable"
)

type MemoryPressureAction string

const (
	MemoryPressureActionNone    MemoryPressureAction = "none"
	MemoryPressureActionRestart MemoryPressureAction = "restart_requested"
)

type MemoryPressurePolicy struct {
	WarnRSSMB      int
	CriticalRSSMB  int
	CriticalAction MemoryPressureAction
}

func DefaultMemoryPressurePolicy() MemoryPressurePolicy {
	return MemoryPressurePolicy{
		WarnRSSMB:      defaultMemoryWarnRSSMB,
		CriticalRSSMB:  defaultMemoryCriticalRSSMB,
		CriticalAction: MemoryPressureActionRestart,
	}
}

type MemoryMonitorOwner struct {
	PID       int
	StartTime int64
}

type MemorySample struct {
	RSSBytes       uint64
	Uptime         time.Duration
	GoRoutines     int
	GCCollections  uint32
	SampleDuration time.Duration
}

type MemorySampler interface {
	SampleMemory(context.Context) (MemorySample, error)
}

// RuntimeMemoryPressureEvidence is the JSON-safe gateway status surface for
// memory pressure. It intentionally carries process counters only, never
// environment variables, command lines, or paths.
type RuntimeMemoryPressureEvidence struct {
	Status          MemoryPressureStatus `json:"status,omitempty"`
	RSSMB           int                  `json:"rss_mb,omitempty"`
	WarnRSSMB       int                  `json:"warn_rss_mb,omitempty"`
	CriticalRSSMB   int                  `json:"critical_rss_mb,omitempty"`
	UptimeSeconds   int64                `json:"uptime_seconds,omitempty"`
	GoRoutines      int                  `json:"goroutines,omitempty"`
	GCCollections   uint32               `json:"gc_collections,omitempty"`
	Action          MemoryPressureAction `json:"action,omitempty"`
	TargetPID       int                  `json:"target_pid,omitempty"`
	TargetStartTime int64                `json:"target_start_time,omitempty"`
	Evidence        []string             `json:"evidence,omitempty"`
	Message         string               `json:"message,omitempty"`
	CheckedAt       string               `json:"checked_at,omitempty"`
	Redacted        bool                 `json:"redacted"`
}

func EvaluateMemoryPressure(sample MemorySample, policy MemoryPressurePolicy, owner MemoryMonitorOwner, now time.Time) RuntimeMemoryPressureEvidence {
	policy = normalizedMemoryPressurePolicy(policy)
	rssMB := int(sample.RSSBytes / bytesPerMegabyte)
	evidence := RuntimeMemoryPressureEvidence{
		Status:        MemoryPressureOK,
		RSSMB:         rssMB,
		WarnRSSMB:     policy.WarnRSSMB,
		CriticalRSSMB: policy.CriticalRSSMB,
		UptimeSeconds: int64(sample.Uptime.Seconds()),
		GoRoutines:    sample.GoRoutines,
		GCCollections: sample.GCCollections,
		Action:        MemoryPressureActionNone,
		CheckedAt:     now.UTC().Format(time.RFC3339Nano),
		Redacted:      true,
		Evidence:      []string{"memory_pressure_ok"},
		Message:       "gateway RSS is below warning threshold",
	}
	switch {
	case rssMB >= policy.CriticalRSSMB:
		evidence.Status = MemoryPressureCritical
		evidence.Action = policy.CriticalAction
		evidence.Evidence = []string{"memory_pressure_critical"}
		evidence.Message = "gateway RSS is above critical threshold"
		if evidence.Action == MemoryPressureActionRestart {
			evidence.TargetPID = owner.PID
			evidence.TargetStartTime = owner.StartTime
			evidence.Evidence = append(evidence.Evidence, "memory_pressure_restart_requested")
		}
	case rssMB >= policy.WarnRSSMB:
		evidence.Status = MemoryPressureWarn
		evidence.Evidence = []string{"memory_pressure_warn"}
		evidence.Message = "gateway RSS is above warning threshold"
	}
	return evidence
}

func normalizedMemoryPressurePolicy(policy MemoryPressurePolicy) MemoryPressurePolicy {
	if policy.WarnRSSMB <= 0 {
		policy.WarnRSSMB = defaultMemoryWarnRSSMB
	}
	if policy.CriticalRSSMB <= 0 {
		policy.CriticalRSSMB = defaultMemoryCriticalRSSMB
	}
	if policy.CriticalRSSMB < policy.WarnRSSMB {
		policy.CriticalRSSMB = policy.WarnRSSMB
	}
	if policy.CriticalAction == "" {
		policy.CriticalAction = MemoryPressureActionRestart
	}
	return policy
}

type MemoryMonitorConfig struct {
	Status   RuntimeStatusWriter
	Sampler  MemorySampler
	Policy   MemoryPressurePolicy
	Owner    func() MemoryMonitorOwner
	Now      func() time.Time
	Interval time.Duration
}

type MemoryMonitor struct {
	cfg    MemoryMonitorConfig
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
}

func NewMemoryMonitor(cfg MemoryMonitorConfig) *MemoryMonitor {
	if cfg.Sampler == nil {
		cfg.Sampler = NewRuntimeMemorySampler()
	}
	if cfg.Owner == nil {
		cfg.Owner = currentMemoryMonitorOwner
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	if cfg.Interval <= 0 {
		cfg.Interval = defaultMemoryMonitorInterval
	}
	cfg.Policy = normalizedMemoryPressurePolicy(cfg.Policy)
	return &MemoryMonitor{cfg: cfg}
}

func (m *MemoryMonitor) Start(ctx context.Context) bool {
	if m == nil || m.cfg.Status == nil || m.cfg.Sampler == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		return false
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.done = make(chan struct{}, 1)
	go m.run(runCtx)
	return true
}

func (m *MemoryMonitor) Stop(ctx context.Context) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	cancel := m.cancel
	done := m.done
	m.cancel = nil
	m.done = nil
	m.mu.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *MemoryMonitor) run(ctx context.Context) {
	defer close(m.done)
	_ = m.SampleOnce(ctx)
	ticker := time.NewTicker(m.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = m.SampleOnce(ctx)
		}
	}
}

func (m *MemoryMonitor) SampleOnce(ctx context.Context) error {
	if m == nil || m.cfg.Status == nil || m.cfg.Sampler == nil {
		return nil
	}
	sample, err := m.cfg.Sampler.SampleMemory(ctx)
	if err != nil {
		evidence := RuntimeMemoryPressureEvidence{
			Status:    MemoryPressureUnavailable,
			Action:    MemoryPressureActionNone,
			Message:   err.Error(),
			CheckedAt: m.cfg.Now().UTC().Format(time.RFC3339Nano),
			Redacted:  true,
			Evidence:  []string{"memory_pressure_unavailable"},
		}
		return m.writeEvidence(ctx, evidence)
	}
	evidence := EvaluateMemoryPressure(sample, m.cfg.Policy, m.cfg.Owner(), m.cfg.Now())
	return m.writeEvidence(ctx, evidence)
}

func (m *MemoryMonitor) writeEvidence(ctx context.Context, evidence RuntimeMemoryPressureEvidence) error {
	update := RuntimeStatusUpdate{MemoryPressureEvidence: &evidence}
	if evidence.Action == MemoryPressureActionRestart {
		restartRequested := true
		update.RestartRequested = &restartRequested
	}
	return m.cfg.Status.UpdateRuntimeStatus(ctx, update)
}

type RuntimeMemorySampler struct {
	startedAt time.Time
}

func NewRuntimeMemorySampler() RuntimeMemorySampler {
	return RuntimeMemorySampler{startedAt: time.Now().UTC()}
}

func (s RuntimeMemorySampler) SampleMemory(ctx context.Context) (MemorySample, error) {
	if err := ctx.Err(); err != nil {
		return MemorySample{}, err
	}
	start := time.Now()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	rssBytes := readCurrentRSSBytes()
	if rssBytes == 0 {
		rssBytes = stats.Sys
	}
	return MemorySample{
		RSSBytes:       rssBytes,
		Uptime:         time.Since(s.startedAt),
		GoRoutines:     runtime.NumGoroutine(),
		GCCollections:  stats.NumGC,
		SampleDuration: time.Since(start),
	}, nil
}

func currentMemoryMonitorOwner() MemoryMonitorOwner {
	pid := os.Getpid()
	startTime, _ := procProcessStartTime(pid)
	return MemoryMonitorOwner{PID: pid, StartTime: startTime}
}

func readCurrentRSSBytes() uint64 {
	raw, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 2 {
		return 0
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	pageSize := os.Getpagesize()
	if pageSize <= 0 {
		return 0
	}
	return pages * uint64(pageSize)
}

func FormatMemoryPressureEvidence(evidence RuntimeMemoryPressureEvidence) string {
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
	if evidence.Action != "" && evidence.Action != MemoryPressureActionNone {
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
