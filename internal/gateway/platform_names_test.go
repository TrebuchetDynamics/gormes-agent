package gateway

import "testing"

func TestAccountScopedPlatformNameCompatibilityWrappers(t *testing.T) {
	if !isTelegramPlatform(" Telegram:ops ") {
		t.Fatal("expected gateway wrapper to recognize account-scoped Telegram platform")
	}
	if got := platformBaseName("Discord:ops"); got != "discord" {
		t.Fatalf("platformBaseName wrapper = %q, want discord", got)
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
