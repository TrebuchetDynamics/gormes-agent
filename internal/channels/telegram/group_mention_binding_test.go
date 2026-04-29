package telegram

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBot_ToInboundEvent_GroupMentionGate_BareCommandDropped(t *testing.T) {
	b := New(Config{RequireMention: true, BotUsername: "gormes_bot"}, newMockClient(), nil)
	update := groupUpdate(-100, "/status", []tgbotapi.MessageEntity{
		{Type: "bot_command", Offset: 0, Length: 7},
	})
	if _, ok := b.toInboundEvent(update); ok {
		t.Fatal("expected bare /status in group to be dropped when require_mention=true")
	}
}

func TestBot_ToInboundEvent_GroupMentionGate_BotCommandSuffixAccepted(t *testing.T) {
	b := New(Config{RequireMention: true, BotUsername: "gormes_bot"}, newMockClient(), nil)
	update := groupUpdate(-100, "/start@gormes_bot", []tgbotapi.MessageEntity{
		{Type: "bot_command", Offset: 0, Length: 17},
	})
	ev, ok := b.toInboundEvent(update)
	if !ok {
		t.Fatal("expected /start@gormes_bot in group to be accepted")
	}
	if ev.Platform != "telegram" {
		t.Fatalf("Platform = %q, want telegram", ev.Platform)
	}
}

func TestBot_ToInboundEvent_GroupMentionGate_PlainTextMentionAccepted(t *testing.T) {
	b := New(Config{RequireMention: true, BotUsername: "gormes_bot"}, newMockClient(), nil)
	text := "hello @gormes_bot"
	update := groupUpdate(-100, text, []tgbotapi.MessageEntity{
		{Type: "mention", Offset: len("hello "), Length: len("@gormes_bot")},
	})
	ev, ok := b.toInboundEvent(update)
	if !ok {
		t.Fatal("expected mention-addressed group text to be accepted")
	}
	if ev.Kind != gateway.EventSubmit {
		t.Fatalf("Kind = %v, want EventSubmit", ev.Kind)
	}
}


func TestBot_ToInboundEvent_DMBypassesMentionGate(t *testing.T) {
	b := New(Config{RequireMention: true, BotUsername: "gormes_bot"}, newMockClient(), nil)
	update := dmUpdate(42, "hello")
	ev, ok := b.toInboundEvent(update)
	if !ok {
		t.Fatal("expected DM text to be accepted regardless of mention gate")
	}
	if ev.Kind != gateway.EventSubmit {
		t.Fatalf("Kind = %v, want EventSubmit for DM", ev.Kind)
	}
}

func groupUpdate(chatID int64, text string, entities []tgbotapi.MessageEntity) tgbotapi.Update {
	return tgbotapi.Update{
		UpdateID: 0,
		Message: &tgbotapi.Message{
			MessageID: 1,
			Text:      text,
			Entities:  entities,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "supergroup"},
			From:      &tgbotapi.User{ID: 99, FirstName: "tester"},
		},
	}
}

func dmUpdate(chatID int64, text string) tgbotapi.Update {
	return tgbotapi.Update{
		UpdateID: 0,
		Message: &tgbotapi.Message{
			MessageID: 1,
			Text:      text,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "private"},
			From:      &tgbotapi.User{ID: chatID, FirstName: "tester"},
		},
	}
}
