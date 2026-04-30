package cron

import (
	"testing"
	"time"
)

func TestDebugPasteSweep_SchedulerCadence(t *testing.T) {
	start := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	var calls []time.Time
	hook := NewDebugPasteSweepTicker(time.Hour, func(now time.Time) { calls = append(calls, now) })

	hook.Tick(start)
	hook.Tick(start.Add(30 * time.Minute))
	hook.Tick(start.Add(time.Hour))
	hook.Tick(start.Add(time.Hour + time.Minute))
	hook.Tick(start.Add(2 * time.Hour))

	if len(calls) != 3 {
		t.Fatalf("expected 3 hourly sweep calls, got %d: %v", len(calls), calls)
	}
	if !calls[0].Equal(start) || !calls[1].Equal(start.Add(time.Hour)) || !calls[2].Equal(start.Add(2*time.Hour)) {
		t.Fatalf("unexpected call times: %v", calls)
	}
}
