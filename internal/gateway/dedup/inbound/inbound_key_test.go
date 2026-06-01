package inbound

import "testing"

func TestResolveMessageIdentity_PrefersMessageIDAndTrims(t *testing.T) {
	identity := ResolveMessageIdentity(EventKeyParts{
		MsgID:     "gateway-msg-1",
		MessageID: " platform-msg-1 ",
	})

	if identity.ID != "platform-msg-1" || identity.Source != MessageIDSourceMessageID {
		t.Fatalf("ResolveMessageIdentity = %+v, want trimmed MessageID provenance", identity)
	}
}

func TestResolveMessageIdentity_FallsBackToMsgID(t *testing.T) {
	identity := ResolveMessageIdentity(EventKeyParts{
		MsgID:     " gateway-msg-1 ",
		MessageID: " ",
	})

	if identity.ID != "gateway-msg-1" || identity.Source != MessageIDSourceMsgID {
		t.Fatalf("ResolveMessageIdentity = %+v, want trimmed MsgID fallback provenance", identity)
	}
}

func TestKey_UsesMsgIDFallbackWhenMessageIDMissing(t *testing.T) {
	result := Key(EventKeyParts{
		Platform:  "telegram",
		ChatID:    "chat-1",
		ThreadID:  "thread-1",
		MsgID:     "gateway-msg-1",
		MessageID: "",
	})

	if result.Evidence != "" {
		t.Fatalf("Key MsgID fallback evidence = %q, want none", result.Evidence)
	}
	if result.Key == "" {
		t.Fatal("Key MsgID fallback key is empty")
	}
	if result.Identity.ID != "gateway-msg-1" || result.Identity.Source != MessageIDSourceMsgID {
		t.Fatalf("Key identity = %+v, want MsgID fallback provenance", result.Identity)
	}
}

func TestKey_MissingBothMessageIDsDegrades(t *testing.T) {
	result := Key(EventKeyParts{
		Platform:  "telegram",
		ChatID:    "chat-1",
		ThreadID:  "thread-1",
		MsgID:     "",
		MessageID: "",
	})

	if result.Key != "" {
		t.Fatalf("Key missing message ID key = %q, want empty", result.Key)
	}
	if result.Evidence != EvidenceMissingMessageID {
		t.Fatalf("Key missing message ID evidence = %q, want %q", result.Evidence, EvidenceMissingMessageID)
	}
}

func TestKey_ReportsScopeEvenWhenMessageIDMissing(t *testing.T) {
	result := Key(EventKeyParts{
		Platform: "telegram",
		ChatID:   "chat-1",
		ThreadID: "thread-1",
	})

	if result.Scope.Platform != "telegram" || result.Scope.ChatID != "chat-1" || result.Scope.ThreadID != "thread-1" {
		t.Fatalf("missing-ID scope = %+v, want original platform/chat/thread", result.Scope)
	}
	if result.Key != "" {
		t.Fatalf("missing-ID key = %q, want empty", result.Key)
	}
}

func TestScopeTrackingKey_LengthPrefixesDelimiterBearingParts(t *testing.T) {
	identity := MessageIdentity{ID: "msg|1", Source: MessageIDSourceMessageID}
	left := Scope{Platform: "tele", ChatID: "gram|chat", ThreadID: "thread"}.TrackingKey(identity)
	right := Scope{Platform: "tele|gram", ChatID: "chat", ThreadID: "thread"}.TrackingKey(identity)

	if left == "" || right == "" {
		t.Fatalf("TrackingKey returned empty keys: left=%q right=%q", left, right)
	}
	if left == right {
		t.Fatalf("TrackingKey collision for delimiter-bearing scope parts: %q", left)
	}
}

func TestScopeTrackingKey_PreservesByteExactScope(t *testing.T) {
	identity := MessageIdentity{ID: "msg-1", Source: MessageIDSourceMessageID}
	trimmed := Scope{Platform: "telegram", ChatID: "chat-1", ThreadID: "thread-1"}.TrackingKey(identity)
	spaced := Scope{Platform: " telegram ", ChatID: "chat-1", ThreadID: "thread-1"}.TrackingKey(identity)

	if trimmed == spaced {
		t.Fatalf("TrackingKey normalized scope unexpectedly: %q", trimmed)
	}
}

func TestKey_StableForSameEvent(t *testing.T) {
	ev := EventKeyParts{
		Platform:  "telegram",
		ChatID:    "chat-1",
		ThreadID:  "thread-1",
		MessageID: "msg-1",
	}

	first := Key(ev)
	second := Key(ev)

	if first.Evidence != "" || second.Evidence != "" {
		t.Fatalf("Key evidence = %q then %q, want none", first.Evidence, second.Evidence)
	}
	if first.Key == "" {
		t.Fatalf("Key key is empty for event with MessageID")
	}
	if first.Key != second.Key {
		t.Fatalf("Key repeated key = %q then %q, want stable key", first.Key, second.Key)
	}
}

func TestKey_ScopesByAccountPlatformChatThreadMessageID(t *testing.T) {
	base := EventKeyParts{
		Platform:  "telegram",
		AccountID: "alerts",
		ChatID:    "chat-1",
		ThreadID:  "thread-1",
		MessageID: "msg-1",
	}
	for name, ev := range map[string]EventKeyParts{
		"account":  {Platform: "telegram", AccountID: "support", ChatID: "chat-1", ThreadID: "thread-1", MessageID: "msg-1"},
		"platform": {Platform: "discord", AccountID: "alerts", ChatID: "chat-1", ThreadID: "thread-1", MessageID: "msg-1"},
		"chat":     {Platform: "telegram", AccountID: "alerts", ChatID: "chat-2", ThreadID: "thread-1", MessageID: "msg-1"},
		"thread":   {Platform: "telegram", AccountID: "alerts", ChatID: "chat-1", ThreadID: "thread-2", MessageID: "msg-1"},
		"message":  {Platform: "telegram", AccountID: "alerts", ChatID: "chat-1", ThreadID: "thread-1", MessageID: "msg-2"},
	} {
		if got, want := Key(ev).Key, Key(base).Key; got == want {
			t.Fatalf("%s variant key = base key %q, want account/platform/chat/thread/message isolation", name, got)
		}
	}
}

func TestKey_ScopesByPlatformChatThreadMessageID(t *testing.T) {
	events := map[string]EventKeyParts{
		"base": {
			Platform:  "telegram",
			ChatID:    "chat-1",
			ThreadID:  "thread-1",
			MessageID: "msg-1",
		},
		"different platform": {
			Platform:  "discord",
			ChatID:    "chat-1",
			ThreadID:  "thread-1",
			MessageID: "msg-1",
		},
		"different chat": {
			Platform:  "telegram",
			ChatID:    "chat-2",
			ThreadID:  "thread-1",
			MessageID: "msg-1",
		},
		"different thread": {
			Platform:  "telegram",
			ChatID:    "chat-1",
			ThreadID:  "thread-2",
			MessageID: "msg-1",
		},
		"different message": {
			Platform:  "telegram",
			ChatID:    "chat-1",
			ThreadID:  "thread-1",
			MessageID: "msg-2",
		},
	}

	seen := map[string]string{}
	for name, ev := range events {
		result := Key(ev)
		if result.Evidence != "" {
			t.Fatalf("%s evidence = %q, want none", name, result.Evidence)
		}
		if result.Key == "" {
			t.Fatalf("%s key is empty", name)
		}
		if prior, ok := seen[result.Key]; ok {
			t.Fatalf("%s key %q matches %s, want scoped key", name, result.Key, prior)
		}
		seen[result.Key] = name
	}
}
