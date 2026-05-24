package wecom

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
	err := adapter.PublishInboundMessage("trace-wecom-msg-1", gateway.InboundEvent{
		Platform:  "wecom",
		ChatID:    "group-1",
		ChatType:  ChatTypeGroup,
		UserID:    "user-1",
		UserName:  "Alice",
		MsgID:     "msg-1",
		MessageID: "msg-1",
		Kind:      gateway.EventSubmit,
		Text:      "hello from WeCom",
	})
	if err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	select {
	case got := <-delivered:
		if got.Type != gateway.TopicMessageReceived {
			t.Fatalf("event type = %q, want %q", got.Type, gateway.TopicMessageReceived)
		}
		if got.Source != "wecom" || got.TraceID != "trace-wecom-msg-1" {
			t.Fatalf("event provenance = source:%q trace:%q, want wecom/trace-wecom-msg-1", got.Source, got.TraceID)
		}
		var payload gateway.MessageEventPayload
		if err := json.Unmarshal(got.Payload, &payload); err != nil {
			t.Fatalf("payload decode: %v", err)
		}
		if payload.Platform != "wecom" || payload.ChatID != "group-1" || payload.ChatType != ChatTypeGroup || payload.UserID != "user-1" {
			t.Fatalf("payload provenance = %+v, want wecom chat/chat_type/user", payload)
		}
		if payload.UserName != "Alice" || payload.MessageID != "msg-1" || payload.MsgID != "msg-1" {
			t.Fatalf("payload message provenance = %+v, want user name and message IDs", payload)
		}
		if payload.Kind != "submit" || payload.Text != "hello from WeCom" || payload.Body != "hello from WeCom" {
			t.Fatalf("payload body = kind:%q text:%q body:%q, want submit/WeCom body", payload.Kind, payload.Text, payload.Body)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WeCom bus event")
	}
}

func TestBusAdapter_PublishInboundFallbackTraceID(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	delivered := make(chan events.Event, 1)
	bus.Subscribe(gateway.TopicMessageReceived, func(e events.Event) {
		delivered <- e
	})

	bot := New(Config{GroupPolicy: "allowlist", GroupAllowFrom: []string{"group-1"}}, newMockClient(), nil)
	ev, ok := bot.toInboundEvent(InboundMessage{
		ChatType:  ChatTypeGroup,
		ChatID:    "group-1",
		UserID:    "user-1",
		UserName:  "Alice",
		MessageID: "msg-1",
		Text:      "/status",
		RequestID: "req-1",
	})
	if !ok {
		t.Fatal("toInboundEvent returned false")
	}
	if ev.ChatType != ChatTypeGroup || ev.MessageID != "msg-1" || ev.MsgID != "msg-1" {
		t.Fatalf("inbound event provenance = %+v, want chat type and message IDs", ev)
	}

	adapter := NewBusAdapter(gateway.NewEventDispatcher(bus))
	if err := adapter.PublishInboundMessage("", ev); err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	select {
	case got := <-delivered:
		if got.TraceID != "wecom:group-1:user-1:msg-1" {
			t.Fatalf("trace ID = %q, want wecom:group-1:user-1:msg-1", got.TraceID)
		}
		var payload gateway.MessageEventPayload
		if err := json.Unmarshal(got.Payload, &payload); err != nil {
			t.Fatalf("payload decode: %v", err)
		}
		if payload.Kind != "status" || payload.Text != "" || payload.Body != "" {
			t.Fatalf("payload command = kind:%q text:%q body:%q, want parsed status command", payload.Kind, payload.Text, payload.Body)
		}
		if payload.ChatType != ChatTypeGroup || payload.MessageID != "msg-1" || payload.MsgID != "msg-1" {
			t.Fatalf("payload provenance = %+v, want bot-derived WeCom route", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WeCom fallback-trace bus event")
	}
}

func TestBusAdapter_PublishInboundRejectsNilDispatcher(t *testing.T) {
	adapter := NewBusAdapter(nil)
	err := adapter.PublishInboundMessage("trace-empty", gateway.InboundEvent{})
	if !errors.Is(err, ErrEventBusUnavailable) {
		t.Fatalf("error = %v, want ErrEventBusUnavailable", err)
	}
}
