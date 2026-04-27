package subagent

import "time"

// DurableWorkerRSSWatchdogReason is the machine-readable policy vocabulary
// emitted before the RSS watchdog is wired into the durable worker loop.
type DurableWorkerRSSWatchdogReason string

const (
	DurableWorkerRSSWatchdogDisabled    DurableWorkerRSSWatchdogReason = "rss_watchdog_disabled"
	DurableWorkerRSSThresholdExceeded   DurableWorkerRSSWatchdogReason = "rss_threshold_exceeded"
	DurableWorkerRSSWatchdogUnavailable DurableWorkerRSSWatchdogReason = "rss_watchdog_unavailable"
)

const durableWorkerBytesPerMiB = 1024 * 1024
const defaultDurableWorkerWatchdogStableRunAfter = 5 * time.Minute

type DurableWorkerWatchdogRestartReason string

const (
	DurableWorkerStableWatchdogRestart DurableWorkerWatchdogRestartReason = "stable_watchdog_restart"
	DurableWorkerWatchdogRestartCrash  DurableWorkerWatchdogRestartReason = "watchdog_restart"
)

// DurableWorkerRSSReader reads RSS in bytes. Tests inject deterministic readers;
// runtime integration can later supply process-specific measurements.
type DurableWorkerRSSReader func() (uint64, error)

// DurableWorkerClock supplies the observation timestamp for policy evidence.
type DurableWorkerClock func() time.Time

// DurableWorkerRSSWatchdogPolicy is a value-only RSS watchdog configuration.
type DurableWorkerRSSWatchdogPolicy struct {
	MaxRSSMB int64
}

// DurableWorkerRSSWatchdogDecision is the pure policy result.
type DurableWorkerRSSWatchdogDecision struct {
	Reason       DurableWorkerRSSWatchdogReason
	RequestDrain bool
	Evidence     DurableWorkerRSSWatchdogEvidence
}

// DurableWorkerRSSWatchdogEvidence is suitable for later ledger/status wiring.
type DurableWorkerRSSWatchdogEvidence struct {
	Reason     DurableWorkerRSSWatchdogReason `json:"reason"`
	ObservedMB int64                          `json:"observed_mb,omitempty"`
	MaxMB      int64                          `json:"max_mb,omitempty"`
	CheckedAt  time.Time                      `json:"checked_at,omitempty"`
	ErrorText  string                         `json:"error,omitempty"`
}

// DurableWorkerWatchdogRestartPolicy classifies supervised watchdog exits.
type DurableWorkerWatchdogRestartPolicy struct {
	StableRunAfter time.Duration
}

// DurableWorkerWatchdogRestartInput is the value-only restart observation.
type DurableWorkerWatchdogRestartInput struct {
	StartedAt          time.Time
	ExitedAt           time.Time
	PreviousCrashCount int
	WatchdogExit       bool
}

// DurableWorkerWatchdogRestartDecision is the crash-count policy result.
type DurableWorkerWatchdogRestartDecision struct {
	Reason     DurableWorkerWatchdogRestartReason
	CrashCount int
}

// Check classifies the RSS watchdog policy without touching worker runtime state.
func (p DurableWorkerRSSWatchdogPolicy) Check(readRSS DurableWorkerRSSReader, now DurableWorkerClock) DurableWorkerRSSWatchdogDecision {
	if p.MaxRSSMB <= 0 {
		return DurableWorkerRSSWatchdogDecision{
			Reason: DurableWorkerRSSWatchdogDisabled,
			Evidence: DurableWorkerRSSWatchdogEvidence{
				Reason: DurableWorkerRSSWatchdogDisabled,
			},
		}
	}
	rssBytes, err := readRSS()
	if err != nil {
		return DurableWorkerRSSWatchdogDecision{
			Reason: DurableWorkerRSSWatchdogUnavailable,
			Evidence: DurableWorkerRSSWatchdogEvidence{
				Reason:    DurableWorkerRSSWatchdogUnavailable,
				CheckedAt: durableWorkerRSSNow(now),
				ErrorText: err.Error(),
			},
		}
	}
	observedMB := durableWorkerRSSBytesToMB(rssBytes)
	if observedMB >= p.MaxRSSMB {
		return DurableWorkerRSSWatchdogDecision{
			Reason:       DurableWorkerRSSThresholdExceeded,
			RequestDrain: true,
			Evidence: DurableWorkerRSSWatchdogEvidence{
				Reason:     DurableWorkerRSSThresholdExceeded,
				ObservedMB: observedMB,
				MaxMB:      p.MaxRSSMB,
				CheckedAt:  durableWorkerRSSNow(now),
			},
		}
	}
	return DurableWorkerRSSWatchdogDecision{}
}

// Classify resets watchdog restart accounting after a stable run.
func (p DurableWorkerWatchdogRestartPolicy) Classify(input DurableWorkerWatchdogRestartInput) DurableWorkerWatchdogRestartDecision {
	if input.WatchdogExit && !input.StartedAt.IsZero() && !input.ExitedAt.IsZero() && input.ExitedAt.Sub(input.StartedAt) >= p.stableRunAfter() {
		return DurableWorkerWatchdogRestartDecision{
			Reason:     DurableWorkerStableWatchdogRestart,
			CrashCount: 1,
		}
	}
	return DurableWorkerWatchdogRestartDecision{
		Reason:     DurableWorkerWatchdogRestartCrash,
		CrashCount: input.PreviousCrashCount + 1,
	}
}

func durableWorkerRSSBytesToMB(bytes uint64) int64 {
	mb := bytes / durableWorkerBytesPerMiB
	if bytes%durableWorkerBytesPerMiB >= durableWorkerBytesPerMiB/2 {
		mb++
	}
	return int64(mb)
}

func durableWorkerRSSNow(now DurableWorkerClock) time.Time {
	if now != nil {
		return now().UTC()
	}
	return time.Now().UTC()
}

func (p DurableWorkerWatchdogRestartPolicy) stableRunAfter() time.Duration {
	if p.StableRunAfter > 0 {
		return p.StableRunAfter
	}
	return defaultDurableWorkerWatchdogStableRunAfter
}
