// Package telegram adapts Telegram bot traffic into kernel.PlatformEvent
// and kernel.RenderFrame streams. The adapter is a sibling to internal/tui —
// both consume the same kernel contracts; neither mutates kernel state.
package telegram

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// telegramClient is the minimal Telegram surface the adapter uses. Production
// wraps *tgbotapi.BotAPI; tests use the mockClient in mock_test.go. Keeping
// this interface tight means the Bot code never pulls a live HTTP dep into
// a test binary.
type telegramClient interface {
	// Token returns the bot token identity used only for local same-token
	// startup locks. Implementations must not log or persist the raw value.
	Token() string

	// GetUpdatesChan starts long-poll and returns the Updates channel.
	GetUpdatesChan(cfg tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel

	// GetUpdates performs one long-poll request. Bot.Run uses this instead of
	// the SDK channel helper so startup can classify polling conflicts and
	// retryable network failures before they disappear into SDK logs.
	GetUpdates(ctx context.Context, cfg tgbotapi.UpdateConfig) ([]tgbotapi.Update, error)

	// Send sends OR edits depending on the Chattable type (NewMessage vs
	// NewEditMessageText). Returns the resulting Message; edit calls return
	// an effectively-ignored Message with the same ID.
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)

	// Request sends non-message Telegram API requests such as setMyCommands.
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)

	// UploadFiles sends a Telegram file endpoint with explicit params. It is
	// used for Bot API fields not exposed by this SDK version's config structs.
	UploadFiles(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error)

	// DeleteMessage removes a bot-posted message through the Bot API
	// deleteMessage request path.
	DeleteMessage(chatID int64, messageID int) error

	// GetFile resolves Telegram file metadata for an attachment file_id.
	GetFile(config tgbotapi.FileConfig) (tgbotapi.File, error)

	// DownloadFile downloads a Telegram file path into memory without exposing
	// token-bearing URLs to callers or logs.
	DownloadFile(ctx context.Context, filePath string) ([]byte, error)

	// StopReceivingUpdates signals the long-poll loop to stop. Called on
	// graceful shutdown.
	StopReceivingUpdates()
}
