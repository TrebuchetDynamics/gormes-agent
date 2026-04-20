package telegram

import (
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type mockClient struct {
	updatesCh chan tgbotapi.Update
	mu        sync.Mutex
	sent      []tgbotapi.Chattable
	nextMsgID int
	stopped   bool

	SendFn func(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

var _ telegramClient = (*mockClient)(nil)

func newMockClient() *mockClient {
	return &mockClient{
		updatesCh: make(chan tgbotapi.Update, 16),
		nextMsgID: 1000,
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

func (m *mockClient) sentMessages() []tgbotapi.Chattable {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]tgbotapi.Chattable, len(m.sent))
	copy(out, m.sent)
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
