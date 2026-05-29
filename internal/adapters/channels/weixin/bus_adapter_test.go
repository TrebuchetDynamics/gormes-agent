package weixin

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
	err := adapter.PublishInboundMessage("trace-weixin-msg-1", gateway.InboundEvent{
		Platform:  "weixin",
		ChatID:    "wxid_chat",
		ChatType:  ChatTypeDirect,
		UserID:    "wxid_user",
		UserName:  "Alice",
		MsgID:     "msg-1",
		MessageID: "msg-1",
		Kind:      gateway.EventSubmit,
		Text:      "hello from Weixin",
	})
	if err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	select {
	case got := <-delivered:
		if got.Type != gateway.TopicMessageReceived {
			t.Fatalf("event type = %q, want %q", got.Type, gateway.TopicMessageReceived)
		}
		if got.Source != "weixin" || got.TraceID != "trace-weixin-msg-1" {
			t.Fatalf("event provenance = source:%q trace:%q, want weixin/trace-weixin-msg-1", got.Source, got.TraceID)
		}
		var payload gateway.MessageEventPayload
		if err := json.Unmarshal(got.Payload, &payload); err != nil {
			t.Fatalf("payload decode: %v", err)
		}
		if payload.Platform != "weixin" || payload.ChatID != "wxid_chat" || payload.ChatType != ChatTypeDirect || payload.UserID != "wxid_user" {
			t.Fatalf("payload provenance = %+v, want weixin chat/chat_type/user", payload)
		}
		if payload.UserName != "Alice" || payload.MessageID != "msg-1" || payload.MsgID != "msg-1" {
			t.Fatalf("payload message provenance = %+v, want user name and message IDs", payload)
		}
		if payload.Kind != "submit" || payload.Text != "hello from Weixin" || payload.Body != "hello from Weixin" {
			t.Fatalf("payload body = kind:%q text:%q body:%q, want submit/Weixin body", payload.Kind, payload.Text, payload.Body)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Weixin bus event")
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
		ChatType:     ChatTypeGroup,
		ChatID:       "group-1",
		UserID:       "user-1",
		UserName:     "Alice",
		MessageID:    "msg-1",
		Text:         "/status",
		ContextToken: "ctx-group",
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
		if got.TraceID != "weixin:group-1:user-1:msg-1" {
			t.Fatalf("trace ID = %q, want weixin:group-1:user-1:msg-1", got.TraceID)
		}
		var payload gateway.MessageEventPayload
		if err := json.Unmarshal(got.Payload, &payload); err != nil {
			t.Fatalf("payload decode: %v", err)
		}
		if payload.Kind != "status" || payload.Text != "" || payload.Body != "" {
			t.Fatalf("payload command = kind:%q text:%q body:%q, want parsed status command", payload.Kind, payload.Text, payload.Body)
		}
		if payload.ChatType != ChatTypeGroup || payload.MessageID != "msg-1" || payload.MsgID != "msg-1" {
			t.Fatalf("payload provenance = %+v, want bot-derived Weixin route", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Weixin fallback-trace bus event")
	}
}

func TestBusAdapter_PublishInboundRejectsNilDispatcher(t *testing.T) {
	adapter := NewBusAdapter(nil)
	err := adapter.PublishInboundMessage("trace-empty", gateway.InboundEvent{})
	if !errors.Is(err, ErrEventBusUnavailable) {
		t.Fatalf("error = %v, want ErrEventBusUnavailable", err)
	}
}
