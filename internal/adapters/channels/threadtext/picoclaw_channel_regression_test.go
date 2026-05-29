package threadtext

import "testing"

func TestPicoClawChannelRegression_NormalizeInboundPreservesSourceMessageID(t *testing.T) {
	ev, ok := NormalizeInbound("matrix", InboundMessage{
		ChatID:       "!ops:example.org",
		UserID:       "@alice:example.org",
		MessageID:    "$mx-message-1",
		Text:         "inspect channel state",
		ThreadRootID: "$mx-thread-root",
	})
	if !ok {
		t.Fatal("NormalizeInbound rejected valid Matrix-style message")
	}
	if ev.MsgID != "$mx-message-1" {
		t.Fatalf("MsgID = %q, want source message ID", ev.MsgID)
	}
	if ev.MessageID != "$mx-message-1" {
		t.Fatalf("MessageID = %q, want source message ID", ev.MessageID)
	}
}
