package memorypressure

import (
	"testing"
	"time"
)

func TestEvaluateClassifiesPressureAndBoundsCriticalAction(t *testing.T) {
	now := time.Date(2026, 5, 18, 4, 30, 0, 0, time.UTC)
	policy := Policy{
		WarnRSSMB:      800,
		CriticalRSSMB:  1200,
		CriticalAction: ActionRestart,
	}
	owner := Owner{PID: 4242, StartTime: 99}

	tests := []struct {
		name   string
		rssMB  int
		status Status
		action Action
	}{
		{name: "ok", rssMB: 256, status: StatusOK, action: ActionNone},
		{name: "warn", rssMB: 900, status: StatusWarn, action: ActionNone},
		{name: "critical", rssMB: 1300, status: StatusCritical, action: ActionRestart},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := Evaluate(Sample{
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
			if tt.status == StatusCritical {
				if ev.TargetPID != 4242 || ev.TargetStartTime != 99 {
					t.Fatalf("critical target = pid %d start %d, want bounded current owner 4242/99", ev.TargetPID, ev.TargetStartTime)
				}
			} else if ev.TargetPID != 0 || ev.TargetStartTime != 0 {
				t.Fatalf("non-critical evidence must not target a process: %+v", ev)
			}
		})
	}
}
