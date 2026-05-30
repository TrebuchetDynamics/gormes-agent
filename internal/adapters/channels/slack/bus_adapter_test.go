package slack

import (
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/internal/adaptertest"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBusAdapter_PublishInboundMessageReceived(t *testing.T) {
	probe := adaptertest.NewMessageEventProbe(t)

	ch := NewChannel(newMockClient(), nil)
	ev, ok := ch.toInboundEvent(Event{
		ChannelID: "C123",
		TeamID:    "T123",
		UserID:    "U1",
		Text:      "  deploy status  ",
		Timestamp: "1711111111.000200",
		ThreadTS:  "1711111111.000100",
	})
	if !ok {
		t.Fatal("toInboundEvent dropped Slack event, want submit")
	}

	adapter := NewBusAdapter(probe.Dispatcher)
	err := adapter.PublishInboundMessage("trace-slack-1711111111", ev)
	if err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	got := probe.Next(t, "Slack bus event")
	adaptertest.RequireMessageReceived(t, got, "slack", "trace-slack-1711111111")
	payload := adaptertest.DecodeMessagePayload(t, got)
	if payload.Platform != "slack" || payload.AccountID != "T123" || payload.ChatID != "C123" || payload.UserID != "U1" || payload.MessageID != "1711111111.000200" {
		t.Fatalf("payload provenance = %+v, want slack team/channel/user/message", payload)
	}
	if payload.Kind != "submit" || payload.Text != "deploy status" || payload.Body != "deploy status" {
		t.Fatalf("payload body = kind:%q text:%q body:%q, want submit/deploy status", payload.Kind, payload.Text, payload.Body)
	}
	if payload.ThreadID != "1711111111.000100" || payload.ReplyToMessageID != "" {
		t.Fatalf("payload thread/reply = thread:%q reply:%q, want Slack thread/empty reply", payload.ThreadID, payload.ReplyToMessageID)
	}
}

func TestBusAdapter_PublishInboundFallbackTraceID(t *testing.T) {
	probe := adaptertest.NewMessageEventProbe(t)

	adapter := NewBusAdapter(probe.Dispatcher)
	err := adapter.PublishInboundMessage("", gateway.InboundEvent{
		Platform:  "slack",
		AccountID: "T123",
		ChatID:    "C123",
		UserID:    "U1",
		ThreadID:  "1711111111.000100",
		MessageID: "1711111111.000200",
		Kind:      gateway.EventSubmit,
		Text:      "hello from Slack",
	})
	if err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	got := probe.Next(t, "Slack bus event")
	if got.TraceID != "slack:T123:C123:1711111111.000100:1711111111.000200" {
		t.Fatalf("trace ID = %q, want Slack team/channel/thread/message provenance", got.TraceID)
	}
}

func TestBusAdapter_PublishInboundRejectsNilDispatcher(t *testing.T) {
	adapter := NewBusAdapter(nil)
	err := adapter.PublishInboundMessage("trace-empty", gateway.InboundEvent{})
	if !errors.Is(err, ErrEventBusUnavailable) {
		t.Fatalf("error = %v, want ErrEventBusUnavailable", err)
	}
}
