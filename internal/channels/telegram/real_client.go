package telegram

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// realClient wraps *tgbotapi.BotAPI to satisfy telegramClient. Every method
// is a thin passthrough — testable behaviour stays in Bot and coalescer,
// which talk to the telegramClient interface.
type realClient struct {
	api *tgbotapi.BotAPI
}

var _ telegramClient = (*realClient)(nil)

// NewRealClient constructs a realClient from a bot token. Fails if the
// token is invalid (tgbotapi validates by calling getMe on construction),
// so token errors surface at binary startup not at first user message.
//
// Exported so cmd/gormes (telegram subcommand) can construct one outside this package.
func NewRealClient(token string) (telegramClient, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("telegram: invalid token: %w", err)
	}
	return &realClient{api: api}, nil
}

func (r *realClient) GetUpdatesChan(cfg tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return r.api.GetUpdatesChan(cfg)
}

func (r *realClient) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	return r.api.Send(c)
}

func (r *realClient) DeleteMessage(chatID int64, messageID int) error {
	_, err := r.api.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
	return err
}

func (r *realClient) StopReceivingUpdates() {
	r.api.StopReceivingUpdates()
}
