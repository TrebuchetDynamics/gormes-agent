package cron

import "time"

// DebugPasteSweepTicker is a hermetic cadence gate for the debug paste cleanup
// hook. It does not start goroutines or perform deletion; callers inject the
// sweep function and drive Tick from their scheduler/ticker boundary.
type DebugPasteSweepTicker struct {
	cadence time.Duration
	sweep   func(time.Time)
	next    time.Time
}

func NewDebugPasteSweepTicker(cadence time.Duration, sweep func(time.Time)) *DebugPasteSweepTicker {
	if cadence <= 0 {
		cadence = time.Hour
	}
	if sweep == nil {
		sweep = func(time.Time) {}
	}
	return &DebugPasteSweepTicker{cadence: cadence, sweep: sweep}
}

func (t *DebugPasteSweepTicker) Tick(now time.Time) {
	if t == nil {
		return
	}
	if t.next.IsZero() || !now.Before(t.next) {
		t.sweep(now)
		t.next = now.Add(t.cadence)
	}
}
