package tools

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/process"
)

const (
	// ProcessWatchMinInterval is the per-session minimum spacing for emitted
	// watch-pattern notifications.
	ProcessWatchMinInterval = process.ProcessWatchMinInterval
	// ProcessWatchStrikeLimit is the number of consecutive cooldown windows with
	// suppressed matches before watch patterns are disabled.
	ProcessWatchStrikeLimit = process.ProcessWatchStrikeLimit
	// ProcessWatchGlobalMaxPerWindow is the maximum number of watch matches
	// emitted across all sessions in ProcessWatchGlobalWindow.
	ProcessWatchGlobalMaxPerWindow = process.ProcessWatchGlobalMaxPerWindow
	// ProcessWatchGlobalWindow is the global overflow accounting window.
	ProcessWatchGlobalWindow = process.ProcessWatchGlobalWindow
	// ProcessWatchGlobalCooldown is the global overflow suppression duration.
	ProcessWatchGlobalCooldown = process.ProcessWatchGlobalCooldown
)

// ProcessNotificationRequest is the process/terminal notification portion of a
// future spawn request.
type ProcessNotificationRequest = process.ProcessNotificationRequest

// ProcessNotificationEvidence is an operator-readable policy decision.
type ProcessNotificationEvidence = process.ProcessNotificationEvidence

// ProcessNotificationPlan is the normalized notification configuration that a
// future process runner can apply before spawning a live process.
type ProcessNotificationPlan = process.ProcessNotificationPlan

// ProcessNotificationSession is the notification state for one background
// process session. It intentionally does not own a live process handle.
type ProcessNotificationSession = process.ProcessNotificationSession

// ProcessNotificationEvent is a queued notification or degraded-mode summary
// produced by the policy.
type ProcessNotificationEvent = process.ProcessNotificationEvent

// ProcessNotificationPolicy applies watch-pattern notification throttles.
type ProcessNotificationPolicy = process.ProcessNotificationPolicy

// NormalizeProcessNotificationRequest resolves invalid notification flag
// combinations without starting a process.
func NormalizeProcessNotificationRequest(req ProcessNotificationRequest) ProcessNotificationPlan {
	return process.NormalizeProcessNotificationRequest(req)
}

// NewProcessNotificationPolicy returns a process notification policy. Tests can
// pass a fake clock; production callers may pass nil to use time.Now.
func NewProcessNotificationPolicy(now func() time.Time) *ProcessNotificationPolicy {
	return process.NewProcessNotificationPolicy(now)
}
