package telegram

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/session"
)

// Config drives the Bot adapter. AllowedChatID and FirstRunDiscovery follow
// the spec's M1/M2 rules: either a non-zero allowlist OR discovery enabled,
// never neither.
type Config struct {
	AllowedChatID     int64
	CoalesceMs        int
	FirstRunDiscovery bool
	// SessionMap + SessionKey (Phase 2.C) — optional. When SessionMap is
	// non-nil, the outbound goroutine persists k.sessionID on every frame
	// where it changed. Nil disables persistence (Phase 2.B.1 behavior).
	SessionMap session.Map
	SessionKey string
}

// Bot is the Telegram adapter. Kernel-side state (draft, phase, history)
// lives in *kernel.Kernel; Bot holds only per-adapter streaming state.
type Bot struct {
	cfg     Config
	client  telegramClient
	kernel  *kernel.Kernel
	log     *slog.Logger
	lastSID string // most recently persisted session_id (prevents duplicate Puts)
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

// Run starts the inbound + outbound goroutines and blocks until ctx
// cancellation. Task 5 extends handleUpdate with command parsing + kernel
// submission; Task 7 wires /new's kernel.ResetSession call.
func (b *Bot) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	wg.Add(1)
	go b.runOutbound(ctx, &wg)

	ucfg := tgbotapi.NewUpdate(0)
	ucfg.Timeout = 30
	updates := b.client.GetUpdatesChan(ucfg)

	defer wg.Wait()

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
					"Then restart gormes telegram.")
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

	// Command routing + kernel submission (Phase 2.B.1 T5).
	text := strings.TrimSpace(u.Message.Text)
	switch {
	case text == "/start":
		_, _ = b.client.Send(tgbotapi.NewMessage(chatID,
			"Gormes is online. Send a message to start a turn. Commands: /stop /new"))
	case text == "/stop":
		_ = b.kernel.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventCancel})
	case text == "/new":
		if err := b.kernel.ResetSession(); err != nil {
			if errors.Is(err, kernel.ErrResetDuringTurn) {
				_, _ = b.client.Send(tgbotapi.NewMessage(chatID,
					"Cannot reset during active turn — send /stop first."))
			} else {
				_, _ = b.client.Send(tgbotapi.NewMessage(chatID,
					"Session reset failed: "+err.Error()))
			}
			return
		}
		_, _ = b.client.Send(tgbotapi.NewMessage(chatID,
			"Session reset. Next message starts fresh."))
	case strings.HasPrefix(text, "/"):
		_, _ = b.client.Send(tgbotapi.NewMessage(chatID, "unknown command"))
	default:
		err := b.kernel.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: text})
		if err != nil {
			_, _ = b.client.Send(tgbotapi.NewMessage(chatID,
				"Busy — try again in a second."))
		}
	}
}

// runOutbound consumes k.Render() and pushes frames into the coalescer.
// One coalescer per turn: on PhaseIdle/Failed/Cancelling we flushImmediate
// and null the coalescer so the next turn starts with a fresh placeholder.
func (b *Bot) runOutbound(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	frames := b.kernel.Render()
	var c *coalescer
	var cCancel context.CancelFunc

	for {
		select {
		case <-ctx.Done():
			if cCancel != nil {
				cCancel()
			}
			return
		case f, ok := <-frames:
			if !ok {
				if cCancel != nil {
					cCancel()
				}
				return
			}
			b.persistIfChanged(ctx, f)
			b.handleFrame(ctx, f, &c, &cCancel, wg)
		}
	}
}

// persistIfChanged writes the frame's SessionID to SessionMap when it has
// changed from the last persisted value. Failures log a WARN and do NOT
// fail the turn — persistence is best-effort on the render path.
func (b *Bot) persistIfChanged(ctx context.Context, f kernel.RenderFrame) {
	if b.cfg.SessionMap == nil || b.cfg.SessionKey == "" {
		return
	}
	if f.SessionID == b.lastSID {
		return
	}
	if err := b.cfg.SessionMap.Put(ctx, b.cfg.SessionKey, f.SessionID); err != nil {
		b.log.Warn("failed to persist session_id",
			"key", b.cfg.SessionKey,
			"session_id", f.SessionID,
			"err", err)
		return
	}
	b.lastSID = f.SessionID
}

// handleFrame dispatches one RenderFrame to the coalescer. Lazy-inits the
// coalescer on the first streaming/connecting frame; tears it down on
// PhaseIdle/Failed/Cancelling with flushImmediate.
func (b *Bot) handleFrame(
	ctx context.Context,
	f kernel.RenderFrame,
	c **coalescer,
	cCancel *context.CancelFunc,
	wg *sync.WaitGroup,
) {
	switch f.Phase {
	case kernel.PhaseIdle:
		if *c != nil {
			(*c).flushImmediate(formatFinal(f))
			(*cCancel)()
			*c = nil
		}
	case kernel.PhaseFailed, kernel.PhaseCancelling:
		if *c != nil {
			(*c).flushImmediate(formatError(f))
			(*cCancel)()
			*c = nil
		} else {
			// No active turn; still surface the error as a new message.
			// Bot sends directly — no coalescer needed for a single send.
			_, _ = b.client.Send(tgbotapi.NewMessage(b.cfg.AllowedChatID, formatError(f)))
		}
	case kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseReconnecting, kernel.PhaseFinalizing:
		if *c == nil {
			// Lazy-init coalescer for a new turn.
			var cCtx context.Context
			cCtx, *cCancel = context.WithCancel(ctx)
			window := time.Duration(b.cfg.CoalesceMs) * time.Millisecond
			newC := newCoalescer(b.client, window, b.cfg.AllowedChatID)
			*c = newC
			wg.Add(1)
			go func(cc *coalescer, cx context.Context) {
				defer wg.Done()
				cc.run(cx)
			}(newC, cCtx)
			newC.flushImmediate("⏳") // establish the message
		}
		(*c).setPending(formatStream(f))
	}
}
