package telegram

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()
	old, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%s): %v", key, err)
	}
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func telegramUploadsSnapshot(client *mockClient) []mockUpload {
	client.mu.Lock()
	defer client.mu.Unlock()
	out := make([]mockUpload, len(client.uploads))
	copy(out, client.uploads)
	return out
}

func TestTelegramReactionsDisabledByDefault(t *testing.T) {
	unsetEnvForTest(t, "TELEGRAM_REACTIONS")
	client := newMockClient()
	bot := New(Config{}, client, nil)

	if err := bot.OnProcessingStart(context.Background(), "123", "456"); err != nil {
		t.Fatalf("OnProcessingStart: %v", err)
	}
	if uploads := telegramUploadsSnapshot(client); len(uploads) != 0 {
		t.Fatalf("uploads = %+v, want no reaction upload when disabled by default", uploads)
	}
}

func TestTelegramReactionLifecycleSetsProcessingAndTerminalEmojis(t *testing.T) {
	t.Setenv("TELEGRAM_REACTIONS", "true")
	client := newMockClient()
	bot := New(Config{}, client, nil)

	if err := bot.OnProcessingStart(context.Background(), "123", "456"); err != nil {
		t.Fatalf("OnProcessingStart: %v", err)
	}
	if err := bot.OnProcessingComplete(context.Background(), "123", "456", gateway.ProcessingOutcomeSuccess); err != nil {
		t.Fatalf("OnProcessingComplete success: %v", err)
	}
	if err := bot.OnProcessingComplete(context.Background(), "123", "456", gateway.ProcessingOutcomeFailure); err != nil {
		t.Fatalf("OnProcessingComplete failure: %v", err)
	}

	uploads := telegramUploadsSnapshot(client)
	if len(uploads) != 3 {
		t.Fatalf("uploads = %+v, want start/success/failure reaction calls", uploads)
	}
	wantEmoji := []string{"👀", "👍", "👎"}
	for i, upload := range uploads {
		if upload.Endpoint != "setMessageReaction" {
			t.Fatalf("upload[%d].Endpoint = %q, want setMessageReaction", i, upload.Endpoint)
		}
		if upload.Params["chat_id"] != "123" || upload.Params["message_id"] != "456" {
			t.Fatalf("upload[%d].Params = %+v, want chat_id/message_id", i, upload.Params)
		}
		if !strings.Contains(upload.Params["reaction"], wantEmoji[i]) {
			t.Fatalf("upload[%d].reaction = %q, want emoji %q", i, upload.Params["reaction"], wantEmoji[i])
		}
	}
}

func TestTelegramReactionLifecycleSkipsCancelledMissingIDsAndAPIErrors(t *testing.T) {
	t.Setenv("TELEGRAM_REACTIONS", "true")
	client := newMockClient()
	client.UploadFilesFn = func(string, tgbotapi.Params, []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		return nil, errors.New("telegram no permissions")
	}
	bot := New(Config{}, client, nil)

	if err := bot.OnProcessingStart(context.Background(), "", "456"); err != nil {
		t.Fatalf("OnProcessingStart missing chat: %v", err)
	}
	if err := bot.OnProcessingComplete(context.Background(), "123", "", gateway.ProcessingOutcomeSuccess); err != nil {
		t.Fatalf("OnProcessingComplete missing message: %v", err)
	}
	if err := bot.OnProcessingComplete(context.Background(), "123", "456", gateway.ProcessingOutcomeCancelled); err != nil {
		t.Fatalf("OnProcessingComplete cancelled: %v", err)
	}
	if err := bot.OnProcessingStart(context.Background(), "123", "456"); err != nil {
		t.Fatalf("OnProcessingStart API error should be swallowed, got %v", err)
	}

	uploads := telegramUploadsSnapshot(client)
	if len(uploads) != 1 {
		t.Fatalf("uploads = %+v, want only API-error start attempt", uploads)
	}
}
