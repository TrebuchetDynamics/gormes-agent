package bluebubbles

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channeltest"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBot_Run_RequiresWebhookAuthAndNormalizesCommands(t *testing.T) {
	mc := newMockClient()
	b := New(Config{Password: "secret"}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.push(InboundMessage{
		AuthToken:      "wrong",
		ChatGUID:       "chat-guid-1",
		ChatIdentifier: "friend@example.com",
		Sender:         "friend@example.com",
		Text:           "/help",
	})
	channeltest.AssertNoInbound(t, inbox)

	mc.push(InboundMessage{
		AuthToken:      "secret",
		MessageID:      "msg-1",
		ChatGUID:       "chat-guid-1",
		ChatIdentifier: "friend@example.com",
		Sender:         "friend@example.com",
		SenderName:     "Alice",
		Text:           "/help",
	})

	select {
	case ev := <-inbox:
		if ev.Platform != "bluebubbles" || ev.ChatID != "chat-guid-1" || ev.UserID != "friend@example.com" {
			t.Fatalf("unexpected event identity: %+v", ev)
		}
		if ev.Kind != gateway.EventStart {
			t.Fatalf("Kind = %v, want %v", ev.Kind, gateway.EventStart)
		}
		if ev.Text != "" {
			t.Fatalf("Text = %q, want empty after /help parse", ev.Text)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no inbound event")
	}
}

func TestBot_Send_UsesCachedChatGUIDAndHomeChannelFallback(t *testing.T) {
	mc := newMockClient()
	mc.resolved["+15551234567"] = "chat-guid-home"
	b := New(Config{
		Password:    "secret",
		HomeChannel: "+15551234567",
	}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.push(InboundMessage{
		AuthToken:      "secret",
		MessageID:      "msg-1",
		ChatGUID:       "chat-guid-1",
		ChatIdentifier: "friend@example.com",
		Sender:         "friend@example.com",
		Text:           "hello",
	})

	select {
	case <-inbox:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected inbound event before send")
	}

	msgID, err := b.Send(context.Background(), "friend@example.com", "**reply**")
	if err != nil {
		t.Fatalf("Send(cached) error = %v", err)
	}
	if msgID != "send-1" {
		t.Fatalf("Send(cached) msgID = %q, want send-1", msgID)
	}
	if len(mc.resolveCalls) != 0 {
		t.Fatalf("resolve calls = %v, want cache hit without resolve", mc.resolveCalls)
	}
	if len(mc.sent) != 1 {
		t.Fatalf("send count after cached send = %d, want 1", len(mc.sent))
	}
	if mc.sent[0].ChatGUID != "chat-guid-1" {
		t.Fatalf("cached chat guid = %q, want chat-guid-1", mc.sent[0].ChatGUID)
	}
	if mc.sent[0].Text != "reply" {
		t.Fatalf("cached text = %q, want stripped markdown reply", mc.sent[0].Text)
	}

	msgID, err = b.Send(context.Background(), "", "_notice_")
	if err != nil {
		t.Fatalf("Send(home) error = %v", err)
	}
	if msgID != "send-2" {
		t.Fatalf("Send(home) msgID = %q, want send-2", msgID)
	}
	if len(mc.resolveCalls) != 1 || mc.resolveCalls[0] != "+15551234567" {
		t.Fatalf("resolve calls = %v, want [+15551234567]", mc.resolveCalls)
	}
	if len(mc.sent) != 2 {
		t.Fatalf("send count after home send = %d, want 2", len(mc.sent))
	}
	if mc.sent[1].ChatGUID != "chat-guid-home" {
		t.Fatalf("home chat guid = %q, want chat-guid-home", mc.sent[1].ChatGUID)
	}
	if mc.sent[1].Text != "notice" {
		t.Fatalf("home text = %q, want stripped markdown notice", mc.sent[1].Text)
	}
}

type mockClient struct {
	events       chan InboundMessage
	resolved     map[string]string
	resolveCalls []string
	sent         []sentMessage
}

type sentMessage struct {
	ChatGUID string
	Text     string
}

func newMockClient() *mockClient {
	return &mockClient{
		events:   make(chan InboundMessage, 16),
		resolved: map[string]string{},
	}
}

func (m *mockClient) Events() <-chan InboundMessage { return m.events }

func (m *mockClient) ResolveChat(_ context.Context, target string) (string, error) {
	m.resolveCalls = append(m.resolveCalls, target)
	return m.resolved[target], nil
}

func (m *mockClient) SendText(_ context.Context, chatGUID, text string) (string, error) {
	m.sent = append(m.sent, sentMessage{ChatGUID: chatGUID, Text: text})
	return "send-" + string(rune('0'+len(m.sent))), nil
}

func (m *mockClient) Close() error { return nil }

func (m *mockClient) push(msg InboundMessage) {
	m.events <- msg
}

func TestBot_Send_SplitsBlankLineParagraphsIntoSeparateBubbles(t *testing.T) {
	mc := newMockClient()
	mc.resolved["+15551234567"] = "chat-guid-1"
	b := New(Config{HomeChannel: "+15551234567"}, mc, nil)

	msgID, err := b.Send(context.Background(), "+15551234567", "**first** paragraph\n\nsecond paragraph\n\n  \n_third_")
	if err != nil {
		t.Fatalf("Send error = %v", err)
	}
	if len(mc.sent) != 3 {
		t.Fatalf("send count = %d, want 3 paragraph bubbles; got %+v", len(mc.sent), mc.sent)
	}
	wantTexts := []string{"first paragraph", "second paragraph", "third"}
	for i, want := range wantTexts {
		if mc.sent[i].Text != want {
			t.Fatalf("bubble %d text = %q, want %q", i, mc.sent[i].Text, want)
		}
		if mc.sent[i].ChatGUID != "chat-guid-1" {
			t.Fatalf("bubble %d chat = %q, want chat-guid-1", i, mc.sent[i].ChatGUID)
		}
	}
	if msgID != "send-3" {
		t.Fatalf("returned msgID = %q, want send-3 (id of last bubble)", msgID)
	}
}

func TestBot_Send_LongParagraphChunkOmitsPaginationSuffix(t *testing.T) {
	mc := newMockClient()
	mc.resolved["+15551234567"] = "chat-guid-1"
	b := New(Config{HomeChannel: "+15551234567"}, mc, nil)

	paragraph := strings.Repeat("a", MaxMessageLength*2+1)
	if _, err := b.Send(context.Background(), "+15551234567", paragraph); err != nil {
		t.Fatalf("Send error = %v", err)
	}
	if len(mc.sent) < 2 {
		t.Fatalf("long paragraph produced %d chunks, want at least 2", len(mc.sent))
	}
	suffix := regexp.MustCompile(`\s*\(\d+/\d+\)$`)
	var joined string
	for i, msg := range mc.sent {
		if suffix.MatchString(msg.Text) {
			t.Fatalf("chunk %d text %q has forbidden pagination suffix", i, msg.Text)
		}
		joined += msg.Text
	}
	if joined != paragraph {
		t.Fatalf("joined chunks length = %d, want %d (must reconstruct stripped original)", len(joined), len(paragraph))
	}
}

func TestBot_DoesNotImplementMessageEditorOrPlaceholderCapable(t *testing.T) {
	var iface interface{} = New(Config{}, newMockClient(), nil)
	if _, ok := iface.(gateway.MessageEditor); ok {
		t.Fatal("Bot must not implement gateway.MessageEditor (iMessage non-editable)")
	}
	if _, ok := iface.(gateway.PlaceholderCapable); ok {
		t.Fatal("Bot must not implement gateway.PlaceholderCapable (iMessage non-placeholder)")
	}
}
