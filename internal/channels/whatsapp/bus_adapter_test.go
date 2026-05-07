package whatsapp

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/events"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBusAdapter_PublishInboundMessageReceived(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	delivered := make(chan events.Event, 1)
	bus.Subscribe(gateway.TopicMessageReceived, func(e events.Event) {
		delivered <- e
	})

	normalized := NormalizeInboundWithIdentity(InboundMessage{
		ChatID:    "120363025000000000@g.us",
		ChatName:  "Ops Room",
		ChatKind:  ChatKindGroup,
		UserID:    "15557654321@s.whatsapp.net",
		UserName:  "Bob",
		MessageID: "wamid-2",
		Text:      "@Gormes deploy status",
		Mentioned: true,
	}, IdentityContext{})
	if !normalized.Routed() {
		t.Fatalf("NormalizeInboundWithIdentity decision = %q, want route", normalized.Decision)
	}

	adapter := NewBusAdapter(gateway.NewEventDispatcher(bus))
	err := adapter.PublishInboundMessage("trace-whatsapp-wamid-2", normalized.Event)
	if err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	select {
	case got := <-delivered:
		if got.Type != gateway.TopicMessageReceived {
			t.Fatalf("event type = %q, want %q", got.Type, gateway.TopicMessageReceived)
		}
		if got.Source != "whatsapp" || got.TraceID != "trace-whatsapp-wamid-2" {
			t.Fatalf("event provenance = source:%q trace:%q, want whatsapp/trace-whatsapp-wamid-2", got.Source, got.TraceID)
		}
		var payload gateway.MessageEventPayload
		if err := json.Unmarshal(got.Payload, &payload); err != nil {
			t.Fatalf("payload decode: %v", err)
		}
		if payload.Platform != "whatsapp" || payload.ChatID != "120363025000000000" || payload.ChatType != "group" {
			t.Fatalf("payload chat = platform:%q chat:%q type:%q, want whatsapp/group provenance", payload.Platform, payload.ChatID, payload.ChatType)
		}
		if payload.UserID != "15557654321" || payload.UserName != "Bob" || payload.MessageID != "wamid-2" || payload.MsgID != "wamid-2" {
			t.Fatalf("payload user/message = user:%q name:%q message:%q msg:%q, want canonical WhatsApp provenance", payload.UserID, payload.UserName, payload.MessageID, payload.MsgID)
		}
		if payload.Kind != "submit" || payload.Text != "deploy status" || payload.Body != "deploy status" {
			t.Fatalf("payload body = kind:%q text:%q body:%q, want submit/deploy status", payload.Kind, payload.Text, payload.Body)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WhatsApp bus event")
	}
}

func TestBusAdapter_PublishInboundFallbackTraceID(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	delivered := make(chan events.Event, 1)
	bus.Subscribe(gateway.TopicMessageReceived, func(e events.Event) {
		delivered <- e
	})

	adapter := NewBusAdapter(gateway.NewEventDispatcher(bus))
	err := adapter.PublishInboundMessage("", gateway.InboundEvent{
		Platform:  "whatsapp",
		ChatID:    "15551234567",
		UserID:    "15551234567",
		MessageID: "wamid-1",
		Kind:      gateway.EventSubmit,
		Text:      "hello from WhatsApp",
	})
	if err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	select {
	case got := <-delivered:
		if got.TraceID != "whatsapp:15551234567:15551234567:wamid-1" {
			t.Fatalf("trace ID = %q, want WhatsApp chat/user/message provenance", got.TraceID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WhatsApp bus event")
	}
}

func TestBusAdapter_PublishInboundRejectsNilDispatcher(t *testing.T) {
	adapter := NewBusAdapter(nil)
	err := adapter.PublishInboundMessage("trace-empty", gateway.InboundEvent{})
	if !errors.Is(err, ErrEventBusUnavailable) {
		t.Fatalf("error = %v, want ErrEventBusUnavailable", err)
	}
}
