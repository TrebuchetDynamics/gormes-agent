package gateway

import (
	"strings"
	"testing"
	"time"
)

func TestRenderStatusSummary_NoChannelsConfigured(t *testing.T) {
	var input ConfiguredChannels

	got := RenderStatusSummary(input)

	if !strings.Contains(got, "no channels configured") {
		t.Fatalf("RenderStatusSummary empty input = %q, want mention of 'no channels configured'", got)
	}
	if strings.Contains(got, "telegram:") || strings.Contains(got, "discord:") {
		t.Fatalf("RenderStatusSummary empty input must not list channels, got: %q", got)
	}
}

func TestRenderStatusSummary_TelegramConfigured_Unpaired(t *testing.T) {
	input := ConfiguredChannels{
		Telegram: &ConfiguredChannel{
			Platform:          "telegram",
			AllowedChatID:     "12345",
			FirstRunDiscovery: true,
		},
	}

	got := RenderStatusSummary(input)

	if !strings.Contains(got, "telegram:") {
		t.Fatalf("want telegram row, got: %q", got)
	}
	if !strings.Contains(got, "configured") {
		t.Fatalf("want 'configured' marker for telegram, got: %q", got)
	}
	if !strings.Contains(got, "allowed_chat_id=12345") {
		t.Fatalf("want allowed_chat_id rendered, got: %q", got)
	}
	if !strings.Contains(got, "first_run_discovery=true") {
		t.Fatalf("want first_run_discovery rendered, got: %q", got)
	}
	if !strings.Contains(got, "paired=unknown") {
		t.Fatalf("unpaired telegram (no runtime status) should render 'paired=unknown', got: %q", got)
	}
	if !strings.Contains(got, "lifecycle=unknown") {
		t.Fatalf("no runtime status should render 'lifecycle=unknown', got: %q", got)
	}
}

func TestRenderStatusSummary_DiscordConfigured_WithRuntimeStatus(t *testing.T) {
	sm := NewStatusModelWithClock(func() time.Time {
		return time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	})
	sm.MarkRunning("discord")

	input := ConfiguredChannels{
		Discord: &ConfiguredChannel{
			Platform:          "discord",
			AllowedChatID:     "987654321",
			FirstRunDiscovery: false,
		},
		Runtime: sm,
	}

	got := RenderStatusSummary(input)

	if !strings.Contains(got, "discord:") {
		t.Fatalf("want discord row, got: %q", got)
	}
	if !strings.Contains(got, "allowed_channel_id=987654321") {
		t.Fatalf("want allowed_channel_id rendered for discord (not allowed_chat_id), got: %q", got)
	}
	if !strings.Contains(got, "lifecycle=running") {
		t.Fatalf("want 'lifecycle=running' from runtime status model, got: %q", got)
	}
}

func TestRenderStatusSummary_DeterministicOrdering(t *testing.T) {
	input := ConfiguredChannels{
		Telegram: &ConfiguredChannel{Platform: "telegram"},
		Discord:  &ConfiguredChannel{Platform: "discord"},
	}

	got := RenderStatusSummary(input)

	// Deterministic platform ordering: discord then telegram (alphabetical).
	discordIdx := strings.Index(got, "discord:")
	telegramIdx := strings.Index(got, "telegram:")
	if discordIdx < 0 || telegramIdx < 0 {
		t.Fatalf("want both channels rendered, got: %q", got)
	}
	if discordIdx > telegramIdx {
		t.Fatalf("want deterministic alphabetical ordering (discord before telegram), got: %q", got)
	}
}

func TestRenderStatusSummary_LifecycleFailedExposesLastError(t *testing.T) {
	sm := NewStatusModel()
	sm.MarkFailed("telegram", errBoom)

	input := ConfiguredChannels{
		Telegram: &ConfiguredChannel{Platform: "telegram"},
		Runtime:  sm,
	}

	got := RenderStatusSummary(input)

	if !strings.Contains(got, "lifecycle=failed") {
		t.Fatalf("want lifecycle=failed, got: %q", got)
	}
	if !strings.Contains(got, errBoom.Error()) {
		t.Fatalf("want last error %q rendered, got: %q", errBoom.Error(), got)
	}
}

// errBoom is a sentinel used in status-view tests to exercise the failed
// path without pulling in a full adapter wiring.
var errBoom = renderTestErr{msg: "adapter exploded"}

type renderTestErr struct{ msg string }

func (e renderTestErr) Error() string { return e.msg }
