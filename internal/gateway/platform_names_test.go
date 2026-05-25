package gateway

import "testing"

func TestAccountScopedPlatformNames(t *testing.T) {
	tests := []struct {
		platform string
		telegram bool
		discord  bool
		slack    bool
		base     string
	}{
		{platform: "telegram", telegram: true, base: "telegram"},
		{platform: "Telegram:ops", telegram: true, base: "telegram"},
		{platform: " telegram:support ", telegram: true, base: "telegram"},
		{platform: "discord", discord: true, base: "discord"},
		{platform: "Discord:ops", discord: true, base: "discord"},
		{platform: "slack", slack: true, base: "slack"},
		{platform: "Slack:team-a", slack: true, base: "slack"},
		{platform: "telegramish", base: "telegramish"},
	}
	for _, tt := range tests {
		if got := isTelegramPlatform(tt.platform); got != tt.telegram {
			t.Fatalf("isTelegramPlatform(%q) = %v, want %v", tt.platform, got, tt.telegram)
		}
		if got := isDiscordPlatform(tt.platform); got != tt.discord {
			t.Fatalf("isDiscordPlatform(%q) = %v, want %v", tt.platform, got, tt.discord)
		}
		if got := isSlackPlatform(tt.platform); got != tt.slack {
			t.Fatalf("isSlackPlatform(%q) = %v, want %v", tt.platform, got, tt.slack)
		}
		if got := platformBaseName(tt.platform); got != tt.base {
			t.Fatalf("platformBaseName(%q) = %q, want %q", tt.platform, got, tt.base)
		}
	}
}

func TestTelegramDMTopicReplyFallbackLaneIncludesAccounts(t *testing.T) {
	if !telegramDMTopicReplyFallbackLane("telegram:ops", "12345", "67890") {
		t.Fatal("expected account-scoped Telegram platform to use DM topic reply fallback")
	}
	if telegramDMTopicReplyFallbackLane("telegram:ops", "-10012345", "67890") {
		t.Fatal("did not expect group chat IDs to use DM topic reply fallback")
	}
	if telegramDMTopicReplyFallbackLane("discord", "12345", "67890") {
		t.Fatal("did not expect non-Telegram platforms to use Telegram DM topic reply fallback")
	}
}

func TestAccountScopedDefaultToolProgressModes(t *testing.T) {
	if got := defaultToolProgressModeForPlatform("discord:ops"); got != "all" {
		t.Fatalf("discord account tool progress mode = %q, want all", got)
	}
	if got := defaultToolProgressModeForPlatform("slack:team-a"); got != "off" {
		t.Fatalf("slack account tool progress mode = %q, want off", got)
	}
}
