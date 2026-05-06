package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"

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

func (r *realClient) Token() string {
	if r == nil || r.api == nil {
		return ""
	}
	return r.api.Token
}

func (r *realClient) GetUpdatesChan(cfg tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return r.api.GetUpdatesChan(cfg)
}

func (r *realClient) GetUpdates(ctx context.Context, cfg tgbotapi.UpdateConfig) ([]tgbotapi.Update, error) {
	type result struct {
		updates []tgbotapi.Update
		err     error
	}
	done := make(chan result, 1)
	go func() {
		updates, err := r.api.GetUpdates(cfg)
		done <- result{updates: updates, err: err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-done:
		return res.updates, res.err
	}
}

func (r *realClient) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	return r.api.Send(c)
}

func (r *realClient) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return r.api.Request(c)
}

func (r *realClient) UploadFiles(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
	return r.api.UploadFiles(endpoint, params, files)
}

func (r *realClient) DeleteMessage(chatID int64, messageID int) error {
	_, err := r.api.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
	return err
}

func (r *realClient) GetFile(config tgbotapi.FileConfig) (tgbotapi.File, error) {
	return r.api.GetFile(config)
}

func (r *realClient) DownloadFile(ctx context.Context, filePath string) ([]byte, error) {
	file := tgbotapi.File{FilePath: filePath}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.Link(r.api.Token), nil)
	if err != nil {
		return nil, fmt.Errorf("telegram: create download request: %w", err)
	}
	resp, err := r.api.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram: download file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("telegram: download file status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("telegram: read file: %w", err)
	}
	return data, nil
}

func (r *realClient) StopReceivingUpdates() {
	r.api.StopReceivingUpdates()
}
