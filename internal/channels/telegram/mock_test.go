package telegram

import (
	"context"
	"errors"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var errTelegramTestDownload = errors.New("telegram download failed")

type mockClient struct {
	updatesCh chan tgbotapi.Update
	mu        sync.Mutex
	sent      []tgbotapi.Chattable
	requests  []tgbotapi.Chattable
	deleted   []tgbotapi.Chattable
	nextMsgID int
	stopped   bool

	SendFn          func(c tgbotapi.Chattable) (tgbotapi.Message, error)
	RequestFn       func(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	DeleteMessageFn func(chatID int64, messageID int) error
	telegramFiles   map[string]tgbotapi.File
	downloads       map[string][]byte
	downloadCalls   int
	getFileErr      error
	downloadErr     error
}

var _ telegramClient = (*mockClient)(nil)

func newMockClient() *mockClient {
	return &mockClient{
		updatesCh:     make(chan tgbotapi.Update, 16),
		nextMsgID:     1000,
		telegramFiles: map[string]tgbotapi.File{},
		downloads:     map[string][]byte{},
	}
}

func (m *mockClient) GetUpdatesChan(_ tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return m.updatesCh
}

func (m *mockClient) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.mu.Lock()
	m.sent = append(m.sent, c)
	id := m.nextMsgID
	m.nextMsgID++
	m.mu.Unlock()

	if m.SendFn != nil {
		return m.SendFn(c)
	}
	return tgbotapi.Message{MessageID: id}, nil
}

func (m *mockClient) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	m.mu.Lock()
	m.requests = append(m.requests, c)
	m.mu.Unlock()
	if m.RequestFn != nil {
		return m.RequestFn(c)
	}
	return &tgbotapi.APIResponse{Ok: true}, nil
}

func (m *mockClient) DeleteMessage(chatID int64, messageID int) error {
	req := tgbotapi.NewDeleteMessage(chatID, messageID)
	m.mu.Lock()
	m.deleted = append(m.deleted, req)
	m.mu.Unlock()

	if m.DeleteMessageFn != nil {
		return m.DeleteMessageFn(chatID, messageID)
	}
	return nil
}

func (m *mockClient) GetFile(config tgbotapi.FileConfig) (tgbotapi.File, error) {
	if m.getFileErr != nil {
		return tgbotapi.File{}, m.getFileErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if f, ok := m.telegramFiles[config.FileID]; ok {
		return f, nil
	}
	return tgbotapi.File{FileID: config.FileID, FilePath: config.FileID}, nil
}

func (m *mockClient) DownloadFile(_ context.Context, filePath string) ([]byte, error) {
	m.mu.Lock()
	m.downloadCalls++
	m.mu.Unlock()
	if m.downloadErr != nil {
		return nil, m.downloadErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if data, ok := m.downloads[filePath]; ok {
		return append([]byte(nil), data...), nil
	}
	return nil, errTelegramTestDownload
}

func (m *mockClient) StopReceivingUpdates() {
	m.mu.Lock()
	m.stopped = true
	m.mu.Unlock()
}

func (m *mockClient) closeUpdates() {
	close(m.updatesCh)
}

func (m *mockClient) pushTextUpdate(chatID int64, text string) {
	m.updatesCh <- tgbotapi.Update{
		UpdateID: 0,
		Message: &tgbotapi.Message{
			MessageID: 1,
			Text:      text,
			Chat:      &tgbotapi.Chat{ID: chatID},
			From:      &tgbotapi.User{ID: chatID, FirstName: "tester"},
		},
	}
}

func (m *mockClient) pushVoiceUpdate(chatID int64, voice tgbotapi.Voice) {
	m.updatesCh <- tgbotapi.Update{
		UpdateID: 0,
		Message: &tgbotapi.Message{
			MessageID: 1,
			Voice:     &voice,
			Chat:      &tgbotapi.Chat{ID: chatID},
			From:      &tgbotapi.User{ID: chatID, FirstName: "tester"},
		},
	}
}

func (m *mockClient) pushAudioUpdate(chatID int64, caption string, audio tgbotapi.Audio) {
	m.updatesCh <- tgbotapi.Update{
		UpdateID: 0,
		Message: &tgbotapi.Message{
			MessageID: 1,
			Caption:   caption,
			Audio:     &audio,
			Chat:      &tgbotapi.Chat{ID: chatID},
			From:      &tgbotapi.User{ID: chatID, FirstName: "tester"},
		},
	}
}

func (m *mockClient) pushDocumentUpdate(chatID int64, messageID int, caption string, document tgbotapi.Document) {
	m.updatesCh <- tgbotapi.Update{
		UpdateID: 0,
		Message: &tgbotapi.Message{
			MessageID: messageID,
			Caption:   caption,
			Document:  &document,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "private"},
			From:      &tgbotapi.User{ID: chatID, FirstName: "tester"},
		},
	}
}

func (m *mockClient) pushPhotoUpdate(chatID int64, messageID int, caption, mediaGroupID string, photos []tgbotapi.PhotoSize) {
	m.updatesCh <- tgbotapi.Update{
		UpdateID: 0,
		Message: &tgbotapi.Message{
			MessageID:    messageID,
			Caption:      caption,
			Photo:        photos,
			MediaGroupID: mediaGroupID,
			Chat:         &tgbotapi.Chat{ID: chatID, Type: "private"},
			From:         &tgbotapi.User{ID: chatID, FirstName: "tester"},
		},
	}
}

func (m *mockClient) pushVideoUpdate(chatID int64, messageID int, caption string, video tgbotapi.Video) {
	m.updatesCh <- tgbotapi.Update{
		UpdateID: 0,
		Message: &tgbotapi.Message{
			MessageID: messageID,
			Caption:   caption,
			Video:     &video,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "private"},
			From:      &tgbotapi.User{ID: chatID, FirstName: "tester"},
		},
	}
}

func (m *mockClient) downloadCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.downloadCalls
}

func (m *mockClient) sentMessages() []tgbotapi.Chattable {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]tgbotapi.Chattable, len(m.sent))
	copy(out, m.sent)
	return out
}

func (m *mockClient) requestMessages() []tgbotapi.Chattable {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]tgbotapi.Chattable, len(m.requests))
	copy(out, m.requests)
	return out
}

func (m *mockClient) deleteRequests() []tgbotapi.Chattable {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]tgbotapi.Chattable, len(m.deleted))
	copy(out, m.deleted)
	return out
}

func (m *mockClient) lastSentText() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		return ""
	}
	last := m.sent[len(m.sent)-1]
	switch v := last.(type) {
	case tgbotapi.MessageConfig:
		return v.Text
	case tgbotapi.EditMessageTextConfig:
		return v.Text
	}
	return ""
}
