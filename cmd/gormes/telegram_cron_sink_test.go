package main

import (
	"context"
	"testing"
)

// fakeBotSender lets the test drive the DeliverySink without a live
// Telegram client. Matches the telegramBotSender interface that
// newTelegramDeliverySink accepts.
type fakeBotSender struct {
	sentChatID int64
	sentText   string
}

func (f *fakeBotSender) SendToChat(ctx context.Context, chatID int64, text string) error {
	f.sentChatID = chatID
	f.sentText = text
	return nil
}

func TestTelegramDeliverySink_ForwardsToConfiguredChatID(t *testing.T) {
	bot := &fakeBotSender{}
	sink := newTelegramDeliverySink(bot, 4242)
	if err := sink.Deliver(context.Background(), "hello"); err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if bot.sentChatID != 4242 {
		t.Errorf("chat_id = %d, want 4242", bot.sentChatID)
	}
	if bot.sentText != "hello" {
		t.Errorf("text = %q, want 'hello'", bot.sentText)
	}
}
