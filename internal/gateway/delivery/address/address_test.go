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

func TestChatMatchesLegacyUnthreadedColonChatKey(t *testing.T) {
	if !ChatMatches("room:server", "room:server", "") {
		t.Fatal("ChatMatches legacy unthreaded colon chat key = false, want true")
	}
	if !ChatMatches(" !room:server ", "!room:server", "") {
		t.Fatal("ChatMatches trimmed matrix-style colon chat key = false, want true")
	}
}

func TestChatWithThreadAvoidsUnthreadedColonChatCollision(t *testing.T) {
	unthreadedColonChat := ChatWithThread("room:thread", "")
	threadedPlainChat := ChatWithThread("room", "thread")
	if unthreadedColonChat == threadedPlainChat {
		t.Fatalf("ChatWithThread collision between unthreaded colon chat and threaded plain chat: %q", unthreadedColonChat)
	}
	if ChatMatches(unthreadedColonChat, "room", "thread") {
		t.Fatalf("ChatMatches matched unthreaded colon chat as threaded plain chat: %q", unthreadedColonChat)
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
