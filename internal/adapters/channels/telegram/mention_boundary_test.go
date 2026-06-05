package telegram

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestTelegramMentionBoundary_CaptionMentionAccepted(t *testing.T) {
	b := New(Config{RequireMention: true, BotUsername: "gormes_bot"}, newMockClient(), nil)
	caption := "photo for @gormes_bot"
	update := groupCaptionUpdate(-100, caption, []tgbotapi.MessageEntity{
		{Type: "mention", Offset: len("photo for "), Length: len("@gormes_bot")},
	})

	if _, ok := b.toInboundEvent(context.Background(), update); !ok {
		t.Fatal("expected caption mention entity to address the bot")
	}
}

func TestTelegramMentionBoundary_TextMentionTargetsConfiguredBotUserID(t *testing.T) {
	b := New(Config{RequireMention: true, BotUsername: "gormes_bot", BotUserID: 999}, newMockClient(), nil)
	text := "hey you"
	update := groupUpdate(-100, text, []tgbotapi.MessageEntity{
		{Type: "text_mention", Offset: len("hey "), Length: len("you"), User: &tgbotapi.User{ID: 999}},
	})

	if _, ok := b.toInboundEvent(context.Background(), update); !ok {
		t.Fatal("expected text_mention entity targeting the configured bot user ID to address the bot")
	}
}

func TestTelegramMentionBoundary_TextMentionDoesNotRequireUsername(t *testing.T) {
	b := New(Config{RequireMention: true, BotUserID: 999}, newMockClient(), nil)
	text := "hey you"
	update := groupUpdate(-100, text, []tgbotapi.MessageEntity{
		{Type: "text_mention", Offset: len("hey "), Length: len("you"), User: &tgbotapi.User{ID: 999}},
	})

	if _, ok := b.toInboundEvent(context.Background(), update); !ok {
		t.Fatal("expected text_mention entity to use bot user ID even when no username is configured")
	}
}

func TestTelegramMentionBoundary_SubstringFalsePositivesRejected(t *testing.T) {
	b := New(Config{RequireMention: true, BotUsername: "gormes_bot"}, newMockClient(), nil)
	for _, text := range []string{
		"email me at foo@gormes_bot.example",
		"contact user@gormes_bot.domain.com",
		"see @gormes_bot_admin for help",
		"see https://example.com/@gormes_bot for details",
		"use the string `@gormes_bot` in config",
	} {
		t.Run(text, func(t *testing.T) {
			update := groupUpdate(-100, text, nil)
			if _, ok := b.toInboundEvent(context.Background(), update); ok {
				t.Fatalf("plain substring %q addressed the bot without a mention entity", text)
			}
		})
	}
}

func TestTelegramMentionBoundary_TextMentionDifferentUserRejected(t *testing.T) {
	b := New(Config{RequireMention: true, BotUserID: 999}, newMockClient(), nil)
	text := "hey you"
	update := groupUpdate(-100, text, []tgbotapi.MessageEntity{
		{Type: "text_mention", Offset: len("hey "), Length: len("you"), User: &tgbotapi.User{ID: 12345}},
	})

	if _, ok := b.toInboundEvent(context.Background(), update); ok {
		t.Fatal("text_mention for a different user addressed the bot")
	}
}

func TestTelegramMentionBoundary_MalformedEntitiesFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		entity tgbotapi.MessageEntity
	}{
		{name: "negative offset", entity: tgbotapi.MessageEntity{Type: "mention", Offset: -1, Length: len("@gormes_bot")}},
		{name: "zero length", entity: tgbotapi.MessageEntity{Type: "mention", Offset: 0, Length: 0}},
		{name: "out of range", entity: tgbotapi.MessageEntity{Type: "mention", Offset: 100, Length: len("@gormes_bot")}},
		{name: "wrong username", entity: tgbotapi.MessageEntity{Type: "mention", Offset: 0, Length: len("@other_bot")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := "@gormes_bot hi"
			if tc.name == "wrong username" {
				text = "@other_bot hi"
			}
			update := groupUpdate(-100, text, []tgbotapi.MessageEntity{tc.entity})
			b := New(Config{RequireMention: true, BotUsername: "gormes_bot"}, newMockClient(), nil)
			if _, ok := b.toInboundEvent(context.Background(), update); ok {
				t.Fatalf("%s entity addressed the bot", tc.name)
			}
		})
	}
}

func groupCaptionUpdate(chatID int64, caption string, captionEntities []tgbotapi.MessageEntity) tgbotapi.Update {
	return tgbotapi.Update{
		UpdateID: 0,
		Message: &tgbotapi.Message{
			MessageID:       1,
			Caption:         caption,
			CaptionEntities: captionEntities,
			Chat:            &tgbotapi.Chat{ID: chatID, Type: "supergroup"},
			From:            &tgbotapi.User{ID: 99, FirstName: "tester"},
			Photo:           []tgbotapi.PhotoSize{{FileID: "photo-file-id", FileUniqueID: "photo-unique-id"}},
		},
	}
}
