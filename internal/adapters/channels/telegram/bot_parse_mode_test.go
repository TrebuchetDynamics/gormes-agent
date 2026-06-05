package telegram

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// TestBot_Send_SetsMarkdownV2ParseMode locks in the Hermes parity contract:
// every Bot.Send call must set ParseMode = MarkdownV2 so that
// internal/gateway/render.go's MarkdownV2-escaped output renders as bold/
// italic/code/spoilers on Telegram clients instead of literal backslashes.
// See progress.json row "Telegram MarkdownV2 parse-mode rendering closeout"
// and Hermes upstream gateway/platforms/telegram.py:1112.
func TestBot_Send_SetsMarkdownV2ParseMode(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	if _, err := b.Send(context.Background(), "42", "*bold* text"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	msg, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent type = %T, want MessageConfig", sent[0])
	}
	if msg.ParseMode != tgbotapi.ModeMarkdownV2 {
		t.Fatalf("ParseMode = %q, want %q", msg.ParseMode, tgbotapi.ModeMarkdownV2)
	}
}

// TestBot_SendReply_SetsMarkdownV2ParseMode mirrors the Send contract for
// the reply path the gateway uses for quoted replies.
func TestBot_SendReply_SetsMarkdownV2ParseMode(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	if _, err := b.SendReply(context.Background(), "42", "99", "*bold* reply"); err != nil {
		t.Fatalf("SendReply: %v", err)
	}

	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	msg, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent type = %T, want MessageConfig", sent[0])
	}
	if msg.ParseMode != tgbotapi.ModeMarkdownV2 {
		t.Fatalf("ParseMode = %q, want %q", msg.ParseMode, tgbotapi.ModeMarkdownV2)
	}
	if msg.ReplyToMessageID != 99 {
		t.Fatalf("ReplyToMessageID = %d, want 99", msg.ReplyToMessageID)
	}
}

// TestBot_EditMessage_SetsMarkdownV2ParseMode covers EditMessageText, which
// Hermes also sends with parse_mode=MARKDOWN_V2 (telegram.py:1224).
func TestBot_EditMessage_SetsMarkdownV2ParseMode(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	if err := b.EditMessage(context.Background(), "42", "1234", "*bold* edit"); err != nil {
		t.Fatalf("EditMessage: %v", err)
	}

	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	msg, ok := sent[0].(tgbotapi.EditMessageTextConfig)
	if !ok {
		t.Fatalf("sent type = %T, want EditMessageTextConfig", sent[0])
	}
	if msg.ParseMode != tgbotapi.ModeMarkdownV2 {
		t.Fatalf("ParseMode = %q, want %q", msg.ParseMode, tgbotapi.ModeMarkdownV2)
	}
}

func TestBot_EditMessageFinal_NonFinalUsesPlainTextWithoutMarkdownV2(t *testing.T) {
	var _ gateway.FinalizingMessageEditor = (*Bot)(nil)

	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	body := "partial **bold"
	if err := b.EditMessageFinal(context.Background(), "42", "1234", body, false); err != nil {
		t.Fatalf("EditMessageFinal(non-final): %v", err)
	}

	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	msg, ok := sent[0].(tgbotapi.EditMessageTextConfig)
	if !ok {
		t.Fatalf("sent type = %T, want EditMessageTextConfig", sent[0])
	}
	if msg.ParseMode != "" {
		t.Fatalf("ParseMode = %q, want empty for non-final streaming edit", msg.ParseMode)
	}
	if msg.Text != body {
		t.Fatalf("body = %q, want exact partial body %q", msg.Text, body)
	}
}

func TestBot_EditMessageFinal_FinalUsesMarkdownV2WithPlainFallback(t *testing.T) {
	mc := newMockClient()
	calls := 0
	mc.SendFn = func(c tgbotapi.Chattable) (tgbotapi.Message, error) {
		calls++
		if calls == 1 {
			return tgbotapi.Message{}, errors.New("Bad Request: can't parse entities: malformed markdown")
		}
		return tgbotapi.Message{MessageID: 1234}, nil
	}
	b := New(Config{AllowedChatID: 42}, mc, nil)

	body := `final *bold* and \(parens\)`
	wantFallback := "final bold and (parens)"
	if err := b.EditMessageFinal(context.Background(), "42", "1234", body, true); err != nil {
		t.Fatalf("EditMessageFinal(final): %v", err)
	}

	sent := mc.sentMessages()
	if len(sent) != 2 {
		t.Fatalf("sent count = %d, want 2 (MarkdownV2 attempt + fallback)", len(sent))
	}
	first, ok := sent[0].(tgbotapi.EditMessageTextConfig)
	if !ok {
		t.Fatalf("first send type = %T, want EditMessageTextConfig", sent[0])
	}
	if first.ParseMode != tgbotapi.ModeMarkdownV2 {
		t.Fatalf("first ParseMode = %q, want %q", first.ParseMode, tgbotapi.ModeMarkdownV2)
	}
	if first.Text != body {
		t.Fatalf("first body = %q, want %q", first.Text, body)
	}
	second, ok := sent[1].(tgbotapi.EditMessageTextConfig)
	if !ok {
		t.Fatalf("second send type = %T, want EditMessageTextConfig", sent[1])
	}
	if second.ParseMode != "" {
		t.Fatalf("fallback ParseMode = %q, want empty", second.ParseMode)
	}
	if second.Text != wantFallback {
		t.Fatalf("fallback body = %q, want %q", second.Text, wantFallback)
	}
}

// TestBot_Send_FallsBackToPlainOnMarkdownV2ParseError mirrors the Hermes
// fallback at gateway/platforms/telegram.py:1117-1129. When Telegram rejects
// MarkdownV2 with a parse-entity error, the bot must retry once with
// ParseMode unset and a clean plaintext body, mirroring Hermes _strip_mdv2.
func TestBot_Send_FallsBackToPlainOnMarkdownV2ParseError(t *testing.T) {
	mc := newMockClient()
	calls := 0
	mc.SendFn = func(c tgbotapi.Chattable) (tgbotapi.Message, error) {
		calls++
		if calls == 1 {
			return tgbotapi.Message{}, errors.New("Bad Request: can't parse entities: bla")
		}
		return tgbotapi.Message{MessageID: 4242}, nil
	}
	b := New(Config{AllowedChatID: 42}, mc, nil)

	body := "literal *star* and \\(parens\\) — escaped by render.go"
	wantFallback := "literal star and (parens) — escaped by render.go"
	id, err := b.Send(context.Background(), "42", body)
	if err != nil {
		t.Fatalf("Send returned error after fallback: %v", err)
	}
	if id != "4242" {
		t.Fatalf("Send returned id %q, want 4242", id)
	}

	sent := mc.sentMessages()
	if len(sent) != 2 {
		t.Fatalf("sent count = %d, want 2 (original + fallback)", len(sent))
	}
	first, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("first send type = %T, want MessageConfig", sent[0])
	}
	if first.ParseMode != tgbotapi.ModeMarkdownV2 {
		t.Fatalf("first ParseMode = %q, want %q", first.ParseMode, tgbotapi.ModeMarkdownV2)
	}
	if first.Text != body {
		t.Fatalf("first body = %q, want %q", first.Text, body)
	}
	second, ok := sent[1].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("second send type = %T, want MessageConfig", sent[1])
	}
	if second.ParseMode != "" {
		t.Fatalf("fallback ParseMode = %q, want empty", second.ParseMode)
	}
	if second.Text != wantFallback {
		t.Fatalf("fallback body = %q, want stripped plaintext %q", second.Text, wantFallback)
	}
}

// TestBot_Send_PreservesEscapingFromRenderLayer guarantees bot.go does not
// double-escape: render.go already produces MarkdownV2-escaped strings
// (verified in internal/gateway/render_test.go). bot.go must hand the body
// through untouched.
func TestBot_Send_PreservesEscapingFromRenderLayer(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	// Body already escaped by render.go: a literal asterisk plus escaped
	// parens. If bot.go double-escapes, the backslashes themselves get
	// escaped and the body changes.
	body := `*literal\* asterisk \(and parens\)`
	if _, err := b.Send(context.Background(), "42", body); err != nil {
		t.Fatalf("Send: %v", err)
	}

	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	msg, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent type = %T, want MessageConfig", sent[0])
	}
	if msg.Text != body {
		t.Fatalf("body = %q, want %q (bot.go must not double-escape)", msg.Text, body)
	}
}
