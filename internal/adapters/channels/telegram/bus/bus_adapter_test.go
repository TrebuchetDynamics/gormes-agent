package bus

import (
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/internal/adaptertest"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBusAdapter_PublishInboundMessageReceived(t *testing.T) {
	probe := adaptertest.NewMessageEventProbe(t)

	adapter := NewBusAdapter(probe.Dispatcher)
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

	got := probe.Next(t, "Telegram bus event")
	adaptertest.RequireMessageReceived(t, got, "telegram", "trace-tg-99")
	payload := adaptertest.DecodeMessagePayload(t, got)
	if payload.Platform != "telegram" || payload.ChatID != "42" || payload.UserID != "7" || payload.MessageID != "99" {
		t.Fatalf("payload provenance = %+v, want telegram chat/user/message", payload)
	}
	if payload.Kind != "submit" || payload.Text != "hello" || payload.Body != "hello" {
		t.Fatalf("payload body = kind:%q text:%q body:%q, want submit/hello/hello", payload.Kind, payload.Text, payload.Body)
	}
	if payload.ThreadID != "1" || payload.ReplyToMessageID != "" {
		t.Fatalf("payload thread/reply = thread:%q reply:%q, want 1/empty", payload.ThreadID, payload.ReplyToMessageID)
	}
}

func TestBusAdapter_PublishInboundRejectsNilDispatcher(t *testing.T) {
	adapter := NewBusAdapter(nil)
	err := adapter.PublishInboundMessage("trace-empty", gateway.InboundEvent{})
	if !errors.Is(err, ErrEventBusUnavailable) {
		t.Fatalf("error = %v, want ErrEventBusUnavailable", err)
	}
}
