package termux

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

const (
	notificationCommand       = "termux-notification"
	defaultNotificationTitle  = "Gormes"
	defaultNotificationBody   = "Gormes notification"
	defaultNotificationRunErr = "termux-notification command failed"
)

const (
	DefaultNotificationTitleRunes = 80
	DefaultNotificationBodyRunes  = 240
)

// NotificationStatus is structured evidence for the optional Termux:API
// notification bridge.
type NotificationStatus string

const (
	NotificationStatusSkipped     NotificationStatus = "skipped"
	NotificationStatusAvailable   NotificationStatus = "available"
	NotificationStatusSent        NotificationStatus = "sent"
	NotificationStatusUnavailable NotificationStatus = "optional_notification_unavailable"
)

// NotificationResult is safe to print in doctor/status output. It never
// carries raw command stderr/stdout or unredacted notification text.
type NotificationResult struct {
	Status   NotificationStatus
	Command  string
	Message  string
	Redacted bool
}

// NotificationRunner is the fakeable exec seam for termux-notification.
type NotificationRunner func(context.Context, string, ...string) error

// NotificationSender sends optional Android notifications through Termux:API.
// Missing Termux or missing Termux:API is non-fatal.
type NotificationSender struct {
	Env      map[string]string
	LookPath func(string) (string, error)
	Run      NotificationRunner

	MaxTitleRunes int
	MaxBodyRunes  int
}

// Status reports whether the optional notification bridge is usable without
// invoking termux-notification.
func (s NotificationSender) Status(context.Context) NotificationResult {
	if !s.termux() {
		return NotificationResult{
			Status:  NotificationStatusSkipped,
			Message: "not running under Termux",
		}
	}
	command, err := s.commandPath()
	if err != nil {
		return NotificationResult{
			Status:  NotificationStatusUnavailable,
			Command: notificationCommand,
			Message: "termux-notification missing; install the Termux:API app and pkg install termux-api to enable Android notifications",
		}
	}
	return NotificationResult{
		Status:  NotificationStatusAvailable,
		Command: command,
		Message: "termux-notification available",
	}
}

// Notify sends a bounded, redacted Termux notification when the optional bridge
// is available. Degraded conditions return structured evidence and do not fail
// the caller.
func (s NotificationSender) Notify(ctx context.Context, title, body string) NotificationResult {
	if ctx == nil {
		ctx = context.Background()
	}
	status := s.Status(ctx)
	if status.Status != NotificationStatusAvailable {
		return status
	}

	safeTitle, titleChanged := sanitizeNotificationText(title, s.titleLimit(), defaultNotificationTitle)
	safeBody, bodyChanged := sanitizeNotificationText(body, s.bodyLimit(), defaultNotificationBody)
	run := s.Run
	if run == nil {
		run = defaultNotificationRunner
	}
	if err := run(ctx, status.Command, "--title", safeTitle, "--content", safeBody); err != nil {
		return NotificationResult{
			Status:  NotificationStatusUnavailable,
			Command: notificationCommand,
			Message: defaultNotificationRunErr,
		}
	}
	return NotificationResult{
		Status:   NotificationStatusSent,
		Command:  notificationCommand,
		Message:  "termux-notification sent",
		Redacted: titleChanged || bodyChanged,
	}
}

func (s NotificationSender) commandPath() (string, error) {
	lookPath := s.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	command, err := lookPath(notificationCommand)
	if err == nil && strings.TrimSpace(command) != "" {
		return command, nil
	}
	if err == nil {
		return "", exec.ErrNotFound
	}
	var execErr *exec.Error
	if errors.As(err, &execErr) || errors.Is(err, exec.ErrNotFound) {
		return "", err
	}
	return "", err
}

func (s NotificationSender) termux() bool {
	return IsEnvironment(s.env)
}

func (s NotificationSender) env(key string) string {
	if s.Env != nil {
		return s.Env[key]
	}
	return os.Getenv(key)
}

func (s NotificationSender) titleLimit() int {
	if s.MaxTitleRunes > 0 {
		return s.MaxTitleRunes
	}
	return DefaultNotificationTitleRunes
}

func (s NotificationSender) bodyLimit() int {
	if s.MaxBodyRunes > 0 {
		return s.MaxBodyRunes
	}
	return DefaultNotificationBodyRunes
}

func sanitizeNotificationText(text string, maxRunes int, fallback string) (string, bool) {
	clean := strings.TrimSpace(text)
	if clean == "" {
		clean = fallback
	}
	redacted, count := redaction.RedactSecretsWithCount(clean, "[redacted]")
	truncated := false
	if maxRunes > 0 {
		redacted, truncated = truncateRunes(redacted, maxRunes)
	}
	return redacted, count > 0 || truncated
}

func truncateRunes(text string, maxRunes int) (string, bool) {
	runes := []rune(text)
	if maxRunes <= 0 || len(runes) <= maxRunes {
		return text, false
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes]), true
	}
	return string(runes[:maxRunes-3]) + "...", true
}

func defaultNotificationRunner(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	return cmd.Run()
}
