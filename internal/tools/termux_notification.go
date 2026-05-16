package tools

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/redaction"
)

const (
	termuxNotificationCommand       = "termux-notification"
	defaultTermuxNotificationTitle  = "Gormes"
	defaultTermuxNotificationBody   = "Gormes notification"
	defaultTermuxNotificationRunErr = "termux-notification command failed"
)

const (
	defaultTermuxNotificationTitleRunes = 80
	defaultTermuxNotificationBodyRunes  = 240
)

// TermuxNotificationStatus is structured evidence for the optional
// Termux:API notification bridge.
type TermuxNotificationStatus string

const (
	TermuxNotificationStatusSkipped     TermuxNotificationStatus = "skipped"
	TermuxNotificationStatusAvailable   TermuxNotificationStatus = "available"
	TermuxNotificationStatusSent        TermuxNotificationStatus = "sent"
	TermuxNotificationStatusUnavailable TermuxNotificationStatus = "optional_notification_unavailable"
)

// TermuxNotificationResult is safe to print in doctor/status output. It never
// carries raw command stderr/stdout or unredacted notification text.
type TermuxNotificationResult struct {
	Status   TermuxNotificationStatus
	Command  string
	Message  string
	Redacted bool
}

// TermuxNotificationRunner is the fakeable exec seam for termux-notification.
type TermuxNotificationRunner func(context.Context, string, ...string) error

// TermuxNotificationSender sends optional Android notifications through
// Termux:API. Missing Termux or missing Termux:API is non-fatal.
type TermuxNotificationSender struct {
	Env      map[string]string
	LookPath func(string) (string, error)
	Run      TermuxNotificationRunner

	MaxTitleRunes int
	MaxBodyRunes  int
}

// Status reports whether the optional notification bridge is usable without
// invoking termux-notification.
func (s TermuxNotificationSender) Status(context.Context) TermuxNotificationResult {
	if !s.termux() {
		return TermuxNotificationResult{
			Status:  TermuxNotificationStatusSkipped,
			Message: "not running under Termux",
		}
	}
	command, err := s.commandPath()
	if err != nil {
		return TermuxNotificationResult{
			Status:  TermuxNotificationStatusUnavailable,
			Command: termuxNotificationCommand,
			Message: "termux-notification missing; install the Termux:API app and pkg install termux-api to enable Android notifications",
		}
	}
	return TermuxNotificationResult{
		Status:  TermuxNotificationStatusAvailable,
		Command: command,
		Message: "termux-notification available",
	}
}

// Notify sends a bounded, redacted Termux notification when the optional bridge
// is available. Degraded conditions return structured evidence and do not fail
// the caller.
func (s TermuxNotificationSender) Notify(ctx context.Context, title, body string) TermuxNotificationResult {
	if ctx == nil {
		ctx = context.Background()
	}
	status := s.Status(ctx)
	if status.Status != TermuxNotificationStatusAvailable {
		return status
	}

	safeTitle, titleChanged := sanitizeTermuxNotificationText(title, s.titleLimit(), defaultTermuxNotificationTitle)
	safeBody, bodyChanged := sanitizeTermuxNotificationText(body, s.bodyLimit(), defaultTermuxNotificationBody)
	run := s.Run
	if run == nil {
		run = defaultTermuxNotificationRunner
	}
	if err := run(ctx, status.Command, "--title", safeTitle, "--content", safeBody); err != nil {
		return TermuxNotificationResult{
			Status:  TermuxNotificationStatusUnavailable,
			Command: termuxNotificationCommand,
			Message: defaultTermuxNotificationRunErr,
		}
	}
	return TermuxNotificationResult{
		Status:   TermuxNotificationStatusSent,
		Command:  termuxNotificationCommand,
		Message:  "termux-notification sent",
		Redacted: titleChanged || bodyChanged,
	}
}

func (s TermuxNotificationSender) commandPath() (string, error) {
	lookPath := s.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	command, err := lookPath(termuxNotificationCommand)
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

func (s TermuxNotificationSender) termux() bool {
	if strings.TrimSpace(s.env("TERMUX_VERSION")) != "" {
		return true
	}
	for _, key := range []string{"PREFIX", "HOME"} {
		if strings.Contains(s.env(key), "com.termux/files") {
			return true
		}
	}
	return false
}

func (s TermuxNotificationSender) env(key string) string {
	if s.Env != nil {
		return s.Env[key]
	}
	return os.Getenv(key)
}

func (s TermuxNotificationSender) titleLimit() int {
	if s.MaxTitleRunes > 0 {
		return s.MaxTitleRunes
	}
	return defaultTermuxNotificationTitleRunes
}

func (s TermuxNotificationSender) bodyLimit() int {
	if s.MaxBodyRunes > 0 {
		return s.MaxBodyRunes
	}
	return defaultTermuxNotificationBodyRunes
}

func sanitizeTermuxNotificationText(text string, maxRunes int, fallback string) (string, bool) {
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

func defaultTermuxNotificationRunner(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	return cmd.Run()
}
