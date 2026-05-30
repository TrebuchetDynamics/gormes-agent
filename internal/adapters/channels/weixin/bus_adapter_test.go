package weixin

import (
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/internal/adaptertest"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBusAdapter_PublishInboundMessageReceived(t *testing.T) {
	probe := adaptertest.NewMessageEventProbe(t)

	adapter := NewBusAdapter(probe.Dispatcher)
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

	got := probe.Next(t, "Weixin bus event")
	adaptertest.RequireMessageReceived(t, got, "weixin", "trace-weixin-msg-1")
	payload := adaptertest.DecodeMessagePayload(t, got)
	if payload.Platform != "weixin" || payload.ChatID != "wxid_chat" || payload.ChatType != ChatTypeDirect || payload.UserID != "wxid_user" {
		t.Fatalf("payload provenance = %+v, want weixin chat/chat_type/user", payload)
	}
	if payload.UserName != "Alice" || payload.MessageID != "msg-1" || payload.MsgID != "msg-1" {
		t.Fatalf("payload message provenance = %+v, want user name and message IDs", payload)
	}
	if payload.Kind != "submit" || payload.Text != "hello from Weixin" || payload.Body != "hello from Weixin" {
		t.Fatalf("payload body = kind:%q text:%q body:%q, want submit/Weixin body", payload.Kind, payload.Text, payload.Body)
	}
}

func TestBusAdapter_PublishInboundFallbackTraceID(t *testing.T) {
	probe := adaptertest.NewMessageEventProbe(t)

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

	adapter := NewBusAdapter(probe.Dispatcher)
	if err := adapter.PublishInboundMessage("", ev); err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	got := probe.Next(t, "Weixin fallback-trace bus event")
	if got.TraceID != "weixin:group-1:user-1:msg-1" {
		t.Fatalf("trace ID = %q, want weixin:group-1:user-1:msg-1", got.TraceID)
	}
	payload := adaptertest.DecodeMessagePayload(t, got)
	if payload.Kind != "status" || payload.Text != "" || payload.Body != "" {
		t.Fatalf("payload command = kind:%q text:%q body:%q, want parsed status command", payload.Kind, payload.Text, payload.Body)
	}
	if payload.ChatType != ChatTypeGroup || payload.MessageID != "msg-1" || payload.MsgID != "msg-1" {
		t.Fatalf("payload provenance = %+v, want bot-derived Weixin route", payload)
	}
}

func TestBusAdapter_PublishInboundRejectsNilDispatcher(t *testing.T) {
	adapter := NewBusAdapter(nil)
	err := adapter.PublishInboundMessage("trace-empty", gateway.InboundEvent{})
	if !errors.Is(err, ErrEventBusUnavailable) {
		t.Fatalf("error = %v, want ErrEventBusUnavailable", err)
	}
}
