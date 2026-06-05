package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/termux"

const (
	defaultTermuxNotificationTitleRunes = termux.DefaultNotificationTitleRunes
	defaultTermuxNotificationBodyRunes  = termux.DefaultNotificationBodyRunes
)

// TermuxNotificationStatus is structured evidence for the optional
// Termux:API notification bridge.
type TermuxNotificationStatus = termux.NotificationStatus

const (
	TermuxNotificationStatusSkipped     = termux.NotificationStatusSkipped
	TermuxNotificationStatusAvailable   = termux.NotificationStatusAvailable
	TermuxNotificationStatusSent        = termux.NotificationStatusSent
	TermuxNotificationStatusUnavailable = termux.NotificationStatusUnavailable
)

// TermuxNotificationResult is safe to print in doctor/status output. It never
// carries raw command stderr/stdout or unredacted notification text.
type TermuxNotificationResult = termux.NotificationResult

// TermuxNotificationRunner is the fakeable exec seam for termux-notification.
type TermuxNotificationRunner = termux.NotificationRunner

// TermuxNotificationSender sends optional Android notifications through
// Termux:API. Missing Termux or missing Termux:API is non-fatal.
type TermuxNotificationSender = termux.NotificationSender
