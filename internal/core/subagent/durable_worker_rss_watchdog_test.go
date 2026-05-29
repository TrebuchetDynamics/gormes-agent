package subagent

import (
	"errors"
	"testing"
	"time"
)

func TestDurableWorkerRSSWatchdog_RSSReadFailure(t *testing.T) {
	checkedAt := time.Date(2026, 4, 27, 5, 30, 0, 0, time.UTC)

	decision := DurableWorkerRSSWatchdogPolicy{MaxRSSMB: 100}.Check(
		func() (uint64, error) {
			return 0, errors.New("rss unavailable")
		},
		func() time.Time {
			return checkedAt
		},
	)

	if decision.Reason != DurableWorkerRSSWatchdogUnavailable {
		t.Fatalf("Reason = %q, want %q", decision.Reason, DurableWorkerRSSWatchdogUnavailable)
	}
	if decision.RequestDrain {
		t.Fatal("RequestDrain = true, want false when RSS read fails")
	}
	if decision.Evidence.Reason != DurableWorkerRSSWatchdogUnavailable {
		t.Fatalf("Evidence.Reason = %q, want %q", decision.Evidence.Reason, DurableWorkerRSSWatchdogUnavailable)
	}
	if decision.Evidence.ErrorText != "rss unavailable" {
		t.Fatalf("Evidence.ErrorText = %q, want rss unavailable", decision.Evidence.ErrorText)
	}
	if !decision.Evidence.CheckedAt.Equal(checkedAt) {
		t.Fatalf("Evidence.CheckedAt = %s, want %s", decision.Evidence.CheckedAt, checkedAt)
	}
}

func TestDurableWorkerWatchdogRestartPolicy_StableRunReset(t *testing.T) {
	startedAt := time.Date(2026, 4, 27, 6, 0, 0, 0, time.UTC)

	decision := DurableWorkerWatchdogRestartPolicy{StableRunAfter: 5 * time.Minute}.Classify(
		DurableWorkerWatchdogRestartInput{
			StartedAt:          startedAt,
			ExitedAt:           startedAt.Add(5 * time.Minute),
			PreviousCrashCount: 4,
			WatchdogExit:       true,
		},
	)

	if decision.Reason != DurableWorkerStableWatchdogRestart {
		t.Fatalf("Reason = %q, want %q", decision.Reason, DurableWorkerStableWatchdogRestart)
	}
	if decision.CrashCount != 1 {
		t.Fatalf("CrashCount = %d, want reset to 1 after a stable watchdog exit", decision.CrashCount)
	}
}

func TestDurableWorkerRSSWatchdog_ThresholdExceeded(t *testing.T) {
	checkedAt := time.Date(2026, 4, 27, 5, 15, 0, 0, time.UTC)

	decision := DurableWorkerRSSWatchdogPolicy{MaxRSSMB: 100}.Check(
		func() (uint64, error) {
			return 151 * 1024 * 1024, nil
		},
		func() time.Time {
			return checkedAt
		},
	)

	if decision.Reason != DurableWorkerRSSThresholdExceeded {
		t.Fatalf("Reason = %q, want %q", decision.Reason, DurableWorkerRSSThresholdExceeded)
	}
	if !decision.RequestDrain {
		t.Fatal("RequestDrain = false, want true when RSS threshold is exceeded")
	}
	if decision.Evidence.Reason != DurableWorkerRSSThresholdExceeded {
		t.Fatalf("Evidence.Reason = %q, want %q", decision.Evidence.Reason, DurableWorkerRSSThresholdExceeded)
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

func TestDurableWorkerRSSWatchdog_DisabledAtZero(t *testing.T) {
	readCount := 0
	decision := DurableWorkerRSSWatchdogPolicy{MaxRSSMB: 0}.Check(
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
	if decision.Reason != DurableWorkerRSSWatchdogDisabled {
		t.Fatalf("Reason = %q, want %q", decision.Reason, DurableWorkerRSSWatchdogDisabled)
	}
	if decision.RequestDrain {
		t.Fatal("RequestDrain = true, want false when watchdog is disabled")
	}
	if decision.Evidence.Reason != DurableWorkerRSSWatchdogDisabled {
		t.Fatalf("Evidence.Reason = %q, want %q", decision.Evidence.Reason, DurableWorkerRSSWatchdogDisabled)
	}
}
