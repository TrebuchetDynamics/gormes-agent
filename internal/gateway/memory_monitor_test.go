package gateway

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryMonitorClassifiesPressureAndBoundsCriticalAction(t *testing.T) {
	now := time.Date(2026, 5, 18, 4, 30, 0, 0, time.UTC)
	policy := MemoryPressurePolicy{
		WarnRSSMB:      800,
		CriticalRSSMB:  1200,
		CriticalAction: MemoryPressureActionRestart,
	}
	owner := MemoryMonitorOwner{PID: 4242, StartTime: 99}

	tests := []struct {
		name   string
		rssMB  int
		status MemoryPressureStatus
		action MemoryPressureAction
	}{
		{name: "ok", rssMB: 256, status: MemoryPressureOK, action: MemoryPressureActionNone},
		{name: "warn", rssMB: 900, status: MemoryPressureWarn, action: MemoryPressureActionNone},
		{name: "critical", rssMB: 1300, status: MemoryPressureCritical, action: MemoryPressureActionRestart},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := EvaluateMemoryPressure(MemorySample{
				RSSBytes:       uint64(tt.rssMB) * 1024 * 1024,
				Uptime:         3 * time.Minute,
				GoRoutines:     17,
				GCCollections:  3,
				SampleDuration: 10 * time.Millisecond,
			}, policy, owner, now)

			if ev.Status != tt.status {
				t.Fatalf("Status = %q, want %q: %+v", ev.Status, tt.status, ev)
			}
			if ev.Action != tt.action {
				t.Fatalf("Action = %q, want %q: %+v", ev.Action, tt.action, ev)
			}
			if ev.RSSMB != tt.rssMB || ev.WarnRSSMB != 800 || ev.CriticalRSSMB != 1200 {
				t.Fatalf("RSS evidence = %+v, want rss=%d warn=800 critical=1200", ev, tt.rssMB)
			}
			if ev.UptimeSeconds != 180 || ev.GoRoutines != 17 || ev.GCCollections != 3 {
				t.Fatalf("runtime counters = %+v, want uptime=180 goroutines=17 gc=3", ev)
			}
			if ev.Redacted != true {
				t.Fatalf("Redacted = false, want true: %+v", ev)
			}
			if ev.CheckedAt != now.Format(time.RFC3339Nano) {
				t.Fatalf("CheckedAt = %q, want %q", ev.CheckedAt, now.Format(time.RFC3339Nano))
			}
			if tt.status == MemoryPressureCritical {
				if ev.TargetPID != 4242 || ev.TargetStartTime != 99 {
					t.Fatalf("critical target = pid %d start %d, want bounded current owner 4242/99", ev.TargetPID, ev.TargetStartTime)
				}
			} else if ev.TargetPID != 0 || ev.TargetStartTime != 0 {
				t.Fatalf("non-critical evidence must not target a process: %+v", ev)
			}
		})
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

type fakeMemorySampler struct {
	sample MemorySample
	err    error
}

func (s fakeMemorySampler) SampleMemory(context.Context) (MemorySample, error) {
	return s.sample, s.err
}
