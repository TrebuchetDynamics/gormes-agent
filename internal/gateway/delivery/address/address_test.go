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

func TestChatWithThreadAvoidsColonDelimiterCollisions(t *testing.T) {
	first := ChatWithThread("room:thread", "x")
	second := ChatWithThread("room", "thread:x")
	if first == second {
		t.Fatalf("ChatWithThread collision for colon-bearing IDs: %q", first)
	}
	if ChatMatches(first, "room", "thread:x") {
		t.Fatalf("ChatMatches matched wrong colon-bearing chat/thread pair: %q", first)
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
