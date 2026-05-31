package process

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/process/notifications"
)

const (
	// ProcessWatchMinInterval is the per-session minimum spacing for emitted
	// watch-pattern notifications.
	ProcessWatchMinInterval = notifications.ProcessWatchMinInterval
	// ProcessWatchStrikeLimit is the number of consecutive cooldown windows with
	// suppressed matches before watch patterns are disabled.
	ProcessWatchStrikeLimit = notifications.ProcessWatchStrikeLimit
	// ProcessWatchGlobalMaxPerWindow is the maximum number of watch matches
	// emitted across all sessions in ProcessWatchGlobalWindow.
	ProcessWatchGlobalMaxPerWindow = notifications.ProcessWatchGlobalMaxPerWindow
	// ProcessWatchGlobalWindow is the global overflow accounting window.
	ProcessWatchGlobalWindow = notifications.ProcessWatchGlobalWindow
	// ProcessWatchGlobalCooldown is the global overflow suppression duration.
	ProcessWatchGlobalCooldown = notifications.ProcessWatchGlobalCooldown
)

// ProcessNotificationRequest is the process/terminal notification portion of a
// future spawn request.
type ProcessNotificationRequest = notifications.ProcessNotificationRequest

// ProcessNotificationEvidence is an operator-readable policy decision.
type ProcessNotificationEvidence = notifications.ProcessNotificationEvidence

// ProcessNotificationPlan is the normalized notification configuration that a
// future process runner can apply before spawning a live process.
type ProcessNotificationPlan = notifications.ProcessNotificationPlan

// ProcessNotificationSession is the notification state for one background
// process session. It intentionally does not own a live process handle.
type ProcessNotificationSession = notifications.ProcessNotificationSession

// ProcessNotificationEvent is a queued notification or degraded-mode summary
// produced by the policy.
type ProcessNotificationEvent = notifications.ProcessNotificationEvent

// ProcessNotificationPolicy applies watch-pattern notification throttles.
type ProcessNotificationPolicy = notifications.ProcessNotificationPolicy

// NormalizeProcessNotificationRequest resolves invalid notification flag
// combinations without starting a process.
func NormalizeProcessNotificationRequest(req ProcessNotificationRequest) ProcessNotificationPlan {
	return notifications.NormalizeProcessNotificationRequest(req)
}

// NewProcessNotificationPolicy returns a process notification policy. Tests can
// pass a fake clock; production callers may pass nil to use time.Now.
func NewProcessNotificationPolicy(now func() time.Time) *ProcessNotificationPolicy {
	return notifications.NewProcessNotificationPolicy(now)
}
