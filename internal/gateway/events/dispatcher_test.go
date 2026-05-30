package events

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

func TestEventDispatcher_PublishMessageReceived(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	var received atomic.Int32
	bus.Subscribe(TopicMessageReceived, func(e Event) {
		received.Add(1)
	})

	disp := NewEventDispatcher(bus)
	err := disp.PublishMessageReceived("telegram", "trace-1", map[string]string{"text": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)

	if received.Load() != 1 {
		t.Fatalf("received %d events, want 1", received.Load())
	}
}

func TestEventDispatcher_MessageEventPayloadRoundTrip(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	delivered := make(chan Event, 1)
	bus.Subscribe(TopicMessageReceived, func(e Event) {
		delivered <- e
	})

	disp := NewEventDispatcher(bus)
	payload := MessageEventPayload{
		Platform:         "telegram",
		ChatID:           "42",
		ChatType:         "private",
		UserID:           "7",
		UserName:         "Ada",
		ThreadID:         "1",
		MessageID:        "99",
		MsgID:            "99",
		ReplyToMessageID: "",
		Kind:             "submit",
		Text:             "hello",
		Body:             "hello",
	}
	if err := disp.PublishMessageReceived("telegram", "trace-tg-99", payload); err != nil {
		t.Fatalf("PublishMessageReceived: %v", err)
	}

	select {
	case got := <-delivered:
		if got.Source != "telegram" || got.TraceID != "trace-tg-99" {
			t.Fatalf("event provenance = source:%q trace:%q, want telegram/trace-tg-99", got.Source, got.TraceID)
		}
		var decoded MessageEventPayload
		if err := json.Unmarshal(got.Payload, &decoded); err != nil {
			t.Fatalf("payload decode: %v", err)
		}
		if decoded != payload {
			t.Fatalf("decoded payload = %+v, want %+v", decoded, payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message payload event")
	}
}

func TestEventDispatcher_MessageEventPayloadRoundTripThreadProvenance(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	delivered := make(chan Event, 1)
	bus.Subscribe(TopicMessageReceived, func(e Event) {
		delivered <- e
	})

	disp := NewEventDispatcher(bus)
	payload := MessageEventPayload{
		Platform:     "discord",
		ChatID:       "forum-100",
		ParentChatID: "forum-100",
		GuildID:      "guild-1",
		UserID:       "user-7",
		ThreadID:     "thread-200",
		MessageID:    "msg-201",
		MsgID:        "msg-201",
		Kind:         "submit",
		Text:         "follow up from the forum post",
		Body:         "follow up from the forum post",
	}
	if err := disp.PublishMessageReceived("discord", "trace-discord-201", payload); err != nil {
		t.Fatalf("PublishMessageReceived: %v", err)
	}

	select {
	case got := <-delivered:
		if got.Source != "discord" || got.TraceID != "trace-discord-201" {
			t.Fatalf("event provenance = source:%q trace:%q, want discord/trace-discord-201", got.Source, got.TraceID)
		}
		var decoded MessageEventPayload
		if err := json.Unmarshal(got.Payload, &decoded); err != nil {
			t.Fatalf("payload decode: %v", err)
		}
		if decoded != payload {
			t.Fatalf("decoded payload = %+v, want %+v", decoded, payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message payload event")
	}
}

func TestEventDispatcher_PublishMessageSent(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	var received atomic.Int32
	bus.Subscribe(TopicMessageSent, func(e Event) {
		received.Add(1)
	})

	disp := NewEventDispatcher(bus)
	disp.PublishMessageSent("discord", "trace-2", map[string]string{"text": "reply"})
	time.Sleep(30 * time.Millisecond)

	if received.Load() != 1 {
		t.Fatalf("received %d events, want 1", received.Load())
	}
}

func TestEventDispatcher_SessionLifecycle(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	var started, ended atomic.Int32
	bus.Subscribe(TopicSessionStarted, func(e Event) { started.Add(1) })
	bus.Subscribe(TopicSessionEnded, func(e Event) { ended.Add(1) })

	disp := NewEventDispatcher(bus)
	disp.PublishSessionStarted("slack", "trace-3", map[string]string{"user": "alice"})
	disp.PublishSessionEnded("slack", "trace-3", map[string]string{"user": "alice"})
	time.Sleep(30 * time.Millisecond)

	if started.Load() != 1 || ended.Load() != 1 {
		t.Fatalf("started=%d ended=%d, want 1 each", started.Load(), ended.Load())
	}
}

func TestEventDispatcher_SubscribeMessages(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	disp := NewEventDispatcher(bus)

	var count atomic.Int32
	unsub := disp.SubscribeMessages(func(e Event) {
		count.Add(1)
	})

	disp.PublishMessageReceived("telegram", "t1", map[string]string{"x": "y"})
	time.Sleep(30 * time.Millisecond)
	if count.Load() != 1 {
		t.Fatal("subscriber did not receive event")
	}

	unsub()
	disp.PublishMessageReceived("telegram", "t2", map[string]string{"x": "y"})
	time.Sleep(30 * time.Millisecond)
	if count.Load() != 1 {
		t.Fatal("unsubscribed handler received event")
	}
}
