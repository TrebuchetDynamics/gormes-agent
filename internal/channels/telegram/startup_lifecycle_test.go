package telegram

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestTelegramStartupClearsWebhookBeforePolling(t *testing.T) {
	client := newMockClient()
	pollingStarted := make(chan struct{})
	client.GetUpdatesFn = func(ctx context.Context, _ tgbotapi.UpdateConfig) ([]tgbotapi.Update, error) {
		closeOnce(pollingStarted)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	bot := New(Config{PollingConflictRetryDelay: time.Millisecond}, client, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := runTelegramBotForTest(ctx, bot)

	select {
	case <-pollingStarted:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("polling did not start")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	requests := client.requestMessages()
	var deleteIndex, commandsIndex = -1, -1
	for i, req := range requests {
		switch cfg := req.(type) {
		case tgbotapi.DeleteWebhookConfig:
			deleteIndex = i
			if cfg.DropPendingUpdates {
				t.Fatalf("deleteWebhook DropPendingUpdates = true, want false")
			}
		case tgbotapi.SetMyCommandsConfig:
			commandsIndex = i
		}
	}
	if deleteIndex == -1 {
		t.Fatalf("Run() did not clear Telegram webhook before polling; requests=%#v", requests)
	}
	if commandsIndex == -1 {
		t.Fatalf("Run() did not register commands; requests=%#v", requests)
	}
	if deleteIndex > commandsIndex {
		t.Fatalf("deleteWebhook request index %d after setMyCommands index %d", deleteIndex, commandsIndex)
	}
}

func TestTelegramStartupTokenLockConflictIsFatalAndReleasesOwnedLock(t *testing.T) {
	t.Run("conflict", func(t *testing.T) {
		client := newMockClient()
		client.token = "123456:telegram-secret-token"
		locker := &fakeTelegramTokenLocker{err: gateway.ErrTokenLockHeld}
		bot := New(Config{tokenLocker: locker}, client, nil)

		err := bot.Run(context.Background(), make(chan<- gateway.InboundEvent))
		startup := assertTelegramStartupError(t, err, telegramStartupCodeBotTokenLock)
		if startup.Retryable {
			t.Fatalf("Retryable = true, want false")
		}
		if strings.Contains(startup.Error(), client.token) {
			t.Fatalf("startup error leaked token: %q", startup.Error())
		}
	})

	t.Run("release", func(t *testing.T) {
		client := newMockClient()
		client.token = "123456:telegram-secret-token"
		lock := &fakeTelegramTokenLock{}
		locker := &fakeTelegramTokenLocker{lock: lock}
		pollingStarted := make(chan struct{})
		client.GetUpdatesFn = func(ctx context.Context, _ tgbotapi.UpdateConfig) ([]tgbotapi.Update, error) {
			closeOnce(pollingStarted)
			<-ctx.Done()
			return nil, ctx.Err()
		}
		bot := New(Config{tokenLocker: locker}, client, nil)
		ctx, cancel := context.WithCancel(context.Background())
		done := runTelegramBotForTest(ctx, bot)
		select {
		case <-pollingStarted:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("polling did not start")
		}
		cancel()
		if err := <-done; err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !lock.released {
			t.Fatal("owned token lock was not released when Run stopped")
		}
	})
}

func TestPollingConflictRetriesBeforeFatal(t *testing.T) {
	client := newMockClient()
	var calls int
	client.GetUpdatesFn = func(ctx context.Context, _ tgbotapi.UpdateConfig) ([]tgbotapi.Update, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("Conflict: terminated by other getUpdates request")
		}
		if calls == 2 {
			return []tgbotapi.Update{{
				UpdateID: 9,
				Message: &tgbotapi.Message{
					MessageID: 10,
					Text:      "after retry",
					Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
					From:      &tgbotapi.User{ID: 7, FirstName: "tester"},
				},
			}}, nil
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	bot := New(Config{PollingConflictRetryDelay: time.Millisecond}, client, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inbox := make(chan gateway.InboundEvent, 1)
	done := make(chan error, 1)
	go func() { done <- bot.Run(ctx, inbox) }()

	select {
	case ev := <-inbox:
		if ev.Text != "after retry" {
			t.Fatalf("event text = %q, want after retry", ev.Text)
		}
	case err := <-done:
		t.Fatalf("Run() returned before retry success: %v", err)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("polling conflict did not retry into a successful update")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run() error after cancel = %v", err)
	}
	if calls < 2 {
		t.Fatalf("GetUpdates calls = %d, want retry", calls)
	}
}

func TestPollingConflictFatalAfterRetries(t *testing.T) {
	client := newMockClient()
	var calls int
	client.GetUpdatesFn = func(context.Context, tgbotapi.UpdateConfig) ([]tgbotapi.Update, error) {
		calls++
		return nil, errors.New("another bot instance is running")
	}
	bot := New(Config{PollingConflictRetryDelay: time.Millisecond}, client, nil)

	err := bot.Run(context.Background(), make(chan<- gateway.InboundEvent))
	startup := assertTelegramStartupError(t, err, telegramStartupCodePollingConflict)
	if startup.Retryable {
		t.Fatalf("Retryable = true, want false")
	}
	if calls != telegramPollingConflictMaxRetries+1 {
		t.Fatalf("GetUpdates calls = %d, want %d", calls, telegramPollingConflictMaxRetries+1)
	}
}

func TestTelegramStartupNetworkFailureIsRetryableAndRedacted(t *testing.T) {
	client := newMockClient()
	client.token = "123456:telegram-secret-token"
	client.GetUpdatesFn = func(context.Context, tgbotapi.UpdateConfig) ([]tgbotapi.Update, error) {
		return nil, &net.DNSError{
			Err:         "no such host for 123456:telegram-secret-token",
			Name:        "api.telegram.org",
			IsTemporary: true,
		}
	}
	bot := New(Config{}, client, nil)

	err := bot.Run(context.Background(), make(chan<- gateway.InboundEvent))
	startup := assertTelegramStartupError(t, err, telegramStartupCodeConnectError)
	if !startup.Retryable {
		t.Fatalf("Retryable = false, want true")
	}
	if strings.Contains(startup.Error(), client.token) {
		t.Fatalf("startup error leaked token: %q", startup.Error())
	}
}

func TestWebhookSecretRequiredOnlyInWebhookMode(t *testing.T) {
	t.Run("webhook fails closed", func(t *testing.T) {
		t.Setenv("TELEGRAM_WEBHOOK_URL", "https://example.test/telegram")
		t.Setenv("TELEGRAM_WEBHOOK_SECRET", " ")
		client := newMockClient()
		bot := New(Config{}, client, nil)

		err := bot.Run(context.Background(), make(chan<- gateway.InboundEvent))
		startup := assertTelegramStartupError(t, err, telegramStartupCodeWebhookSecretMissing)
		if startup.Retryable {
			t.Fatalf("Retryable = true, want false")
		}
		if got := len(client.requestMessages()); got != 0 {
			t.Fatalf("Telegram API requests = %d, want none when webhook secret is missing", got)
		}
	})

	t.Run("polling does not require secret", func(t *testing.T) {
		t.Setenv("TELEGRAM_WEBHOOK_URL", "")
		t.Setenv("TELEGRAM_WEBHOOK_SECRET", "")
		client := newMockClient()
		pollingStarted := make(chan struct{})
		client.GetUpdatesFn = func(ctx context.Context, _ tgbotapi.UpdateConfig) ([]tgbotapi.Update, error) {
			closeOnce(pollingStarted)
			<-ctx.Done()
			return nil, ctx.Err()
		}
		bot := New(Config{}, client, nil)
		ctx, cancel := context.WithCancel(context.Background())
		done := runTelegramBotForTest(ctx, bot)
		select {
		case <-pollingStarted:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("polling did not start without webhook secret")
		}
		cancel()
		if err := <-done; err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	})
}

func runTelegramBotForTest(ctx context.Context, bot *Bot) <-chan error {
	done := make(chan error, 1)
	go func() {
		done <- bot.Run(ctx, make(chan<- gateway.InboundEvent))
	}()
	return done
}

func assertTelegramStartupError(t *testing.T, err error, want TelegramStartupCode) *TelegramStartupError {
	t.Helper()
	if err == nil {
		t.Fatalf("Run() error = nil, want %s", want)
	}
	var startup *TelegramStartupError
	if !errors.As(err, &startup) {
		t.Fatalf("Run() error type = %T %v, want *TelegramStartupError", err, err)
	}
	if startup.Code != want {
		t.Fatalf("startup code = %q, want %q; err=%v", startup.Code, want, err)
	}
	return startup
}

type fakeTelegramTokenLocker struct {
	lock *fakeTelegramTokenLock
	err  error
}

func (f *fakeTelegramTokenLocker) AcquireTelegramToken(context.Context, string) (telegramTokenLock, gateway.TokenLockEvidence, error) {
	if f.err != nil {
		return nil, gateway.TokenLockEvidence{Status: gateway.TokenLockStatusHeld, Message: "credential lock is held"}, f.err
	}
	if f.lock == nil {
		f.lock = &fakeTelegramTokenLock{}
	}
	return f.lock, gateway.TokenLockEvidence{Status: gateway.TokenLockStatusAcquired}, nil
}

type fakeTelegramTokenLock struct {
	released bool
}

func (f *fakeTelegramTokenLock) Release(context.Context) (gateway.TokenLockEvidence, error) {
	f.released = true
	return gateway.TokenLockEvidence{Status: gateway.TokenLockStatusReleased}, nil
}

var closeOnceMu sync.Mutex
var closedChannels = map[chan struct{}]bool{}

func closeOnce(ch chan struct{}) {
	closeOnceMu.Lock()
	defer closeOnceMu.Unlock()
	if closedChannels[ch] {
		return
	}
	closedChannels[ch] = true
	close(ch)
}
