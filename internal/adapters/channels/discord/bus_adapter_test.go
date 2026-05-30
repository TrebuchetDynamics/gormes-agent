package discord

import (
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/internal/adaptertest"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBusAdapter_PublishInboundMessageReceived(t *testing.T) {
	probe := adaptertest.NewMessageEventProbe(t)

	adapter := NewBusAdapter(probe.Dispatcher)
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

	got := probe.Next(t, "Discord bus event")
	adaptertest.RequireMessageReceived(t, got, "discord", "trace-discord-201")
	payload := adaptertest.DecodeMessagePayload(t, got)
	if payload.Platform != "discord" || payload.ChatID != "forum-100" || payload.UserID != "user-7" || payload.MessageID != "msg-201" {
		t.Fatalf("payload provenance = %+v, want discord chat/user/message", payload)
	}
	if payload.Kind != "submit" || payload.Text != "follow up from the forum post" || payload.Body != "follow up from the forum post" {
		t.Fatalf("payload body = kind:%q text:%q body:%q, want submit/forum body", payload.Kind, payload.Text, payload.Body)
	}
	if payload.ThreadID != "thread-200" || payload.ParentChatID != "forum-100" || payload.GuildID != "guild-1" {
		t.Fatalf("payload thread provenance = thread:%q parent:%q guild:%q, want thread-200/forum-100/guild-1", payload.ThreadID, payload.ParentChatID, payload.GuildID)
	}
}

func TestBusAdapter_PublishInboundForumThreadProvenance(t *testing.T) {
	probe := adaptertest.NewMessageEventProbe(t)

	bot := New(Config{AllowedChannelID: "forum-100"}, newMockSession(), nil)
	bot.rememberThread(loadDiscordChannelFixture(t, "forum_thread_create.json"))
	msg := loadDiscordMessageCreateFixture(t, "forum_thread_message.json")
	ev, ok := bot.toInboundEvent(msg.Message)
	if !ok {
		t.Fatal("toInboundEvent returned false")
	}

	adapter := NewBusAdapter(probe.Dispatcher)
	if err := adapter.PublishInboundMessage("", ev); err != nil {
		t.Fatalf("PublishInboundMessage: %v", err)
	}

	got := probe.Next(t, "Discord forum bus event")
	if got.TraceID != "discord:guild-1:forum-100:thread-200:msg-201" {
		t.Fatalf("trace ID = %q, want discord:guild-1:forum-100:thread-200:msg-201", got.TraceID)
	}
	payload := adaptertest.DecodeMessagePayload(t, got)
	if payload.ChatID != "forum-100" || payload.ParentChatID != "forum-100" || payload.ThreadID != "thread-200" || payload.GuildID != "guild-1" {
		t.Fatalf("payload route = chat:%q parent:%q thread:%q guild:%q, want forum/thread/guild provenance", payload.ChatID, payload.ParentChatID, payload.ThreadID, payload.GuildID)
	}
	if payload.MessageID != "msg-201" || payload.MsgID != "msg-201" || payload.Body != "follow up from the forum post" {
		t.Fatalf("payload source/body = message:%q msg:%q body:%q, want msg-201/forum body", payload.MessageID, payload.MsgID, payload.Body)
	}
}

func TestBusAdapter_PublishInboundRejectsNilDispatcher(t *testing.T) {
	adapter := NewBusAdapter(nil)
	err := adapter.PublishInboundMessage("trace-empty", gateway.InboundEvent{})
	if !errors.Is(err, ErrEventBusUnavailable) {
		t.Fatalf("error = %v, want ErrEventBusUnavailable", err)
	}
}
