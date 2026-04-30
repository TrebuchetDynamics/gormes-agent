package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TestBot_SendChatAction_RequestsTypingAction proves that a successful call
// routes through client.Request with the correct ChatActionConfig payload.
func TestBot_SendChatAction_RequestsTypingAction(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 12345}, mc, nil)

	if err := b.SendChatAction(context.Background(), "12345", "typing"); err != nil {
		t.Fatalf("SendChatAction error: %v", err)
	}

	reqs := mc.requestMessages()
	if len(reqs) != 1 {
		t.Fatalf("Request call count = %d, want 1", len(reqs))
	}
	cfg, ok := reqs[0].(tgbotapi.ChatActionConfig)
	if !ok {
		t.Fatalf("Request payload type = %T, want tgbotapi.ChatActionConfig", reqs[0])
	}
	if cfg.ChatID != 12345 {
		t.Errorf("ChatActionConfig.ChatID = %d, want 12345", cfg.ChatID)
	}
	if cfg.Action != "typing" {
		t.Errorf("ChatActionConfig.Action = %q, want \"typing\"", cfg.Action)
	}
}

// TestBot_SendChatAction_NonFatalOnAPIError proves that a Request error is
// returned to the caller without panic, goroutine, or retry.
func TestBot_SendChatAction_NonFatalOnAPIError(t *testing.T) {
	mc := newMockClient()
	apiErr := errors.New("telegram: 429 too many requests")
	mc.RequestFn = func(_ tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
		return nil, apiErr
	}
	b := New(Config{AllowedChatID: 12345}, mc, nil)

	err := b.SendChatAction(context.Background(), "12345", "typing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apiErr) && !strings.Contains(err.Error(), "too many requests") {
		t.Errorf("error = %v, want wrapped apiErr", err)
	}
}

// TestBot_SendChatAction_RejectsBlankChatID proves blank and whitespace-only
// chat IDs are rejected before any Request call.
func TestBot_SendChatAction_RejectsBlankChatID(t *testing.T) {
	for _, chatID := range []string{"", "  "} {
		mc := newMockClient()
		b := New(Config{}, mc, nil)

		err := b.SendChatAction(context.Background(), chatID, "typing")
		if err == nil {
			t.Fatalf("chatID=%q: expected error, got nil", chatID)
		}
		if !strings.Contains(err.Error(), "chat_id") {
			t.Errorf("chatID=%q: error = %q, want mention of chat_id", chatID, err.Error())
		}
		reqs := mc.requestMessages()
		if len(reqs) != 0 {
			t.Errorf("chatID=%q: Request was called %d time(s), want 0", chatID, len(reqs))
		}
	}
}

// TestBot_SendChatAction_RejectsBlankAction proves a blank action is rejected
// before any Request call.
func TestBot_SendChatAction_RejectsBlankAction(t *testing.T) {
	mc := newMockClient()
	b := New(Config{}, mc, nil)

	err := b.SendChatAction(context.Background(), "12345", "")
	if err == nil {
		t.Fatal("expected error for blank action, got nil")
	}
	if !strings.Contains(err.Error(), "action") {
		t.Errorf("error = %q, want mention of action", err.Error())
	}
	reqs := mc.requestMessages()
	if len(reqs) != 0 {
		t.Errorf("Request was called %d time(s), want 0", len(reqs))
	}
}

// TestBot_SendChatAction_RejectsNonNumericChatID proves non-numeric chat IDs
// are rejected before any Request call, matching the parseChatID convention
// used by other Bot methods.
func TestBot_SendChatAction_RejectsNonNumericChatID(t *testing.T) {
	mc := newMockClient()
	b := New(Config{}, mc, nil)

	err := b.SendChatAction(context.Background(), "not-a-number", "typing")
	if err == nil {
		t.Fatal("expected error for non-numeric chat_id, got nil")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "chat_id") && !strings.Contains(lower, "chat id") && !strings.Contains(lower, "not-a-number") {
		t.Errorf("error = %q, want mention of invalid chat_id", err.Error())
	}
	reqs := mc.requestMessages()
	if len(reqs) != 0 {
		t.Errorf("Request was called %d time(s), want 0", len(reqs))
	}
}
