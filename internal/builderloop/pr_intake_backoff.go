package builderloop

import "time"

// BackoffState is a pure state machine for the PR-intake idle backoff.
// The caller injects the clock (now time.Time) and the listed result count;
// the helper does no GitHub I/O and starts no goroutines.
//
// After Threshold consecutive listed=0 results, ShouldPoll returns false
// until now >= SuppressUntil. Any listed > 0 result resets ConsecutiveEmpty
// and clears SuppressUntil so the next poll fires at Baseline cadence again.
//
// Baseline is carried so callers can pick the next interval (Baseline when
// active, Idle when suppressed) without keeping the value in a second place.
type BackoffState struct {
	ConsecutiveEmpty int
	SuppressUntil    time.Time
	Threshold        int
	Baseline         time.Duration
	Idle             time.Duration
}

// ShouldPoll reports whether the caller should run pr_intake at now. It is
// false only while a suppression window is active (now < SuppressUntil).
func (s BackoffState) ShouldPoll(now time.Time) bool {
	if s.SuppressUntil.IsZero() {
		return true
	}
	return !now.Before(s.SuppressUntil)
}

// RecordResult folds the outcome of a single pr_intake into the state and
// returns the next BackoffState. listed > 0 resets the counter; listed == 0
// increments it and arms a SuppressUntil window once Threshold is reached.
func (s BackoffState) RecordResult(now time.Time, listed int) BackoffState {
	if listed > 0 {
		s.ConsecutiveEmpty = 0
		s.SuppressUntil = time.Time{}
		return s
	}
	s.ConsecutiveEmpty++
	if s.Threshold > 0 && s.ConsecutiveEmpty >= s.Threshold {
		s.SuppressUntil = now.Add(s.Idle)
	}
	return s
}
