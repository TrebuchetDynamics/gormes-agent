package gateway

import "testing"

func TestInboundDedupKeyCompatibilityWrapperIncludesAccountID(t *testing.T) {
	alerts := InboundDedupKey(InboundEvent{Platform: "telegram", AccountID: "alerts", ChatID: "chat", ThreadID: "thread", MessageID: "msg"})
	support := InboundDedupKey(InboundEvent{Platform: "telegram", AccountID: "support", ChatID: "chat", ThreadID: "thread", MessageID: "msg"})
	if alerts.Key == "" || support.Key == "" || alerts.Key == support.Key {
		t.Fatalf("account-scoped wrapper keys = %q and %q, want non-empty distinct keys", alerts.Key, support.Key)
	}
	if alerts.Scope.AccountID != "alerts" || support.Scope.AccountID != "support" {
		t.Fatalf("wrapper scopes = %+v and %+v, want account provenance", alerts.Scope, support.Scope)
	}
}

func TestInboundDedupKeyCompatibilityWrapper(t *testing.T) {
	result := InboundDedupKey(InboundEvent{Platform: "telegram", ChatID: "chat", ThreadID: "thread", MessageID: "msg"})
	if result.Key == "" || result.Evidence != "" {
		t.Fatalf("InboundDedupKey wrapper = %+v, want key without evidence", result)
	}
	if result.Identity.ID != "msg" || result.Identity.Source != InboundMessageIDSourceMessageID {
		t.Fatalf("InboundDedupKey identity = %+v, want MessageID provenance", result.Identity)
	}
	missing := InboundDedupKey(InboundEvent{Platform: "telegram", ChatID: "chat"})
	if missing.Evidence != MessageDeduplicatorEvidenceMissingMessageID {
		t.Fatalf("InboundDedupKey missing evidence = %q, want %q", missing.Evidence, MessageDeduplicatorEvidenceMissingMessageID)
	}
}

func TestResolveInboundMessageIdentityCompatibilityWrapper(t *testing.T) {
	identity := ResolveInboundMessageIdentity(InboundEvent{MsgID: " fallback-msg ", MessageID: " "})
	if identity.ID != "fallback-msg" || identity.Source != InboundMessageIDSourceMsgID {
		t.Fatalf("ResolveInboundMessageIdentity wrapper = %+v, want MsgID provenance", identity)
	}
}
