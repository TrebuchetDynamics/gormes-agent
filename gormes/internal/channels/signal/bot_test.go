package signal

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/gateway"
)

func TestBot_Name(t *testing.T) {
	b := New(newMockClient(), nil)
	if got := b.Name(); got != "signal" {
		t.Fatalf("Name() = %q, want signal", got)
	}
}

func TestBot_Send_UsesDirectReplyMetadata(t *testing.T) {
	mc := newMockClient()
	b := New(mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.push(InboundMessage{
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

	direct := mc.directSnapshot()
	if len(direct) != 1 {
		t.Fatalf("direct send count = %d, want 1", len(direct))
	}
	if direct[0].RecipientID != "+15551234567" {
		t.Fatalf("direct recipient = %q, want +15551234567", direct[0].RecipientID)
	}
	if direct[0].Options.ReplyToMessageID != "msg-1" {
		t.Fatalf("direct reply metadata = %+v, want msg-1", direct[0].Options)
	}
}

func TestBot_Send_GroupRoutesCanonicalChatIDToNativeGroupID(t *testing.T) {
	mc := newMockClient()
	b := New(mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.push(InboundMessage{
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

	group := mc.groupSnapshot()
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
	b := New(newMockClient(), nil)
	if _, err := b.Send(context.Background(), "group:missing", "reply"); err == nil {
		t.Fatal("Send() error = nil, want unknown chat error")
	}
}

type mockClient struct {
	events  chan InboundMessage
	direct  []sentMessage
	group   []sentMessage
	closeCh chan struct{}
}

type sentMessage struct {
	RecipientID string
	Text        string
	Options     SendOptions
}

func newMockClient() *mockClient {
	return &mockClient{
		events:  make(chan InboundMessage, 16),
		closeCh: make(chan struct{}, 1),
	}
}

func (m *mockClient) Events() <-chan InboundMessage { return m.events }

func (m *mockClient) SendDirect(_ context.Context, recipientID, text string, opts SendOptions) (string, error) {
	m.direct = append(m.direct, sentMessage{RecipientID: recipientID, Text: text, Options: opts})
	return "direct-send-1", nil
}

func (m *mockClient) SendGroup(_ context.Context, recipientID, text string, opts SendOptions) (string, error) {
	m.group = append(m.group, sentMessage{RecipientID: recipientID, Text: text, Options: opts})
	return "group-send-1", nil
}

func (m *mockClient) Close() error {
	select {
	case m.closeCh <- struct{}{}:
	default:
	}
	return nil
}

func (m *mockClient) push(msg InboundMessage) {
	m.events <- msg
}

func (m *mockClient) directSnapshot() []sentMessage {
	out := make([]sentMessage, len(m.direct))
	copy(out, m.direct)
	return out
}

func (m *mockClient) groupSnapshot() []sentMessage {
	out := make([]sentMessage, len(m.group))
	copy(out, m.group)
	return out
}
