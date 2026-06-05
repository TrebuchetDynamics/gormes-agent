package platforms

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
		if got := IsTelegramPlatform(tt.platform); got != tt.telegram {
			t.Fatalf("IsTelegramPlatform(%q) = %v, want %v", tt.platform, got, tt.telegram)
		}
		if got := IsDiscordPlatform(tt.platform); got != tt.discord {
			t.Fatalf("IsDiscordPlatform(%q) = %v, want %v", tt.platform, got, tt.discord)
		}
		if got := IsSlackPlatform(tt.platform); got != tt.slack {
			t.Fatalf("IsSlackPlatform(%q) = %v, want %v", tt.platform, got, tt.slack)
		}
		if got := PlatformBaseName(tt.platform); got != tt.base {
			t.Fatalf("PlatformBaseName(%q) = %q, want %q", tt.platform, got, tt.base)
		}
	}
}

func TestTelegramDMTopicReplyFallbackLaneIncludesAccounts(t *testing.T) {
	if !TelegramDMTopicReplyFallbackLane("telegram:ops", "12345", "67890") {
		t.Fatal("expected account-scoped Telegram platform to use DM topic reply fallback")
	}
	if TelegramDMTopicReplyFallbackLane("telegram:ops", "-10012345", "67890") {
		t.Fatal("did not expect group chat IDs to use DM topic reply fallback")
	}
	if TelegramDMTopicReplyFallbackLane("discord", "12345", "67890") {
		t.Fatal("did not expect non-Telegram platforms to use Telegram DM topic reply fallback")
	}
}

func TestAccountScopedDefaultToolProgressModes(t *testing.T) {
	if got := DefaultToolProgressModeForPlatform("discord:ops"); got != "all" {
		t.Fatalf("discord account tool progress mode = %q, want all", got)
	}
	if got := DefaultToolProgressModeForPlatform("slack:team-a"); got != "off" {
		t.Fatalf("slack account tool progress mode = %q, want off", got)
	}
}
