package telegram

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
)

func TestTelegramGatewayE2EAccountChannelRoutesAndFormatsFinal(t *testing.T) {
	mc := newMockClient()
	bot := New(Config{AllowedChatID: 42, AccountID: "ops"}, mc, nil)
	if got := bot.Name(); got != "telegram:ops" {
		t.Fatalf("bot name = %q, want account-scoped platform", got)
	}

	provider := hermes.NewMockClient()
	provider.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "Use a_b(c)!"},
		{Kind: hermes.EventDone, FinishReason: "stop"},
	}, "sess-telegram-e2e")
	k := kernel.New(kernel.Config{
		Model:     "mock-telegram-model",
		Endpoint:  "http://mock-provider",
		Admission: kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, provider, store.NewNoop(), telemetry.New(), slog.Default())

	mgr := gateway.NewManager(gateway.ManagerConfig{
		AllowedChats: map[string]string{"telegram:ops": "42"},
		CoalesceMs:   5,
	}, k, slog.Default())
	if err := mgr.Register(bot); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() { _ = k.Run(ctx) }()
	go func() { _ = mgr.Run(ctx) }()

	mc.updatesCh <- tgbotapi.Update{
		UpdateID: 10,
		Message: &tgbotapi.Message{
			MessageID: 77,
			Text:      "hello from the ops account",
			Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
			From:      &tgbotapi.User{ID: 9001, FirstName: "operator"},
		},
	}

	waitForTelegramE2E(t, time.Second, func() bool { return len(provider.Requests()) == 1 })
	req := provider.Requests()[0]
	last := req.Messages[len(req.Messages)-1]
	if last.Role != "user" || last.Content != "hello from the ops account" {
		t.Fatalf("provider final request message = %+v, want Telegram user text", last)
	}

	waitForTelegramE2E(t, time.Second, func() bool {
		_, _, ok := lastTelegramTextSend(mc.sentMessages())
		return ok
	})
	text, parseMode, _ := lastTelegramTextSend(mc.sentMessages())
	if parseMode != tgbotapi.ModeMarkdownV2 {
		t.Fatalf("final send ParseMode = %q, want MarkdownV2; sent=%#v", parseMode, mc.sentMessages())
	}
	if !strings.Contains(text, `a\_b\(c\)\!`) {
		t.Fatalf("final Telegram text = %q, want MarkdownV2 escaped assistant content", text)
	}
	if strings.Contains(text, "a_b(c)!") {
		t.Fatalf("final Telegram text kept raw MarkdownV2 specials: %q", text)
	}
}

func TestTelegramInboundEventUsesAccountScopedPlatform(t *testing.T) {
	b := New(Config{AllowedChatID: 42, AccountID: "ops"}, newMockClient(), nil)
	ev, ok := b.toInboundEvent(context.Background(), tgbotapi.Update{Message: &tgbotapi.Message{
		MessageID: 1,
		Text:      "hello",
		Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
		From:      &tgbotapi.User{ID: 7},
	}})
	if !ok {
		t.Fatal("toInboundEvent rejected account message")
	}
	if ev.Platform != "telegram:ops" {
		t.Fatalf("Platform = %q, want account-scoped channel name", ev.Platform)
	}
	if ev.AccountID != "ops" {
		t.Fatalf("AccountID = %q, want ops", ev.AccountID)
	}
}

func lastTelegramTextSend(sent []tgbotapi.Chattable) (text, parseMode string, ok bool) {
	for i := len(sent) - 1; i >= 0; i-- {
		switch msg := sent[i].(type) {
		case tgbotapi.MessageConfig:
			if strings.TrimSpace(msg.Text) != "" {
				return msg.Text, msg.ParseMode, true
			}
		case tgbotapi.EditMessageTextConfig:
			if strings.TrimSpace(msg.Text) != "" {
				return msg.Text, msg.ParseMode, true
			}
		}
	}
	return "", "", false
}

func waitForTelegramE2E(t *testing.T, timeout time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
