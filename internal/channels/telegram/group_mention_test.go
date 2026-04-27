package telegram

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestTelegramGroupMentionGate_BotCommandWithMatchingSuffixAllowsCommand(t *testing.T) {
	text := "/status@gormes_bot"
	entities := []tgbotapi.MessageEntity{botCommandEntity(text)}

	for _, username := range []string{"gormes_bot", "@gormes_bot"} {
		t.Run(username, func(t *testing.T) {
			addressed := telegramGroupMentionGateAddressed(text, entities, username, true)
			if !addressed {
				t.Fatalf("addressed = false, want true")
			}
		})
	}
}

func TestTelegramGroupMentionGate_OtherBotSuffixRejected(t *testing.T) {
	text := "/status@other_bot"
	entities := []tgbotapi.MessageEntity{botCommandEntity(text)}

	addressed := telegramGroupMentionGateAddressed(text, entities, "gormes_bot", true)
	if addressed {
		t.Fatalf("addressed = true, want false")
	}
}

func TestTelegramGroupMentionGate_BareCommandStillGated(t *testing.T) {
	text := "/status"
	entities := []tgbotapi.MessageEntity{botCommandEntity(text)}

	if addressed := telegramGroupMentionGateAddressed(text, entities, "gormes_bot", true); addressed {
		t.Fatalf("requireMention=true addressed = true, want false")
	}
	if addressed := telegramGroupMentionGateAddressed(text, entities, "gormes_bot", false); !addressed {
		t.Fatalf("requireMention=false addressed = false, want true")
	}
}

func TestTelegramGroupMentionGate_MentionEntityStillAllowsText(t *testing.T) {
	text := "hello @gormes_bot"
	entities := []tgbotapi.MessageEntity{{
		Type:   "mention",
		Offset: len("hello "),
		Length: len("@gormes_bot"),
	}}

	addressed := telegramGroupMentionGateAddressed(text, entities, "gormes_bot", true)
	if !addressed {
		t.Fatalf("addressed = false, want true")
	}
}

func botCommandEntity(command string) tgbotapi.MessageEntity {
	return tgbotapi.MessageEntity{
		Type:   "bot_command",
		Offset: 0,
		Length: len(command),
	}
}
