package builderloop

import (
	"testing"
	"time"
)

// fixed reference clock so tests stay deterministic without the wall clock.
var backoffEpoch = time.Date(2026, time.April, 26, 17, 0, 0, 0, time.UTC)

func newBackoffState() BackoffState {
	return BackoffState{
		Threshold: 3,
		Baseline:  30 * time.Second,
		Idle:      5 * time.Minute,
	}
}

func TestBackoffState_ShouldPollWhenNotSuppressed(t *testing.T) {
	s := newBackoffState()
	if !s.ShouldPoll(backoffEpoch) {
		t.Fatalf("fresh BackoffState.ShouldPoll(t0) = false, want true")
	}
}

func TestBackoffState_RecordEmptyBelowThreshold(t *testing.T) {
	s := newBackoffState()

	s = s.RecordResult(backoffEpoch, 0)
	if s.ConsecutiveEmpty != 1 {
		t.Fatalf("after one empty: ConsecutiveEmpty = %d, want 1", s.ConsecutiveEmpty)
	}
	if !s.SuppressUntil.IsZero() {
		t.Fatalf("after one empty: SuppressUntil = %s, want zero", s.SuppressUntil)
	}
	if !s.ShouldPoll(backoffEpoch) {
		t.Fatalf("after one empty: ShouldPoll = false, want true (still below threshold)")
	}

	s = s.RecordResult(backoffEpoch.Add(1*time.Minute), 0)
	if s.ConsecutiveEmpty != 2 {
		t.Fatalf("after two empties: ConsecutiveEmpty = %d, want 2", s.ConsecutiveEmpty)
	}
	if !s.SuppressUntil.IsZero() {
		t.Fatalf("after two empties: SuppressUntil = %s, want zero", s.SuppressUntil)
	}
	if !s.ShouldPoll(backoffEpoch.Add(1 * time.Minute)) {
		t.Fatalf("after two empties: ShouldPoll = false, want true (still below threshold)")
	}
}

func TestBackoffState_RecordEmptyAtThreshold(t *testing.T) {
	s := newBackoffState()

	tickAt := backoffEpoch
	for i := 0; i < s.Threshold; i++ {
		s = s.RecordResult(tickAt, 0)
		tickAt = tickAt.Add(1 * time.Minute)
	}

	thresholdTick := backoffEpoch.Add(time.Duration(s.Threshold-1) * time.Minute)
	wantSuppress := thresholdTick.Add(s.Idle)
	if !s.SuppressUntil.Equal(wantSuppress) {
		t.Fatalf("at threshold: SuppressUntil = %s, want %s", s.SuppressUntil, wantSuppress)
	}
	if s.ConsecutiveEmpty != s.Threshold {
		t.Fatalf("at threshold: ConsecutiveEmpty = %d, want %d", s.ConsecutiveEmpty, s.Threshold)
	}

	if s.ShouldPoll(thresholdTick) {
		t.Fatalf("at threshold: ShouldPoll(now=thresholdTick) = true, want false (suppressed)")
	}
	if s.ShouldPoll(wantSuppress.Add(-1 * time.Nanosecond)) {
		t.Fatalf("just before SuppressUntil: ShouldPoll = true, want false")
	}
}

func TestBackoffState_RecordNonEmptyResetsState(t *testing.T) {
	s := newBackoffState()

	tickAt := backoffEpoch
	for i := 0; i < s.Threshold; i++ {
		s = s.RecordResult(tickAt, 0)
		tickAt = tickAt.Add(1 * time.Minute)
	}
	if s.ConsecutiveEmpty == 0 || s.SuppressUntil.IsZero() {
		t.Fatalf("precondition: state should be suppressed before reset, got %+v", s)
	}

	s = s.RecordResult(tickAt, 1)
	if s.ConsecutiveEmpty != 0 {
		t.Fatalf("after non-empty: ConsecutiveEmpty = %d, want 0", s.ConsecutiveEmpty)
	}
	if !s.SuppressUntil.IsZero() {
		t.Fatalf("after non-empty: SuppressUntil = %s, want zero", s.SuppressUntil)
	}
	if !s.ShouldPoll(tickAt) {
		t.Fatalf("after non-empty: ShouldPoll = false, want true")
	}
}

func TestBackoffState_ShouldPollAfterSuppressionElapsed(t *testing.T) {
	s := newBackoffState()

	tickAt := backoffEpoch
	for i := 0; i < s.Threshold; i++ {
		s = s.RecordResult(tickAt, 0)
		tickAt = tickAt.Add(1 * time.Minute)
	}

	if s.ShouldPoll(s.SuppressUntil.Add(-1 * time.Nanosecond)) {
		t.Fatalf("just before SuppressUntil: ShouldPoll = true, want false")
	}
	if !s.ShouldPoll(s.SuppressUntil) {
		t.Fatalf("at SuppressUntil: ShouldPoll = false, want true (suppression window elapsed)")
	}
	if !s.ShouldPoll(s.SuppressUntil.Add(1 * time.Hour)) {
		t.Fatalf("after SuppressUntil: ShouldPoll = false, want true")
	}
	if s.ConsecutiveEmpty != s.Threshold {
		t.Fatalf("ConsecutiveEmpty mutated by ShouldPoll: got %d, want %d", s.ConsecutiveEmpty, s.Threshold)
	}
}
