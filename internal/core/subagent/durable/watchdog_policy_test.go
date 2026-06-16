package durable

import (
	"errors"
	"testing"
	"time"
)

func TestRSSWatchdogPolicy_NilRSSReaderUnavailable(t *testing.T) {
	checkedAt := time.Date(2026, 4, 27, 5, 45, 0, 0, time.UTC)

	decision := RSSWatchdogPolicy{MaxRSSMB: 100}.Check(nil, func() time.Time {
		return checkedAt
	})

	if decision.Reason != RSSWatchdogUnavailable {
		t.Fatalf("Reason = %q, want %q", decision.Reason, RSSWatchdogUnavailable)
	}
	if decision.RequestDrain {
		t.Fatal("RequestDrain = true, want false when RSS reader is nil")
	}
	if decision.Evidence.Reason != RSSWatchdogUnavailable {
		t.Fatalf("Evidence.Reason = %q, want %q", decision.Evidence.Reason, RSSWatchdogUnavailable)
	}
	if decision.Evidence.ErrorText != "rss reader is nil" {
		t.Fatalf("Evidence.ErrorText = %q, want rss reader is nil", decision.Evidence.ErrorText)
	}
	if !decision.Evidence.CheckedAt.Equal(checkedAt) {
		t.Fatalf("Evidence.CheckedAt = %s, want %s", decision.Evidence.CheckedAt, checkedAt)
	}
}

func TestRSSWatchdogPolicy_RSSReadFailure(t *testing.T) {
	checkedAt := time.Date(2026, 4, 27, 5, 30, 0, 0, time.UTC)

	decision := RSSWatchdogPolicy{MaxRSSMB: 100}.Check(
		func() (uint64, error) {
			return 0, errors.New("rss unavailable")
		},
		func() time.Time {
			return checkedAt
		},
	)

	if decision.Reason != RSSWatchdogUnavailable {
		t.Fatalf("Reason = %q, want %q", decision.Reason, RSSWatchdogUnavailable)
	}
	if decision.RequestDrain {
		t.Fatal("RequestDrain = true, want false when RSS read fails")
	}
	if decision.Evidence.Reason != RSSWatchdogUnavailable {
		t.Fatalf("Evidence.Reason = %q, want %q", decision.Evidence.Reason, RSSWatchdogUnavailable)
	}
	if decision.Evidence.ErrorText != "rss unavailable" {
		t.Fatalf("Evidence.ErrorText = %q, want rss unavailable", decision.Evidence.ErrorText)
	}
	if !decision.Evidence.CheckedAt.Equal(checkedAt) {
		t.Fatalf("Evidence.CheckedAt = %s, want %s", decision.Evidence.CheckedAt, checkedAt)
	}
}

func TestWatchdogRestartPolicy_StableRunReset(t *testing.T) {
	startedAt := time.Date(2026, 4, 27, 6, 0, 0, 0, time.UTC)

	decision := WatchdogRestartPolicy{StableRunAfter: 5 * time.Minute}.Classify(
		WatchdogRestartInput{
			StartedAt:          startedAt,
			ExitedAt:           startedAt.Add(5 * time.Minute),
			PreviousCrashCount: 4,
			WatchdogExit:       true,
		},
	)

	if decision.Reason != StableWatchdogRestart {
		t.Fatalf("Reason = %q, want %q", decision.Reason, StableWatchdogRestart)
	}
	if decision.CrashCount != 1 {
		t.Fatalf("CrashCount = %d, want reset to 1 after a stable watchdog exit", decision.CrashCount)
	}
}

func TestRSSWatchdogPolicy_ThresholdExceeded(t *testing.T) {
	checkedAt := time.Date(2026, 4, 27, 5, 15, 0, 0, time.UTC)

	decision := RSSWatchdogPolicy{MaxRSSMB: 100}.Check(
		func() (uint64, error) {
			return 151 * 1024 * 1024, nil
		},
		func() time.Time {
			return checkedAt
		},
	)

	if decision.Reason != RSSThresholdExceeded {
		t.Fatalf("Reason = %q, want %q", decision.Reason, RSSThresholdExceeded)
	}
	if !decision.RequestDrain {
		t.Fatal("RequestDrain = false, want true when RSS threshold is exceeded")
	}
	if decision.Evidence.Reason != RSSThresholdExceeded {
		t.Fatalf("Evidence.Reason = %q, want %q", decision.Evidence.Reason, RSSThresholdExceeded)
	}
	if decision.Evidence.ObservedMB != 151 {
		t.Fatalf("Evidence.ObservedMB = %d, want 151", decision.Evidence.ObservedMB)
	}
	if decision.Evidence.MaxMB != 100 {
		t.Fatalf("Evidence.MaxMB = %d, want 100", decision.Evidence.MaxMB)
	}
	if !decision.Evidence.CheckedAt.Equal(checkedAt) {
		t.Fatalf("Evidence.CheckedAt = %s, want %s", decision.Evidence.CheckedAt, checkedAt)
	}
}

func TestRSSWatchdogPolicy_DisabledAtZero(t *testing.T) {
	readCount := 0
	decision := RSSWatchdogPolicy{MaxRSSMB: 0}.Check(
		func() (uint64, error) {
			readCount++
			return 999 * 1024 * 1024, nil
		},
		func() time.Time {
			return time.Date(2026, 4, 27, 5, 0, 0, 0, time.UTC)
		},
	)

	if readCount != 0 {
		t.Fatalf("RSS read count = %d, want 0 when max_rss_mb=0", readCount)
	}
	if decision.Reason != RSSWatchdogDisabled {
		t.Fatalf("Reason = %q, want %q", decision.Reason, RSSWatchdogDisabled)
	}
	if decision.RequestDrain {
		t.Fatal("RequestDrain = true, want false when watchdog is disabled")
	}
	if decision.Evidence.Reason != RSSWatchdogDisabled {
		t.Fatalf("Evidence.Reason = %q, want %q", decision.Evidence.Reason, RSSWatchdogDisabled)
	}
}
