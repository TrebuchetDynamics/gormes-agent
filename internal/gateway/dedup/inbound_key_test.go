package dedup

import "testing"

func TestResolveInboundMessageIdentity_PrefersMessageIDAndTrims(t *testing.T) {
	identity := ResolveInboundMessageIdentity(InboundEventKeyParts{
		MsgID:     "gateway-msg-1",
		MessageID: " platform-msg-1 ",
	})

	if identity.ID != "platform-msg-1" || identity.Source != InboundMessageIDSourceMessageID {
		t.Fatalf("ResolveInboundMessageIdentity = %+v, want trimmed MessageID provenance", identity)
	}
}

func TestResolveInboundMessageIdentity_FallsBackToMsgID(t *testing.T) {
	identity := ResolveInboundMessageIdentity(InboundEventKeyParts{
		MsgID:     " gateway-msg-1 ",
		MessageID: " ",
	})

	if identity.ID != "gateway-msg-1" || identity.Source != InboundMessageIDSourceMsgID {
		t.Fatalf("ResolveInboundMessageIdentity = %+v, want trimmed MsgID fallback provenance", identity)
	}
}

func TestInboundDedupKey_UsesMsgIDFallbackWhenMessageIDMissing(t *testing.T) {
	result := InboundDedupKey(InboundEventKeyParts{
		Platform:  "telegram",
		ChatID:    "chat-1",
		ThreadID:  "thread-1",
		MsgID:     "gateway-msg-1",
		MessageID: "",
	})

	if result.Evidence != "" {
		t.Fatalf("InboundDedupKey MsgID fallback evidence = %q, want none", result.Evidence)
	}
	if result.Key == "" {
		t.Fatal("InboundDedupKey MsgID fallback key is empty")
	}
	if result.Identity.ID != "gateway-msg-1" || result.Identity.Source != InboundMessageIDSourceMsgID {
		t.Fatalf("InboundDedupKey identity = %+v, want MsgID fallback provenance", result.Identity)
	}
}

func TestInboundDedupKey_MissingBothMessageIDsDegrades(t *testing.T) {
	result := InboundDedupKey(InboundEventKeyParts{
		Platform:  "telegram",
		ChatID:    "chat-1",
		ThreadID:  "thread-1",
		MsgID:     "",
		MessageID: "",
	})

	if result.Key != "" {
		t.Fatalf("InboundDedupKey missing message ID key = %q, want empty", result.Key)
	}
	if result.Evidence != EvidenceMissingMessageID {
		t.Fatalf("InboundDedupKey missing message ID evidence = %q, want %q", result.Evidence, EvidenceMissingMessageID)
	}
}

func TestInboundDedupKey_ReportsScopeEvenWhenMessageIDMissing(t *testing.T) {
	result := InboundDedupKey(InboundEventKeyParts{
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

func TestInboundDedupScopeTrackingKey_LengthPrefixesDelimiterBearingParts(t *testing.T) {
	identity := InboundMessageIdentity{ID: "msg|1", Source: InboundMessageIDSourceMessageID}
	left := InboundDedupScope{Platform: "tele", ChatID: "gram|chat", ThreadID: "thread"}.TrackingKey(identity)
	right := InboundDedupScope{Platform: "tele|gram", ChatID: "chat", ThreadID: "thread"}.TrackingKey(identity)

	if left == "" || right == "" {
		t.Fatalf("TrackingKey returned empty keys: left=%q right=%q", left, right)
	}
	if left == right {
		t.Fatalf("TrackingKey collision for delimiter-bearing scope parts: %q", left)
	}
}

func TestInboundDedupScopeTrackingKey_PreservesByteExactScope(t *testing.T) {
	identity := InboundMessageIdentity{ID: "msg-1", Source: InboundMessageIDSourceMessageID}
	trimmed := InboundDedupScope{Platform: "telegram", ChatID: "chat-1", ThreadID: "thread-1"}.TrackingKey(identity)
	spaced := InboundDedupScope{Platform: " telegram ", ChatID: "chat-1", ThreadID: "thread-1"}.TrackingKey(identity)

	if trimmed == spaced {
		t.Fatalf("TrackingKey normalized scope unexpectedly: %q", trimmed)
	}
}

func TestInboundDedupKey_StableForSameEvent(t *testing.T) {
	ev := InboundEventKeyParts{
		Platform:  "telegram",
		ChatID:    "chat-1",
		ThreadID:  "thread-1",
		MessageID: "msg-1",
	}

	first := InboundDedupKey(ev)
	second := InboundDedupKey(ev)

	if first.Evidence != "" || second.Evidence != "" {
		t.Fatalf("InboundDedupKey evidence = %q then %q, want none", first.Evidence, second.Evidence)
	}
	if first.Key == "" {
		t.Fatalf("InboundDedupKey key is empty for event with MessageID")
	}
	if first.Key != second.Key {
		t.Fatalf("InboundDedupKey repeated key = %q then %q, want stable key", first.Key, second.Key)
	}
}

func TestInboundDedupKey_ScopesByPlatformChatThreadMessageID(t *testing.T) {
	events := map[string]InboundEventKeyParts{
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
		result := InboundDedupKey(ev)
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
