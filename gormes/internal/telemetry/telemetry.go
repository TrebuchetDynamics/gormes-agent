// Package telemetry derives per-session counters from SSE events. In Phase 1
// there is no DB and no external exporter; the kernel reads Snapshot() each
// frame and the TUI renders the numbers in the sidebar.
package telemetry

import "time"

type Snapshot struct {
	Model          string
	TokensInTotal  int
	TokensOutTotal int
	LatencyMsLast  int
	TokensPerSec   float64
}

// Telemetry is NOT goroutine-safe. The kernel holds it on its single owner
// goroutine; no other goroutine touches it. If that invariant changes, add
// a mutex or an atomic snapshot read.
type Telemetry struct {
	snap       Snapshot
	turnStart  time.Time
	turnTokens int
	ema        float64
}

func New() *Telemetry { return &Telemetry{} }

func (t *Telemetry) SetModel(m string) { t.snap.Model = m }

// StartTurn resets the per-turn bookkeeping (tok/s denominator, etc.).
func (t *Telemetry) StartTurn() {
	t.turnStart = time.Now()
	t.turnTokens = 0
}

// Tick records the running tokens-out counter reported by the latest delta.
// It adds the delta from the last tick to the lifetime TokensOutTotal and
// updates the EMA tokens/sec estimator.
func (t *Telemetry) Tick(tokensOut int) {
	delta := tokensOut - t.turnTokens
	if delta < 0 {
		delta = 0
	}
	t.turnTokens = tokensOut
	t.snap.TokensOutTotal += delta
	if el := time.Since(t.turnStart).Seconds(); el > 0 {
		tps := float64(t.turnTokens) / el
		const alpha = 0.2
		t.ema = alpha*tps + (1-alpha)*t.ema
		t.snap.TokensPerSec = t.ema
	}
}

// FinishTurn records the turn's latency.
func (t *Telemetry) FinishTurn(latency time.Duration) {
	t.snap.LatencyMsLast = int(latency / time.Millisecond)
}

// SetTokensIn adds to the lifetime TokensInTotal. Called once per turn when
// the server reports final usage.
func (t *Telemetry) SetTokensIn(n int) { t.snap.TokensInTotal += n }

// Snapshot returns the current counters. Safe to call at any time on the
// owning goroutine.
func (t *Telemetry) Snapshot() Snapshot { return t.snap }
