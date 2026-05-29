package discord

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
	err := adapter.PublishInboundMessage("trace-discord-201", gateway.InboundEvent{
		Platform:     "discord",
		ChatID:       "forum-100",
		UserID:       "user-7",
		ThreadID:     "thread-200",
		MsgID:        "msg-201",
		MessageID:    "msg-201",
		GuildID:      "guild-1",
		ParentChatID: "forum-100",
		Kind:         gateway.EventSubmit,
		Text:         "follow up from the forum post",
	})
	if err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	select {
	case got := <-delivered:
		if got.Type != gateway.TopicMessageReceived {
			t.Fatalf("event type = %q, want %q", got.Type, gateway.TopicMessageReceived)
		}
		if got.Source != "discord" || got.TraceID != "trace-discord-201" {
			t.Fatalf("event provenance = source:%q trace:%q, want discord/trace-discord-201", got.Source, got.TraceID)
		}
		var payload gateway.MessageEventPayload
		if err := json.Unmarshal(got.Payload, &payload); err != nil {
			t.Fatalf("payload decode: %v", err)
		}
		if payload.Platform != "discord" || payload.ChatID != "forum-100" || payload.UserID != "user-7" || payload.MessageID != "msg-201" {
			t.Fatalf("payload provenance = %+v, want discord chat/user/message", payload)
		}
		if payload.Kind != "submit" || payload.Text != "follow up from the forum post" || payload.Body != "follow up from the forum post" {
			t.Fatalf("payload body = kind:%q text:%q body:%q, want submit/forum body", payload.Kind, payload.Text, payload.Body)
		}
		if payload.ThreadID != "thread-200" || payload.ParentChatID != "forum-100" || payload.GuildID != "guild-1" {
			t.Fatalf("payload thread provenance = thread:%q parent:%q guild:%q, want thread-200/forum-100/guild-1", payload.ThreadID, payload.ParentChatID, payload.GuildID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Discord bus event")
	}
}

func TestBusAdapter_PublishInboundForumThreadProvenance(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	delivered := make(chan events.Event, 1)
	bus.Subscribe(gateway.TopicMessageReceived, func(e events.Event) {
		delivered <- e
	})

	bot := New(Config{AllowedChannelID: "forum-100"}, newMockSession(), nil)
	bot.rememberThread(loadDiscordChannelFixture(t, "forum_thread_create.json"))
	msg := loadDiscordMessageCreateFixture(t, "forum_thread_message.json")
	ev, ok := bot.toInboundEvent(msg.Message)
	if !ok {
		t.Fatal("toInboundEvent returned false")
	}

	adapter := NewBusAdapter(gateway.NewEventDispatcher(bus))
	if err := adapter.PublishInboundMessage("", ev); err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	select {
	case got := <-delivered:
		if got.TraceID != "discord:guild-1:forum-100:thread-200:msg-201" {
			t.Fatalf("trace ID = %q, want discord:guild-1:forum-100:thread-200:msg-201", got.TraceID)
		}
		var payload gateway.MessageEventPayload
		if err := json.Unmarshal(got.Payload, &payload); err != nil {
			t.Fatalf("payload decode: %v", err)
		}
		if payload.ChatID != "forum-100" || payload.ParentChatID != "forum-100" || payload.ThreadID != "thread-200" || payload.GuildID != "guild-1" {
			t.Fatalf("payload route = chat:%q parent:%q thread:%q guild:%q, want forum/thread/guild provenance", payload.ChatID, payload.ParentChatID, payload.ThreadID, payload.GuildID)
		}
		if payload.MessageID != "msg-201" || payload.MsgID != "msg-201" || payload.Body != "follow up from the forum post" {
			t.Fatalf("payload source/body = message:%q msg:%q body:%q, want msg-201/forum body", payload.MessageID, payload.MsgID, payload.Body)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Discord forum bus event")
	}
}

func TestBusAdapter_PublishInboundRejectsNilDispatcher(t *testing.T) {
	adapter := NewBusAdapter(nil)
	err := adapter.PublishInboundMessage("trace-empty", gateway.InboundEvent{})
	if !errors.Is(err, ErrEventBusUnavailable) {
		t.Fatalf("error = %v, want ErrEventBusUnavailable", err)
	}
}
