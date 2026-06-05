package gateway

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/memorypressure"
)

const defaultMemoryMonitorInterval = 5 * time.Minute

type MemoryPressureStatus = memorypressure.Status

const (
	MemoryPressureOK          = memorypressure.StatusOK
	MemoryPressureWarn        = memorypressure.StatusWarn
	MemoryPressureCritical    = memorypressure.StatusCritical
	MemoryPressureUnavailable = memorypressure.StatusUnavailable
)

type MemoryPressureAction = memorypressure.Action

const (
	MemoryPressureActionNone    = memorypressure.ActionNone
	MemoryPressureActionRestart = memorypressure.ActionRestart
)

type MemoryPressurePolicy = memorypressure.Policy

func DefaultMemoryPressurePolicy() MemoryPressurePolicy {
	return memorypressure.DefaultPolicy()
}

type MemoryMonitorOwner = memorypressure.Owner

type MemorySample = memorypressure.Sample

// RuntimeMemoryPressureEvidence is the JSON-safe gateway status surface for
// memory pressure. It intentionally carries process counters only, never
// environment variables, command lines, or paths.
type RuntimeMemoryPressureEvidence = memorypressure.Evidence

func EvaluateMemoryPressure(sample MemorySample, policy MemoryPressurePolicy, owner MemoryMonitorOwner, now time.Time) RuntimeMemoryPressureEvidence {
	return memorypressure.Evaluate(sample, policy, owner, now)
}

func normalizedMemoryPressurePolicy(policy MemoryPressurePolicy) MemoryPressurePolicy {
	return memorypressure.NormalizePolicy(policy)
}

type MemorySampler interface {
	SampleMemory(context.Context) (MemorySample, error)
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
	return memorypressure.Format(evidence)
}
