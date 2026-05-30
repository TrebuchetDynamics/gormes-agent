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
