package yuanbao

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// fakeClient drives the Yuanbao Channel without any network access. The test
// hands push events to the channel via the inbound queue and observes outbound
// sends through Sent.
type fakeClient struct {
	connectErr error

	inboundMu sync.Mutex
	inbound   []InboundPush

	sentMu sync.Mutex
	sent   []SentMessage
}

func (f *fakeClient) queueInbound(push InboundPush) {
	f.inboundMu.Lock()
	defer f.inboundMu.Unlock()
	f.inbound = append(f.inbound, push)
}

func (f *fakeClient) Connect(_ context.Context) error {
	return f.connectErr
}

func (f *fakeClient) Run(ctx context.Context, deliver func(context.Context, InboundPush)) error {
	for {
		f.inboundMu.Lock()
		if len(f.inbound) == 0 {
			f.inboundMu.Unlock()
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Millisecond):
				continue
			}
		}
		next := f.inbound[0]
		f.inbound = f.inbound[1:]
		f.inboundMu.Unlock()
		deliver(ctx, next)
	}
}

func (f *fakeClient) Send(_ context.Context, conversationID, text string) (string, error) {
	f.sentMu.Lock()
	defer f.sentMu.Unlock()
	f.sent = append(f.sent, SentMessage{ConversationID: conversationID, Text: text})
	return "fake-yuanbao-msg-id", nil
}

func (f *fakeClient) sentSnapshot() []SentMessage {
	f.sentMu.Lock()
	defer f.sentMu.Unlock()
	return append([]SentMessage(nil), f.sent...)
}

func TestYuanbaoChannel_NameIsStablePlatformIdentifier(t *testing.T) {
	t.Parallel()
	ch := NewChannel(Config{}, &fakeClient{}, slog.Default())
	if got, want := ch.Name(), "yuanbao"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
}

func TestYuanbaoChannel_RunDeliversInboundEnvelopeIntoInbox(t *testing.T) {
	t.Parallel()
	client := &fakeClient{}
	client.queueInbound(InboundPush{
		ConversationID: "conv-99",
		MessageID:      "msg-7",
		AuthorRole:     "user",
		Text:           "hello gormes",
	})

	ch := NewChannel(Config{AllowedConversationID: "conv-99"}, client, slog.Default())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	inbox := make(chan gateway.InboundEvent, 1)

	done := make(chan error, 1)
	go func() { done <- ch.Run(ctx, inbox) }()

	select {
	case ev := <-inbox:
		if ev.Platform != "yuanbao" {
			t.Fatalf("ev.Platform = %q, want yuanbao", ev.Platform)
		}
		if ev.ChatID != "conv-99" || ev.MsgID != "msg-7" {
			t.Fatalf("ev = %+v", ev)
		}
		if ev.Text != "hello gormes" {
			t.Fatalf("ev.Text = %q", ev.Text)
		}
		if ev.Kind != gateway.EventSubmit {
			t.Fatalf("ev.Kind = %v, want EventSubmit", ev.Kind)
		}
	case <-time.After(time.Second):
		t.Fatal("inbound event not delivered")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("Run after cancel = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancellation")
	}
}

func TestYuanbaoChannel_RunDropsEventsFromDisallowedConversation(t *testing.T) {
	t.Parallel()
	client := &fakeClient{}
	client.queueInbound(InboundPush{
		ConversationID: "other-conv",
		MessageID:      "msg-x",
		AuthorRole:     "user",
		Text:           "should not pass",
	})
	client.queueInbound(InboundPush{
		ConversationID: "conv-99",
		MessageID:      "msg-y",
		AuthorRole:     "user",
		Text:           "allowed",
	})

	ch := NewChannel(Config{AllowedConversationID: "conv-99"}, client, slog.Default())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	inbox := make(chan gateway.InboundEvent, 2)

	go func() { _ = ch.Run(ctx, inbox) }()

	select {
	case ev := <-inbox:
		if ev.ChatID != "conv-99" {
			t.Fatalf("first delivered event ChatID=%q, want conv-99 only", ev.ChatID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected event from allowed conversation")
	}

	select {
	case ev := <-inbox:
		t.Fatalf("unexpected second event %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
}

func TestYuanbaoChannel_SendForwardsToFakeClient(t *testing.T) {
	t.Parallel()
	client := &fakeClient{}
	ch := NewChannel(Config{}, client, slog.Default())

	id, err := ch.Send(context.Background(), "conv-42", "outbound payload")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if id != "fake-yuanbao-msg-id" {
		t.Fatalf("Send msgID = %q", id)
	}
	sent := client.sentSnapshot()
	if len(sent) != 1 || sent[0].ConversationID != "conv-42" || sent[0].Text != "outbound payload" {
		t.Fatalf("client did not record outbound send: %#v", sent)
	}
}

func TestYuanbaoChannel_RunSurfacesConnectFailure(t *testing.T) {
	t.Parallel()
	connectErr := errors.New("login refused")
	client := &fakeClient{connectErr: connectErr}
	ch := NewChannel(Config{}, client, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := ch.Run(ctx, make(chan gateway.InboundEvent, 1))
	if !errors.Is(err, connectErr) {
		t.Fatalf("Run() = %v, want %v", err, connectErr)
	}
}
