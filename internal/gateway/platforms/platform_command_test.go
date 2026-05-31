package platforms

import (
	"strings"
	"testing"
	"time"
)

func TestPlatformPauseAndResume(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	failures := map[string]PlatformFailure{
		"whatsapp": {Platform: "whatsapp", Attempts: 4, Retryable: true, NextRetry: now},
	}

	if PauseFailedPlatform(failures, "discord", "manual") {
		t.Fatalf("pausing an unqueued platform must return false")
	}
	if !PauseFailedPlatform(failures, "whatsapp", "manual pause") {
		t.Fatalf("pausing a queued platform must return true")
	}
	if f := failures["whatsapp"]; !f.Paused || f.PauseReason != "manual pause" {
		t.Fatalf("pause must set Paused + PauseReason, got %+v", f)
	}
	if !PauseFailedPlatform(failures, "whatsapp", "again") {
		t.Fatalf("pause must be idempotent (true on already-paused queued platform)")
	}

	if ResumePausedPlatform(failures, "discord", func() time.Time { return now }) {
		t.Fatalf("resuming an unqueued platform must return false")
	}
	if !ResumePausedPlatform(failures, "whatsapp", func() time.Time { return now }) {
		t.Fatalf("resuming a paused queued platform must return true")
	}
	f := failures["whatsapp"]
	if f.Paused || f.Attempts != 0 || !f.NextRetry.Equal(now) {
		t.Fatalf("resume must unpause, reset attempts, schedule immediate retry, got %+v", f)
	}
	if ResumePausedPlatform(failures, "whatsapp", func() time.Time { return now }) {
		t.Fatalf("resuming an already-retrying platform must return false")
	}
}

func TestPlatformCommandListPauseResumeUsage(t *testing.T) {
	bare := HandlePlatformCommand("/platform", []string{"telegram"}, nil)
	if !strings.Contains(bare, "Gateway platforms") || !strings.Contains(bare, "telegram") {
		t.Fatalf("bare /platform must default to list with connected platforms, got:\n%s", bare)
	}

	failures := map[string]PlatformFailure{
		"whatsapp": {Platform: "whatsapp", Attempts: 10, Retryable: true, Paused: true, PauseReason: "down"},
	}
	list := HandlePlatformCommand("/platform list", []string{"telegram"}, failures)
	if !strings.Contains(list, "whatsapp") || !strings.Contains(strings.ToUpper(list), "PAUSED") {
		t.Fatalf("/platform list must show paused platforms, got:\n%s", list)
	}

	if out := HandlePlatformCommand("/platform pause whatsapp", []string{"telegram"}, nil); !strings.Contains(out, "not in the retry queue") {
		t.Fatalf("/platform pause on an unqueued platform must reject, got: %s", out)
	}
	if out := HandlePlatformCommand("/platform pause notarealplatform", []string{"telegram"}, nil); !strings.Contains(out, "Unknown platform") {
		t.Fatalf("/platform pause on an unknown platform must reject, got: %s", out)
	}
	if out := HandlePlatformCommand("/platform pause telegram:ops", []string{"telegram"}, nil); !strings.Contains(out, "not in the retry queue") {
		t.Fatalf("/platform pause on an unqueued account-scoped platform must recognize the base platform, got: %s", out)
	}

	queued := map[string]PlatformFailure{
		"whatsapp": {Platform: "whatsapp", Attempts: 2, Retryable: true},
	}
	if out := HandlePlatformCommand("/platform pause whatsapp", []string{"telegram"}, queued); !strings.Contains(strings.ToLower(out), "paused") {
		t.Fatalf("/platform pause on a queued platform must confirm, got: %s", out)
	}
	if !queued["whatsapp"].Paused {
		t.Fatalf("/platform pause must actually pause the queued platform")
	}
	if out := HandlePlatformCommand("/platform resume whatsapp", []string{"telegram"}, queued); !strings.Contains(strings.ToLower(out), "resumed") {
		t.Fatalf("/platform resume must confirm, got: %s", out)
	}
	if queued["whatsapp"].Paused {
		t.Fatalf("/platform resume must unpause the platform")
	}

	accountQueued := map[string]PlatformFailure{
		"telegram:ops": {Platform: "telegram:ops", Attempts: 2, Retryable: true},
	}
	if out := HandlePlatformCommand("/platform pause telegram:ops", []string{"telegram"}, accountQueued); !strings.Contains(strings.ToLower(out), "paused") {
		t.Fatalf("/platform pause must support account-scoped queued platforms, got: %s", out)
	}
	if !accountQueued["telegram:ops"].Paused {
		t.Fatalf("/platform pause must pause account-scoped queued platforms")
	}
	if out := HandlePlatformCommand("/platform resume telegram:ops", []string{"telegram"}, accountQueued); !strings.Contains(strings.ToLower(out), "resumed") {
		t.Fatalf("/platform resume must support account-scoped queued platforms, got: %s", out)
	}
	if accountQueued["telegram:ops"].Paused {
		t.Fatalf("/platform resume must unpause account-scoped queued platforms")
	}

	if out := HandlePlatformCommand("/platform bogusaction", []string{"telegram"}, nil); !strings.Contains(out, "Usage:") {
		t.Fatalf("unknown action must return usage, got: %s", out)
	}
}
