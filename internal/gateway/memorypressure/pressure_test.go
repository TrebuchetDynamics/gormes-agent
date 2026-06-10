package memorypressure

import (
	"strings"
	"testing"
	"time"
)

func TestFormatClampsNegativeCounters(t *testing.T) {
	got := Format(Evidence{
		Status:          StatusWarn,
		RSSMB:           -900,
		WarnRSSMB:       -800,
		CriticalRSSMB:   -1200,
		UptimeSeconds:   -30,
		GoRoutines:      -7,
		TargetPID:       -42,
		TargetStartTime: -99,
	})
	for _, forbidden := range []string{"rss=-", "warn=-", "critical=-", "uptime=-", "goroutines=-", "target_pid=-", "target_start_time=-"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("Format leaked negative counter %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "rss=0MB warn=0MB critical=0MB") {
		t.Fatalf("Format missing clamped memory counters in %q", got)
	}
}

func TestFormatSanitizesMultilineStatus(t *testing.T) {
	got := Format(Evidence{Status: "warn\nforged=true", RSSMB: 900, WarnRSSMB: 800, CriticalRSSMB: 1200})
	if strings.Contains(got, "\nforged") {
		t.Fatalf("Format leaked multiline status in %q", got)
	}
	if !strings.Contains(got, "memory_pressure: warn forged=true") {
		t.Fatalf("Format missing sanitized status in %q", got)
	}
}

func TestFormatSanitizesMultilineEvidence(t *testing.T) {
	got := Format(Evidence{
		Status:        StatusWarn,
		RSSMB:         900,
		WarnRSSMB:     800,
		CriticalRSSMB: 1200,
		Action:        "restart_requested\nforged=true",
		Evidence:      []string{"memory_pressure_warn", "extra\nline"},
		Message:       "warning\nforged status",
	})
	for _, forbidden := range []string{"\nforged", "extra\nline", "warning\nforged"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("Format leaked multiline evidence %q in %q", forbidden, got)
		}
	}
	for _, want := range []string{
		"action=restart_requested forged=true",
		"evidence=memory_pressure_warn,extra line",
		"message=\"warning forged status\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Format missing sanitized value %q in %q", want, got)
		}
	}
}

func TestNormalizePolicyBoundsInvalidCriticalAction(t *testing.T) {
	policy := NormalizePolicy(Policy{WarnRSSMB: 800, CriticalRSSMB: 1200, CriticalAction: "restart_requested\nforged=false"})
	if policy.CriticalAction != ActionRestart {
		t.Fatalf("CriticalAction = %q, want bounded default %q", policy.CriticalAction, ActionRestart)
	}

	ev := Evaluate(Sample{RSSBytes: 1300 * BytesPerMegabyte}, policy, Owner{PID: 4242, StartTime: 99}, time.Date(2026, 5, 18, 4, 30, 0, 0, time.UTC))
	if ev.Action != ActionRestart || ev.TargetPID != 4242 || ev.TargetStartTime != 99 {
		t.Fatalf("critical evidence = %+v, want bounded restart action with owner target", ev)
	}
}

func TestEvaluateCriticalRestartClampsInvalidOwnerIdentity(t *testing.T) {
	now := time.Date(2026, 5, 18, 4, 30, 0, 0, time.UTC)
	ev := Evaluate(
		Sample{RSSBytes: 1300 * BytesPerMegabyte},
		Policy{WarnRSSMB: 800, CriticalRSSMB: 1200, CriticalAction: ActionRestart},
		Owner{PID: -42, StartTime: -99},
		now,
	)
	if ev.Status != StatusCritical || ev.Action != ActionRestart {
		t.Fatalf("critical restart evidence = %+v, want critical restart", ev)
	}
	if ev.TargetPID != 0 || ev.TargetStartTime != 0 {
		t.Fatalf("critical restart evidence trusted invalid owner identity: %+v", ev)
	}
	formatted := Format(ev)
	if strings.Contains(formatted, "-42") || strings.Contains(formatted, "-99") {
		t.Fatalf("formatted evidence leaked invalid owner identity: %q", formatted)
	}
}

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
		status PressureStatus
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
