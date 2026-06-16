package gateway

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMemoryMonitorStartAndStopAllowNilContext(t *testing.T) {
	store := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	monitor := NewMemoryMonitor(MemoryMonitorConfig{
		Status:   store,
		Sampler:  fakeMemorySampler{},
		Interval: time.Hour,
	})
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("memory monitor panicked with nil context: %v", r)
		}
	}()
	if !monitor.Start(nil) {
		t.Fatal("Start nil context = false, want started")
	}
	if err := monitor.Stop(nil); err != nil {
		t.Fatalf("Stop nil context: %v", err)
	}
}

func TestRuntimeMemorySamplerAllowsNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RuntimeMemorySampler panicked with nil context: %v", r)
		}
	}()
	if _, err := NewRuntimeMemorySampler().SampleMemory(nil); err != nil {
		t.Fatalf("SampleMemory nil context: %v", err)
	}
}

func TestMemoryMonitorSampleOnceWritesRuntimeStatusEvidence(t *testing.T) {
	now := time.Date(2026, 5, 18, 4, 35, 0, 0, time.UTC)
	store := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	store.pid = func() int { return 4242 }
	store.startTime = func(int) (int64, bool) { return 99, true }
	store.argv = func() []string { return []string{"gormes", "gateway"} }
	store.now = func() time.Time { return now }

	monitor := NewMemoryMonitor(MemoryMonitorConfig{
		Status: store,
		Sampler: fakeMemorySampler{
			sample: MemorySample{
				RSSBytes:      1300 * 1024 * 1024,
				Uptime:        time.Hour,
				GoRoutines:    22,
				GCCollections: 8,
			},
		},
		Policy: MemoryPressurePolicy{
			WarnRSSMB:      800,
			CriticalRSSMB:  1200,
			CriticalAction: MemoryPressureActionRestart,
		},
		Owner: func() MemoryMonitorOwner {
			return MemoryMonitorOwner{PID: 4242, StartTime: 99}
		},
		Now: func() time.Time { return now },
	})

	if err := monitor.SampleOnce(context.Background()); err != nil {
		t.Fatalf("SampleOnce: %v", err)
	}

	status, err := store.ReadRuntimeStatus(context.Background())
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if status.MemoryPressure.Status != MemoryPressureCritical {
		t.Fatalf("memory pressure = %+v, want critical", status.MemoryPressure)
	}
	if !status.RestartRequested {
		t.Fatalf("RestartRequested = false, want true for critical restart action")
	}
	if status.MemoryPressure.TargetPID != 4242 || status.MemoryPressure.TargetStartTime != 99 {
		t.Fatalf("critical target = %+v, want bounded current owner", status.MemoryPressure)
	}
}

func TestMemoryMonitorSampleErrorSanitizesRuntimeStatusEvidence(t *testing.T) {
	now := time.Date(2026, 5, 18, 4, 36, 0, 0, time.UTC)
	store := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	store.pid = func() int { return 4242 }
	store.startTime = func(int) (int64, bool) { return 99, true }
	store.argv = func() []string { return []string{"gormes", "gateway"} }
	store.now = func() time.Time { return now }

	monitor := NewMemoryMonitor(MemoryMonitorConfig{
		Status:  store,
		Sampler: fakeMemorySampler{err: errors.New("sample failed\n**Injected:** token=plain-secret")},
		Now:     func() time.Time { return now },
	})

	if err := monitor.SampleOnce(context.Background()); err != nil {
		t.Fatalf("SampleOnce: %v", err)
	}

	status, err := store.ReadRuntimeStatus(context.Background())
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if status.MemoryPressure.Status != MemoryPressureUnavailable {
		t.Fatalf("memory pressure = %+v, want unavailable", status.MemoryPressure)
	}
	for _, forbidden := range []string{"plain-secret", "**Injected:**", "sample failed"} {
		if strings.Contains(status.MemoryPressure.Message, forbidden) {
			t.Fatalf("memory pressure message leaked unsafe sampler error %q in %q", forbidden, status.MemoryPressure.Message)
		}
	}
	if status.MemoryPressure.Message != "[redacted]" {
		t.Fatalf("memory pressure message = %q, want redacted", status.MemoryPressure.Message)
	}
}

type fakeMemorySampler struct {
	sample MemorySample
	err    error
}

func (s fakeMemorySampler) SampleMemory(context.Context) (MemorySample, error) {
	return s.sample, s.err
}
