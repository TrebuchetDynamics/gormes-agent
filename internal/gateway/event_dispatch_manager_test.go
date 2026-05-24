package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"
)

func TestGatewayMessageSentEvent_PublishesAfterSuccessfulManagerSend(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	eventsCh := make(chan events.Event, 4)
	bus.Subscribe(TopicMessageSent, func(e events.Event) {
		eventsCh <- e
	})

	tg := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:    map[string]string{"telegram": "42"},
		EventDispatcher: NewEventDispatcher(bus),
	}, &fakeKernel{}, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "origin-99",
		Kind:     EventStart,
	})

	var event events.Event
	select {
	case event = <-eventsCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for gateway.message.sent event")
	}

	if event.Type != TopicMessageSent || event.Source != "telegram" || event.TraceID == "" {
		t.Fatalf("event provenance = type:%q source:%q trace:%q", event.Type, event.Source, event.TraceID)
	}

	var payload MessageEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Platform != "telegram" || payload.ChatID != "42" {
		t.Fatalf("payload target = %+v, want telegram chat 42", payload)
	}
	if payload.MessageID != "1000" || payload.MsgID != "1000" {
		t.Fatalf("payload message ids = message:%q msg:%q, want sent id 1000", payload.MessageID, payload.MsgID)
	}
	if payload.Kind != "message" || payload.Text != startGreeting || payload.Body != startGreeting {
		t.Fatalf("payload content = kind:%q text:%q body:%q", payload.Kind, payload.Text, payload.Body)
	}
	if payload.ReplyToMessageID != "" {
		t.Fatalf("ReplyToMessageID = %q, want empty for non-reply send", payload.ReplyToMessageID)
	}

	select {
	case extra := <-eventsCh:
		t.Fatalf("extra message-sent event: %+v", extra)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestGatewayMessageSentEvent_PreservesReplyContext(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	eventsCh := make(chan events.Event, 1)
	bus.Subscribe(TopicMessageSent, func(e events.Event) {
		eventsCh <- e
	})

	ch := newReplyMessageSentChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		EventDispatcher: NewEventDispatcher(bus),
	}, &fakeKernel{}, slog.Default())

	msgID, err := m.sendWithHooksReply(context.Background(), ch, "42", "origin-99", "reply text")
	if err != nil {
		t.Fatalf("sendWithHooksReply: %v", err)
	}
	if msgID != "3000" {
		t.Fatalf("msgID = %q, want 3000", msgID)
	}

	var event events.Event
	select {
	case event = <-eventsCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for gateway.message.sent event")
	}

	var payload MessageEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Kind != "reply" || payload.ReplyToMessageID != "origin-99" {
		t.Fatalf("reply payload = kind:%q reply_to:%q, want reply/origin-99", payload.Kind, payload.ReplyToMessageID)
	}
	if payload.MessageID != "3000" || payload.Text != "reply text" {
		t.Fatalf("reply payload = %+v, want sent id 3000 and reply text", payload)
	}
}

func TestGatewayMessageSentEvent_DoesNotPublishWhenChannelSendFails(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	eventsCh := make(chan events.Event, 1)
	bus.Subscribe(TopicMessageSent, func(e events.Event) {
		eventsCh <- e
	})

	ch := newFakeChannel("telegram")
	ch.sendErr = errors.New("send failed")
	m := NewManagerWithSubmitter(ManagerConfig{
		EventDispatcher: NewEventDispatcher(bus),
	}, &fakeKernel{}, slog.Default())

	_, err := m.sendWithHooks(context.Background(), ch, "42", "will fail")
	if err == nil {
		t.Fatal("sendWithHooks error = nil, want send failed")
	}

	select {
	case event := <-eventsCh:
		t.Fatalf("unexpected message-sent event after failed send: %+v", event)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestGatewayMessageSentEvent_PublishFailureDoesNotBlockSend(t *testing.T) {
	bus := &failingEventBus{err: errors.New("event bus down")}
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		EventDispatcher: NewEventDispatcher(bus),
	}, &fakeKernel{}, slog.Default())

	msgID, err := m.sendWithHooks(context.Background(), ch, "42", "still visible")
	if err != nil {
		t.Fatalf("sendWithHooks: %v", err)
	}
	if msgID != "1000" {
		t.Fatalf("msgID = %q, want channel send to succeed despite publish failure", msgID)
	}
	if got := bus.publishCount.Load(); got != 1 {
		t.Fatalf("publish count = %d, want 1 attempted message-sent publish", got)
	}
}

type replyMessageSentChannel struct {
	*fakeChannel
}

func newReplyMessageSentChannel(name string) *replyMessageSentChannel {
	ch := &replyMessageSentChannel{fakeChannel: newFakeChannel(name)}
	ch.nextMsgID = 3000
	return ch
}

func (c *replyMessageSentChannel) SendReply(_ context.Context, chatID, replyToMsgID, text string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sendErr != nil {
		return "", c.sendErr
	}
	id := c.nextMsgID
	c.nextMsgID++
	msgID := strconv.Itoa(id)
	c.sent = append(c.sent, fakeSent{ChatID: chatID, Text: text, MsgID: msgID})
	return msgID, nil
}

type failingEventBus struct {
	err          error
	publishCount atomic.Int64
}

func (b *failingEventBus) Publish(string, events.Event) error {
	b.publishCount.Add(1)
	return b.err
}

func (b *failingEventBus) Subscribe(string, events.EventHandler) func() {
	return func() {}
}

func (b *failingEventBus) Close() error {
	return nil
}
