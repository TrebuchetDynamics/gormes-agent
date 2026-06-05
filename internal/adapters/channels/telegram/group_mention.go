package telegram

import (
	telegrammention "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/mention"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func telegramIsGroupChat(chat *tgbotapi.Chat) bool {
	return telegrammention.IsGroupChat(chat)
}

func telegramGroupMentionGateAddressed(text string, entities []tgbotapi.MessageEntity, expectedBotUsername string, requireMention bool) bool {
	return telegrammention.GroupMentionGateAddressed(text, entities, expectedBotUsername, requireMention)
}

func telegramGroupMentionGateAddressedForBot(text string, entities []tgbotapi.MessageEntity, expectedBotUsername string, expectedBotUserID int64, requireMention bool) bool {
	return telegrammention.GroupMentionGateAddressedForBot(text, entities, expectedBotUsername, expectedBotUserID, requireMention)
}

func telegramGroupMentionGateMessageAddressed(message *tgbotapi.Message, expectedBotUsername string, expectedBotUserID int64, requireMention bool) bool {
	return telegrammention.GroupMentionGateMessageAddressed(message, expectedBotUsername, expectedBotUserID, requireMention)
}

func normalizeTelegramBotUsername(username string) string {
	return telegrammention.NormalizeBotUsername(username)
}

func telegramEntityText(text string, entity tgbotapi.MessageEntity) (string, bool) {
	return telegrammention.EntityText(text, entity)
}

func utf16EntityByteRange(text string, offset, length int) (int, int, bool) {
	return telegrammention.UTF16EntityByteRange(text, offset, length)
}
