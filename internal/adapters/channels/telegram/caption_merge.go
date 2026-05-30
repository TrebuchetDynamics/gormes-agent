package telegram

import telegramcontent "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/content"

func telegramMergeCaption(existing, next string) string {
	return telegramcontent.MergeCaption(existing, next)
}
