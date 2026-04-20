package telegram

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/XelHaku/golang-hermes-agent/gormes/internal/kernel"
)

// Config drives the Bot adapter. AllowedChatID and FirstRunDiscovery follow
// the spec's M1/M2 rules: either a non-zero allowlist OR discovery enabled,
// never neither.
type Config struct {
	AllowedChatID     int64
	CoalesceMs        int
	FirstRunDiscovery bool
}

// Bot is the Telegram adapter. Kernel-side state (draft, phase, history)
// lives in *kernel.Kernel; Bot holds only per-adapter streaming state.
type Bot struct {
	cfg    Config
	client telegramClient
	kernel *kernel.Kernel
	log    *slog.Logger
}

// New constructs a Bot wired to the given telegramClient + kernel.
func New(cfg Config, client telegramClient, k *kernel.Kernel, log *slog.Logger) *Bot {
	if log == nil {
		log = slog.Default()
	}
	if cfg.CoalesceMs <= 0 {
		cfg.CoalesceMs = 1000
	}
	return &Bot{cfg: cfg, client: client, kernel: k, log: log}
}

// Run starts the inbound long-poll loop and blocks until ctx cancellation.
// Task 4 adds the outbound goroutine; Task 5 adds command parsing and
// kernel submission. Task 1 only handles the auth gate.
func (b *Bot) Run(ctx context.Context) error {
	ucfg := tgbotapi.NewUpdate(0)
	ucfg.Timeout = 30
	updates := b.client.GetUpdatesChan(ucfg)

	for {
		select {
		case <-ctx.Done():
			b.client.StopReceivingUpdates()
			return nil
		case u, ok := <-updates:
			if !ok {
				return nil
			}
			b.handleUpdate(ctx, u)
		}
	}
}

// handleUpdate processes one Telegram Update. Task 1: auth gate only.
func (b *Bot) handleUpdate(ctx context.Context, u tgbotapi.Update) {
	if u.Message == nil {
		return
	}
	chatID := u.Message.Chat.ID

	if b.cfg.AllowedChatID == 0 {
		if b.cfg.FirstRunDiscovery {
			b.log.Info("first-run discovery: unknown chat", "chat_id", chatID)
			reply := tgbotapi.NewMessage(chatID,
				"Gormes is not authorised for this chat.\n"+
					"To allow: set [telegram].allowed_chat_id in config.toml.\n"+
					"Then restart gormes-telegram.")
			_, _ = b.client.Send(reply)
		} else {
			b.log.Warn("unauthorised chat blocked", "chat_id", chatID)
		}
		return
	}
	if chatID != b.cfg.AllowedChatID {
		b.log.Warn("unauthorised chat blocked", "chat_id", chatID)
		return
	}

	// Task 5 replaces this no-op with command parsing + kernel.Submit.
	b.log.Info("inbound message", "chat_id", chatID, "text", u.Message.Text)
}
