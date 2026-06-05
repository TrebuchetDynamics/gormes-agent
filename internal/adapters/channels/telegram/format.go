package telegram

import telegramformat "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/format"

func stripTelegramMarkdownV2(text string) string {
	return telegramformat.StripMarkdownV2(text)
}
