package telegram

import (
	"context"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestTelegramTextBatch(t *testing.T) {
	t.Run("single message dispatches after delay", func(t *testing.T) {
		b := New(Config{TextBatchDelay: 20 * time.Millisecond}, newMockClient(), nil)
		inbox := make(chan gateway.InboundEvent, 2)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := b.handleUpdate(ctx, inbox, telegramTextUpdate(42, 1, "hello world")); err != nil {
			t.Fatalf("handleUpdate: %v", err)
		}
		assertNoTelegramBatchEvent(t, inbox)

		ev := receiveTelegramBatchEvent(t, inbox)
		if ev.Text != "hello world" {
			t.Fatalf("Text = %q, want merged text", ev.Text)
		}
		if b.pendingTextBatchCount() != 0 {
			t.Fatalf("pendingTextBatchCount = %d, want cleanup", b.pendingTextBatchCount())
		}
	})

	t.Run("rapid same chat messages merge", func(t *testing.T) {
		b := New(Config{TextBatchDelay: 30 * time.Millisecond}, newMockClient(), nil)
		inbox := make(chan gateway.InboundEvent, 2)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := b.handleUpdate(ctx, inbox, telegramTextUpdate(42, 1, "chunk 1")); err != nil {
			t.Fatalf("handleUpdate first: %v", err)
		}
		if err := b.handleUpdate(ctx, inbox, telegramTextUpdate(42, 2, "chunk 2")); err != nil {
			t.Fatalf("handleUpdate second: %v", err)
		}

		ev := receiveTelegramBatchEvent(t, inbox)
		if ev.Text != "chunk 1\nchunk 2" {
			t.Fatalf("Text = %q, want newline-merged chunks", ev.Text)
		}
		assertNoTelegramBatchEvent(t, inbox)
	})

	t.Run("different chats stay isolated", func(t *testing.T) {
		b := New(Config{TextBatchDelay: 20 * time.Millisecond}, newMockClient(), nil)
		inbox := make(chan gateway.InboundEvent, 2)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := b.handleUpdate(ctx, inbox, telegramTextUpdate(42, 1, "from A")); err != nil {
			t.Fatalf("handleUpdate chat A: %v", err)
		}
		if err := b.handleUpdate(ctx, inbox, telegramTextUpdate(99, 2, "from B")); err != nil {
			t.Fatalf("handleUpdate chat B: %v", err)
		}

		got := map[string]string{}
		for range 2 {
			ev := receiveTelegramBatchEvent(t, inbox)
			got[ev.ChatID] = ev.Text
		}
		if got["42"] != "from A" || got["99"] != "from B" {
			t.Fatalf("events = %#v, want isolated chat batches", got)
		}
	})

	t.Run("disconnect drains without leaking pending state", func(t *testing.T) {
		b := New(Config{TextBatchDelay: time.Hour}, newMockClient(), nil)
		inbox := make(chan gateway.InboundEvent, 2)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := b.handleUpdate(ctx, inbox, telegramTextUpdate(42, 1, "pending")); err != nil {
			t.Fatalf("handleUpdate: %v", err)
		}
		if b.pendingTextBatchCount() != 1 {
			t.Fatalf("pendingTextBatchCount = %d, want one pending batch", b.pendingTextBatchCount())
		}
		if err := b.Disconnect(ctx); err != nil {
			t.Fatalf("Disconnect: %v", err)
		}

		ev := receiveTelegramBatchEvent(t, inbox)
		if ev.Text != "pending" {
			t.Fatalf("Text = %q, want pending batch drained", ev.Text)
		}
		if b.pendingTextBatchCount() != 0 {
			t.Fatalf("pendingTextBatchCount = %d, want cleanup after disconnect", b.pendingTextBatchCount())
		}
	})

	t.Run("group mention gate runs before batching", func(t *testing.T) {
		b := New(Config{
			RequireMention: true,
			BotUsername:    "gormes_bot",
			TextBatchDelay: 20 * time.Millisecond,
		}, newMockClient(), nil)
		inbox := make(chan gateway.InboundEvent, 1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		update := telegramTextUpdate(-100, 1, "hello group")
		update.Message.Chat.Type = "supergroup"
		if err := b.handleUpdate(ctx, inbox, update); err != nil {
			t.Fatalf("handleUpdate: %v", err)
		}

		if b.pendingTextBatchCount() != 0 {
			t.Fatalf("pendingTextBatchCount = %d, want unauthorized group message not buffered", b.pendingTextBatchCount())
		}
		assertNoTelegramBatchEvent(t, inbox)
	})
}

func telegramTextUpdate(chatID int64, messageID int, text string) tgbotapi.Update {
	return tgbotapi.Update{
		UpdateID: messageID,
		Message: &tgbotapi.Message{
			MessageID: messageID,
			From:      &tgbotapi.User{ID: 111},
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "private"},
			Text:      text,
		},
	}
}

func receiveTelegramBatchEvent(t *testing.T, inbox <-chan gateway.InboundEvent) gateway.InboundEvent {
	t.Helper()
	select {
	case ev := <-inbox:
		return ev
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for Telegram batch event")
		return gateway.InboundEvent{}
	}
}

func assertNoTelegramBatchEvent(t *testing.T, inbox <-chan gateway.InboundEvent) {
	t.Helper()
	select {
	case ev := <-inbox:
		t.Fatalf("unexpected Telegram batch event: %+v", ev)
	case <-time.After(5 * time.Millisecond):
	}
}
