// Package telegram adapts Telegram bot traffic into kernel.PlatformEvent
// and kernel.RenderFrame streams. The adapter is a sibling to internal/tui —
// both consume the same kernel contracts; neither mutates kernel state.
package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// telegramClient is the minimal Telegram surface the adapter uses. Production
// wraps *tgbotapi.BotAPI; tests use the mockClient in mock_test.go. Keeping
// this interface tight means the Bot code never pulls a live HTTP dep into
// a test binary.
type telegramClient interface {
	// GetUpdatesChan starts long-poll and returns the Updates channel.
	GetUpdatesChan(cfg tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel

	// Send sends OR edits depending on the Chattable type (NewMessage vs
	// NewEditMessageText). Returns the resulting Message; edit calls return
	// an effectively-ignored Message with the same ID.
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)

	// DeleteMessage removes a bot-posted message through the Bot API
	// deleteMessage request path.
	DeleteMessage(chatID int64, messageID int) error

	// StopReceivingUpdates signals the long-poll loop to stop. Called on
	// graceful shutdown.
	StopReceivingUpdates()
}
