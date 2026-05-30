package telegram

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	telegramstartup "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/startup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramStartupCode = telegramstartup.Code

const (
	telegramStartupCodeBotTokenLock         TelegramStartupCode = telegramstartup.CodeBotTokenLock
	telegramStartupCodePollingConflict      TelegramStartupCode = telegramstartup.CodePollingConflict
	telegramStartupCodeConnectError         TelegramStartupCode = telegramstartup.CodeConnectError
	telegramStartupCodeWebhookSecretMissing TelegramStartupCode = telegramstartup.CodeWebhookSecretMissing
)

const telegramPollingConflictMaxRetries = 3
const telegramPollingNetworkMaxRetries = 10

// TelegramStartupError is the typed operator evidence returned when Telegram
// startup must fail before message ingress begins.
type TelegramStartupError = telegramstartup.Error

type telegramTokenLock interface {
	Release(context.Context) (gateway.TokenLockEvidence, error)
}

type telegramTokenLocker interface {
	AcquireTelegramToken(context.Context, string) (telegramTokenLock, gateway.TokenLockEvidence, error)
}

type telegramPollingDrainer interface {
	DrainPollingConnections(context.Context) error
}

type telegramHeartbeatProber interface {
	ProbeTelegramHeartbeat(context.Context) error
}

type gatewayTelegramTokenLocker struct {
	dir string
}

func (l gatewayTelegramTokenLocker) AcquireTelegramToken(ctx context.Context, token string) (telegramTokenLock, gateway.TokenLockEvidence, error) {
	store := gateway.NewTokenLockStore(l.dir)
	return store.Acquire(ctx, gateway.TokenLockRequest{
		Platform:   "telegram",
		Credential: token,
	})
}

func (b *Bot) prepareStartup(ctx context.Context) error {
	if err := telegramValidateWebhookSecret(); err != nil {
		return err
	}
	if err := b.acquireStartupTokenLock(ctx); err != nil {
		return err
	}
	if _, err := b.client.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false}); err != nil {
		b.releaseStartupTokenLock(context.Background())
		retryable := telegramLooksLikeNetworkError(err)
		return newTelegramStartupError(
			telegramStartupCodeConnectError,
			"telegram startup API request failed: "+sanitizeTelegramStartupError(err),
			retryable,
			err,
		)
	}
	return nil
}

func telegramValidateWebhookSecret() error {
	if strings.TrimSpace(os.Getenv("TELEGRAM_WEBHOOK_URL")) == "" {
		return nil
	}
	if strings.TrimSpace(os.Getenv("TELEGRAM_WEBHOOK_SECRET")) != "" {
		return nil
	}
	return newTelegramStartupError(
		telegramStartupCodeWebhookSecretMissing,
		"TELEGRAM_WEBHOOK_SECRET is required when TELEGRAM_WEBHOOK_URL is set; generate one with `openssl rand -hex 32` and register it with Telegram as the webhook secret_token",
		false,
		nil,
	)
}

func (b *Bot) acquireStartupTokenLock(ctx context.Context) error {
	if b == nil || b.client == nil {
		return nil
	}
	token := strings.TrimSpace(b.client.Token())
	if token == "" {
		return nil
	}
	locker := b.cfg.tokenLocker
	if locker == nil {
		locker = gatewayTelegramTokenLocker{dir: b.cfg.TokenLockDir}
	}
	lock, evidence, err := locker.AcquireTelegramToken(ctx, token)
	if err != nil {
		message := "Telegram bot token is already in use on this host"
		if strings.TrimSpace(evidence.Message) != "" {
			message += ": " + sanitizeTelegramStartupText(evidence.Message)
		}
		return newTelegramStartupError(telegramStartupCodeBotTokenLock, message, false, err)
	}
	b.startupMu.Lock()
	b.startupLock = lock
	b.startupMu.Unlock()
	return nil
}

func (b *Bot) releaseStartupTokenLock(ctx context.Context) {
	if b == nil {
		return
	}
	b.startupMu.Lock()
	lock := b.startupLock
	b.startupLock = nil
	b.startupMu.Unlock()
	if lock == nil {
		return
	}
	if _, err := lock.Release(ctx); err != nil {
		b.log.Warn("telegram token lock release failed", "err", sanitizeTelegramStartupError(err))
	}
}

func (b *Bot) handlePollingError(ctx context.Context, err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false, nil
	}
	if telegramLooksLikePollingConflict(err) {
		b.pollingConflictCount++
		if b.pollingConflictCount <= telegramPollingConflictMaxRetries {
			delay := b.telegramPollingConflictRetryDelay()
			if delay > 0 {
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return false, nil
				case <-timer.C:
				}
			}
			b.log.Warn("telegram polling conflict; retrying", "attempt", b.pollingConflictCount, "max", telegramPollingConflictMaxRetries)
			b.drainPollingConnections(ctx)
			return true, nil
		}
		return false, newTelegramStartupError(
			telegramStartupCodePollingConflict,
			fmt.Sprintf("another process is already polling this Telegram bot token; stopped after %d retries", telegramPollingConflictMaxRetries),
			false,
			err,
		)
	}
	if telegramLooksLikeNetworkError(err) {
		b.pollingConflictCount++
		if b.pollingConflictCount > telegramPollingNetworkMaxRetries {
			return false, newTelegramStartupError(
				telegramStartupCodeConnectError,
				fmt.Sprintf("telegram polling could not reconnect after %d network retries: %s", telegramPollingNetworkMaxRetries, sanitizeTelegramStartupError(err)),
				true,
				err,
			)
		}
		b.log.Warn("telegram polling network error; reconnecting", "attempt", b.pollingConflictCount, "max", telegramPollingNetworkMaxRetries, "err", sanitizeTelegramStartupError(err))
		b.drainPollingConnections(ctx)
		delay := b.telegramPollingConflictRetryDelay()
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return false, nil
			case <-timer.C:
			}
		}
		b.probeTelegramHeartbeat(ctx)
		return true, nil
	}
	return false, err
}

func (b *Bot) drainPollingConnections(ctx context.Context) {
	if b == nil || b.client == nil {
		return
	}
	drainer, ok := b.client.(telegramPollingDrainer)
	if !ok {
		return
	}
	if err := drainer.DrainPollingConnections(ctx); err != nil {
		b.log.Debug("telegram polling request drain failed", "err", sanitizeTelegramStartupError(err))
	}
}

func (b *Bot) probeTelegramHeartbeat(ctx context.Context) {
	if b == nil || b.client == nil {
		return
	}
	prober, ok := b.client.(telegramHeartbeatProber)
	if !ok {
		return
	}
	if err := prober.ProbeTelegramHeartbeat(ctx); err != nil {
		b.log.Debug("telegram heartbeat probe failed", "err", sanitizeTelegramStartupError(err))
	}
}

func (b *Bot) telegramPollingConflictRetryDelay() time.Duration {
	if b != nil && b.cfg.PollingConflictRetryDelay > 0 {
		return b.cfg.PollingConflictRetryDelay
	}
	return 10 * time.Second
}

func newTelegramStartupError(code TelegramStartupCode, message string, retryable bool, err error) *TelegramStartupError {
	return telegramstartup.NewError(code, message, retryable, err)
}

func telegramLooksLikePollingConflict(err error) bool {
	return telegramstartup.LooksLikePollingConflict(err)
}

func telegramLooksLikeNetworkError(err error) bool {
	return telegramstartup.LooksLikeNetworkError(err)
}

func sanitizeTelegramStartupError(err error) string {
	return telegramstartup.SanitizeError(err)
}

func sanitizeTelegramStartupText(text string) string {
	return telegramstartup.SanitizeText(text)
}
