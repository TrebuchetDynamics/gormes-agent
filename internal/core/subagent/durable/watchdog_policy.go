package durable

import "time"

// RSSWatchdogReason is the machine-readable policy vocabulary emitted before
// the RSS watchdog is wired into the durable worker loop.
type RSSWatchdogReason string

const (
	RSSWatchdogDisabled    RSSWatchdogReason = "rss_watchdog_disabled"
	RSSThresholdExceeded   RSSWatchdogReason = "rss_threshold_exceeded"
	RSSWatchdogUnavailable RSSWatchdogReason = "rss_watchdog_unavailable"
	RSSDrainStarted        RSSWatchdogReason = "rss_drain_started"
)

const bytesPerMiB = 1024 * 1024
const defaultWatchdogStableRunAfter = 5 * time.Minute

type WatchdogRestartReason string

const (
	StableWatchdogRestart WatchdogRestartReason = "stable_watchdog_restart"
	WatchdogRestartCrash  WatchdogRestartReason = "watchdog_restart"
)

// RSSReader reads RSS in bytes. Tests inject deterministic readers; runtime
// integration can later supply process-specific measurements.
type RSSReader func() (uint64, error)

// Clock supplies the observation timestamp for policy evidence.
type Clock func() time.Time

// RSSWatchdogPolicy is a value-only RSS watchdog configuration.
type RSSWatchdogPolicy struct {
	MaxRSSMB int64
}

// RSSWatchdogDecision is the pure policy result.
type RSSWatchdogDecision struct {
	Reason       RSSWatchdogReason
	RequestDrain bool
	Evidence     RSSWatchdogEvidence
}

// RSSWatchdogEvidence is suitable for later ledger/status wiring.
type RSSWatchdogEvidence struct {
	Reason     RSSWatchdogReason `json:"reason"`
	ObservedMB int64             `json:"observed_mb,omitempty"`
	MaxMB      int64             `json:"max_mb,omitempty"`
	CheckedAt  time.Time         `json:"checked_at,omitempty"`
	ErrorText  string            `json:"error,omitempty"`
}

// WatchdogRestartPolicy classifies supervised watchdog exits.
type WatchdogRestartPolicy struct {
	StableRunAfter time.Duration
}

// WatchdogRestartInput is the value-only restart observation.
type WatchdogRestartInput struct {
	StartedAt          time.Time
	ExitedAt           time.Time
	PreviousCrashCount int
	WatchdogExit       bool
}

// WatchdogRestartDecision is the crash-count policy result.
type WatchdogRestartDecision struct {
	Reason     WatchdogRestartReason
	CrashCount int
}

// Check classifies the RSS watchdog policy without touching worker runtime state.
func (p RSSWatchdogPolicy) Check(readRSS RSSReader, now Clock) RSSWatchdogDecision {
	if p.MaxRSSMB <= 0 {
		return RSSWatchdogDecision{
			Reason: RSSWatchdogDisabled,
			Evidence: RSSWatchdogEvidence{
				Reason: RSSWatchdogDisabled,
			},
		}
	}
	if readRSS == nil {
		return RSSWatchdogDecision{
			Reason: RSSWatchdogUnavailable,
			Evidence: RSSWatchdogEvidence{
				Reason:    RSSWatchdogUnavailable,
				CheckedAt: nowUTC(now),
				ErrorText: "rss reader is nil",
			},
		}
	}
	rssBytes, err := readRSS()
	if err != nil {
		return RSSWatchdogDecision{
			Reason: RSSWatchdogUnavailable,
			Evidence: RSSWatchdogEvidence{
				Reason:    RSSWatchdogUnavailable,
				CheckedAt: nowUTC(now),
				ErrorText: err.Error(),
			},
		}
	}
	observedMB := bytesToMB(rssBytes)
	if observedMB >= p.MaxRSSMB {
		return RSSWatchdogDecision{
			Reason:       RSSThresholdExceeded,
			RequestDrain: true,
			Evidence: RSSWatchdogEvidence{
				Reason:     RSSThresholdExceeded,
				ObservedMB: observedMB,
				MaxMB:      p.MaxRSSMB,
				CheckedAt:  nowUTC(now),
			},
		}
	}
	return RSSWatchdogDecision{}
}

// Classify resets watchdog restart accounting after a stable run.
func (p WatchdogRestartPolicy) Classify(input WatchdogRestartInput) WatchdogRestartDecision {
	if input.WatchdogExit && !input.StartedAt.IsZero() && !input.ExitedAt.IsZero() && input.ExitedAt.Sub(input.StartedAt) >= p.stableRunAfter() {
		return WatchdogRestartDecision{
			Reason:     StableWatchdogRestart,
			CrashCount: 1,
		}
	}
	return WatchdogRestartDecision{
		Reason:     WatchdogRestartCrash,
		CrashCount: input.PreviousCrashCount + 1,
	}
}

func bytesToMB(bytes uint64) int64 {
	mb := bytes / bytesPerMiB
	if bytes%bytesPerMiB >= bytesPerMiB/2 {
		mb++
	}
	return int64(mb)
}

func nowUTC(now Clock) time.Time {
	if now != nil {
		return now().UTC()
	}
	return time.Now().UTC()
}

func (p WatchdogRestartPolicy) stableRunAfter() time.Duration {
	if p.StableRunAfter > 0 {
		return p.StableRunAfter
	}
	return defaultWatchdogStableRunAfter
}
