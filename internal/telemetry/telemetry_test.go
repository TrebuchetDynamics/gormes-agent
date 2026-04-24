package telemetry

import (
	"testing"
	"time"
)

func TestSnapshotAfterTicks(t *testing.T) {
	tm := New()
	tm.StartTurn()
	tm.Tick(1)
	tm.Tick(3)
	tm.FinishTurn(12 * time.Millisecond)
	s := tm.Snapshot()
	if s.TokensOutTotal != 3 {
		t.Errorf("out_total = %d, want 3", s.TokensOutTotal)
	}
	if s.LatencyMsLast != 12 {
		t.Errorf("latency = %d, want 12", s.LatencyMsLast)
	}
}

func TestModelPlumbing(t *testing.T) {
	tm := New()
	tm.SetModel("hermes-agent")
	if got := tm.Snapshot().Model; got != "hermes-agent" {
		t.Errorf("Model = %q", got)
	}
}

func TestSetTokensIn(t *testing.T) {
	tm := New()
	tm.SetTokensIn(5)
	tm.SetTokensIn(3)
	if got := tm.Snapshot().TokensInTotal; got != 8 {
		t.Errorf("in_total = %d, want 8 (accumulated)", got)
	}
}
