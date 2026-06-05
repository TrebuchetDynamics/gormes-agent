package address

import "testing"

func TestPlatformAndIDNormalizeDeliveryAddressFields(t *testing.T) {
	if got := Platform(" Telegram "); got != "telegram" {
		t.Fatalf("Platform = %q, want telegram", got)
	}
	if got := ID(" Chat:Thread "); got != "Chat:Thread" {
		t.Fatalf("ID = %q, want provider identifier casing preserved", got)
	}
}

func TestChatMatchesThreadedSessionKey(t *testing.T) {
	if !ChatMatches(" -100:10 ", "-100", "10") {
		t.Fatal("ChatMatches threaded key = false, want true")
	}
	if ChatMatches("-100", "-100", "10") {
		t.Fatal("ChatMatches unthreaded key with requested thread = true, want false")
	}
}
