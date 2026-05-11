package dingtalk

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var _ Client = (*StreamClient)(nil)

func TestNewStreamClient_AcceptsValidCredentials(t *testing.T) {
	sc := NewStreamClient("my-client-id", "my-client-secret", nil)
	if sc == nil {
		t.Fatal("NewStreamClient returned nil")
	}
	if sc.Events() == nil {
		t.Fatal("Events() channel is nil")
	}
}

func TestStreamClient_SendReply_RejectsEmptyWebhook(t *testing.T) {
	sc := NewStreamClient("x", "x", nil)
	_, err := sc.SendReply(context.Background(), "", "hello")
	if err == nil {
		t.Fatal("expected error for empty webhook, got nil")
	}
}

func TestStreamClient_CloseReturnsNil(t *testing.T) {
	sc := NewStreamClient("x", "x", nil)
	if err := sc.Close(); err != nil {
		t.Fatalf("Close() = %v, want nil", err)
	}
}

func TestDingTalkReceiveLoop(t *testing.T) {
	mc := newMockClient()
	b := New(Config{}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.push(InboundMessage{
		MessageID:        "msg-1",
		ConversationID:   "dm-42",
		ConversationType: "1",
		SenderID:         "user-42",
		Text:             "hello",
		SessionWebhook:   "https://example.invalid/reply",
	})

	select {
	case ev := <-inbox:
		if ev.Platform != "dingtalk" {
			t.Fatalf("Platform = %q, want dingtalk", ev.Platform)
		}
		if ev.ChatID != "dm-42" {
			t.Fatalf("ChatID = %q, want dm-42", ev.ChatID)
		}
		if ev.Text != "hello" {
			t.Fatalf("Text = %q, want hello", ev.Text)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for inbound event")
	}
}

func TestDingTalkSendLifecycle(t *testing.T) {
	mc := newMockClient()
	b := New(Config{}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.push(InboundMessage{
		MessageID:        "msg-1",
		ConversationID:   "dm-42",
		ConversationType: "1",
		SenderID:         "user-42",
		Text:             "hello",
		SessionWebhook:   "https://example.invalid/reply",
	})

	select {
	case <-inbox:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected inbound event before send")
	}

	msgID, err := b.Send(context.Background(), "dm-42", "reply")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if msgID == "" {
		t.Fatal("Send() returned empty msgID")
	}

	sent := mc.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("send count = %d, want 1", len(sent))
	}
	if sent[0].Text != "reply" {
		t.Fatalf("text = %q, want reply", sent[0].Text)
	}
}
