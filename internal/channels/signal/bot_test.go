package signal

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBot_Name(t *testing.T) {
	b := New(newMockTransport(), nil)
	if got := b.Name(); got != "signal" {
		t.Fatalf("Name() = %q, want signal", got)
	}
}

func TestBot_Send_UsesDirectReplyMetadata(t *testing.T) {
	mt := newMockTransport()
	b := New(mt, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mt.push(InboundMessage{
		ChatType:   ChatTypeDirect,
		SenderID:   "+15551234567",
		SenderUUID: "uuid-alice",
		SenderName: "Alice",
		MessageID:  "msg-1",
		Text:       "hello",
	})

	select {
	case <-inbox:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected inbound event before send")
	}

	msgID, err := b.Send(context.Background(), "+15551234567", "reply one")
	if err != nil {
		t.Fatalf("Send() error = %v, want nil", err)
	}
	if msgID != "direct-send-1" {
		t.Fatalf("Send() msgID = %q, want direct-send-1", msgID)
	}

	direct := mt.directSnapshot()
	if len(direct) != 1 {
		t.Fatalf("direct send count = %d, want 1", len(direct))
	}
	if direct[0].RecipientID != "+15551234567" {
		t.Fatalf("direct recipient = %q, want +15551234567", direct[0].RecipientID)
	}
	if direct[0].Options.ReplyToMessageID != "msg-1" {
		t.Fatalf("direct reply metadata = %+v, want msg-1", direct[0].Options)
	}
	if len(direct[0].Options.Attachments) != 0 {
		t.Fatalf("plain Send should not carry attachments, got %d", len(direct[0].Options.Attachments))
	}
}

func TestBot_Send_GroupRoutesCanonicalChatIDToNativeGroupID(t *testing.T) {
	mt := newMockTransport()
	b := New(mt, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mt.push(InboundMessage{
		ChatType:   ChatTypeGroup,
		GroupID:    "grp-123==",
		GroupName:  "Ops",
		SenderID:   "+15557654321",
		SenderUUID: "uuid-bob",
		SenderName: "Bob",
		MessageID:  "msg-2",
		Text:       "hello group",
	})

	select {
	case <-inbox:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected inbound event before send")
	}

	msgID, err := b.Send(context.Background(), "group:grp-123==", "reply one")
	if err != nil {
		t.Fatalf("Send() error = %v, want nil", err)
	}
	if msgID != "group-send-1" {
		t.Fatalf("Send() msgID = %q, want group-send-1", msgID)
	}

	group := mt.groupSnapshot()
	if len(group) != 1 {
		t.Fatalf("group send count = %d, want 1", len(group))
	}
	if group[0].RecipientID != "grp-123==" {
		t.Fatalf("group recipient = %q, want grp-123==", group[0].RecipientID)
	}
	if group[0].Options.ReplyToMessageID != "msg-2" {
		t.Fatalf("group reply metadata = %+v, want msg-2", group[0].Options)
	}
}

func TestBot_Send_RejectsUnknownChatID(t *testing.T) {
	b := New(newMockTransport(), nil)
	if _, err := b.Send(context.Background(), "group:missing", "reply"); err == nil {
		t.Fatal("Send() error = nil, want unknown chat error")
	}
}

type sentMessage struct {
	RecipientID string
	Text        string
	Options     SendOptions
}
