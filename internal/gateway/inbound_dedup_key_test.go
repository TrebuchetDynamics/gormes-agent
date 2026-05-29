package gateway

import "testing"

func TestInboundDedupKeyCompatibilityWrapper(t *testing.T) {
	result := InboundDedupKey(InboundEvent{Platform: "telegram", ChatID: "chat", ThreadID: "thread", MessageID: "msg"})
	if result.Key == "" || result.Evidence != "" {
		t.Fatalf("InboundDedupKey wrapper = %+v, want key without evidence", result)
	}
	missing := InboundDedupKey(InboundEvent{Platform: "telegram", ChatID: "chat"})
	if missing.Evidence != MessageDeduplicatorEvidenceMissingMessageID {
		t.Fatalf("InboundDedupKey missing evidence = %q, want %q", missing.Evidence, MessageDeduplicatorEvidenceMissingMessageID)
	}
}
