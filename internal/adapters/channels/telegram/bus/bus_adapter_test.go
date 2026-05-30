package bus

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"
)

func TestBusAdapter_PublishInboundMessageReceived(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	delivered := make(chan events.Event, 1)
	bus.Subscribe(gateway.TopicMessageReceived, func(e events.Event) {
		delivered <- e
	})

	adapter := NewBusAdapter(gateway.NewEventDispatcher(bus))
	err := adapter.PublishInboundMessage("trace-tg-99", gateway.InboundEvent{
		Platform:  "telegram",
		ChatID:    "42",
		ChatType:  "private",
		UserID:    "7",
		UserName:  "Ada",
		ThreadID:  "1",
		MsgID:     "99",
		MessageID: "99",
		Kind:      gateway.EventSubmit,
		Text:      "hello",
	})
	if err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	select {
	case got := <-delivered:
		if got.Type != gateway.TopicMessageReceived {
			t.Fatalf("event type = %q, want %q", got.Type, gateway.TopicMessageReceived)
		}
		if got.Source != "telegram" || got.TraceID != "trace-tg-99" {
			t.Fatalf("event provenance = source:%q trace:%q, want telegram/trace-tg-99", got.Source, got.TraceID)
		}
		var payload gateway.MessageEventPayload
		if err := json.Unmarshal(got.Payload, &payload); err != nil {
			t.Fatalf("payload decode: %v", err)
		}
		if payload.Platform != "telegram" || payload.ChatID != "42" || payload.UserID != "7" || payload.MessageID != "99" {
			t.Fatalf("payload provenance = %+v, want telegram chat/user/message", payload)
		}
		if payload.Kind != "submit" || payload.Text != "hello" || payload.Body != "hello" {
			t.Fatalf("payload body = kind:%q text:%q body:%q, want submit/hello/hello", payload.Kind, payload.Text, payload.Body)
		}
		if payload.ThreadID != "1" || payload.ReplyToMessageID != "" {
			t.Fatalf("payload thread/reply = thread:%q reply:%q, want 1/empty", payload.ThreadID, payload.ReplyToMessageID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Telegram bus event")
	}
}

func TestBusAdapter_PublishInboundRejectsNilDispatcher(t *testing.T) {
	adapter := NewBusAdapter(nil)
	err := adapter.PublishInboundMessage("trace-empty", gateway.InboundEvent{})
	if !errors.Is(err, ErrEventBusUnavailable) {
		t.Fatalf("error = %v, want ErrEventBusUnavailable", err)
	}
}
