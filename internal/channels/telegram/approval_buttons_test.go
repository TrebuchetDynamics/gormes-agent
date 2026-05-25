package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

type approvalRecorder struct {
	mu    sync.Mutex
	calls []gateway.ApprovalResolution
	err   error
}

func (r *approvalRecorder) ResolveGatewayApproval(_ context.Context, res gateway.ApprovalResolution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, res)
	return r.err
}

func (r *approvalRecorder) snapshot() []gateway.ApprovalResolution {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]gateway.ApprovalResolution, len(r.calls))
	copy(out, r.calls)
	return out
}

func TestTelegramApprovalButtons_SendInlineKeyboardPrompt(t *testing.T) {
	client := newMockClient()
	resolver := &approvalRecorder{}
	b := New(Config{
		AllowedChatID:    42,
		ApprovalResolver: resolver,
	}, client, nil)

	msgID, err := b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChatID:      "42",
		Command:     "rm -rf /important",
		SessionKey:  "agent:main:telegram:group:42:99",
		Description: "dangerous deletion",
	})
	if err != nil {
		t.Fatalf("SendExecApproval: %v", err)
	}
	if msgID == "" {
		t.Fatal("message id empty")
	}

	sent := client.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(sent))
	}
	msg, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent[0] = %T, want MessageConfig", sent[0])
	}
	if msg.ChatID != 42 {
		t.Fatalf("ChatID = %d, want 42", msg.ChatID)
	}
	if msg.ParseMode != tgbotapi.ModeHTML {
		t.Fatalf("ParseMode = %q, want HTML", msg.ParseMode)
	}
	if msg.DisableNotification {
		t.Fatal("approval prompt DisableNotification = true, want approval prompts to notify")
	}
	if !strings.Contains(msg.Text, "Command Approval Required") ||
		!strings.Contains(msg.Text, "rm -rf /important") ||
		!strings.Contains(msg.Text, "dangerous deletion") {
		t.Fatalf("approval text = %q, want command and reason", msg.Text)
	}

	markup := telegramApprovalMarkup(t, msg.ReplyMarkup)
	wantRows := [][]string{
		{"✅ Allow Once", "✅ Session"},
		{"✅ Always", "❌ Deny"},
	}
	wantChoices := []string{"once", "session", "always", "deny"}
	var gotData []string
	for rowIdx, row := range markup.InlineKeyboard {
		if len(row) != len(wantRows[rowIdx]) {
			t.Fatalf("row %d buttons = %d, want %d", rowIdx, len(row), len(wantRows[rowIdx]))
		}
		for colIdx, button := range row {
			if button.Text != wantRows[rowIdx][colIdx] {
				t.Fatalf("button[%d][%d] text = %q, want %q", rowIdx, colIdx, button.Text, wantRows[rowIdx][colIdx])
			}
			if button.CallbackData == nil {
				t.Fatalf("button[%d][%d] callback data nil", rowIdx, colIdx)
			}
			gotData = append(gotData, *button.CallbackData)
		}
	}
	for i, choice := range wantChoices {
		if !strings.HasPrefix(gotData[i], "ea:"+choice+":") {
			t.Fatalf("callback %d = %q, want ea:%s:<id>", i, gotData[i], choice)
		}
	}

	approvalID, ok := b.approvalIDFromCallbackData(gotData[0])
	if !ok {
		t.Fatalf("could not parse approval id from %q", gotData[0])
	}
	if got, ok := b.lookupApprovalSessionKey(approvalID); !ok || got != "agent:main:telegram:group:42:99" {
		t.Fatalf("approval state = %q/%v, want stored session key", got, ok)
	}
}

func TestTelegramApprovalButtons_ThreadMetadataAndTruncation(t *testing.T) {
	client := newMockClient()
	b := New(Config{
		AllowedChatID:    42,
		ApprovalResolver: &approvalRecorder{},
	}, client, nil)

	_, err := b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChatID:      "42",
		ThreadID:    "777",
		Command:     strings.Repeat("x", 5000),
		SessionKey:  "telegram:42:long",
		Description: strings.Repeat("needs review ", 300),
	})
	if err != nil {
		t.Fatalf("SendExecApproval: %v", err)
	}

	uploads := client.uploadRequests()
	if len(uploads) != 1 {
		t.Fatalf("upload/raw requests = %+v, want one sendMessage raw request", uploads)
	}
	got := uploads[0]
	if got.Endpoint != "sendMessage" {
		t.Fatalf("endpoint = %q, want sendMessage", got.Endpoint)
	}
	if got.Params["chat_id"] != "42" || got.Params["message_thread_id"] != "777" {
		t.Fatalf("params = %+v, want chat_id/message_thread_id", got.Params)
	}
	if got.Params["parse_mode"] != tgbotapi.ModeHTML {
		t.Fatalf("parse_mode = %q, want HTML", got.Params["parse_mode"])
	}
	text := got.Params["text"]
	if len([]rune(text)) > telegramApprovalTextLimit {
		t.Fatalf("text length = %d, want <= %d", len([]rune(text)), telegramApprovalTextLimit)
	}
	if !strings.Contains(text, "...") {
		t.Fatalf("text = %q, want truncation marker", text)
	}
	var markup tgbotapi.InlineKeyboardMarkup
	if err := json.Unmarshal([]byte(got.Params["reply_markup"]), &markup); err != nil {
		t.Fatalf("reply_markup json: %v; raw=%q", err, got.Params["reply_markup"])
	}
	if gotButtons := len(markup.InlineKeyboard[0]) + len(markup.InlineKeyboard[1]); gotButtons != 4 {
		t.Fatalf("buttons = %d, want 4 after truncation", gotButtons)
	}
}

func TestTelegramApprovalButtons_UnavailableWhenStorageOrChannelMissing(t *testing.T) {
	resolver := &approvalRecorder{}
	b := New(Config{AllowedChatID: 42, ApprovalResolver: resolver}, nil, nil)
	_, err := b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChatID:     "42",
		Command:    "rm -rf /important",
		SessionKey: "telegram:42:missing-client",
	})
	if !errors.Is(err, ErrTelegramApprovalUnavailable) {
		t.Fatalf("nil client err = %v, want ErrTelegramApprovalUnavailable", err)
	}

	client := newMockClient()
	b = New(Config{AllowedChatID: 42}, client, nil)
	_, err = b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChatID:     "42",
		Command:    "rm -rf /important",
		SessionKey: "telegram:42:no-store",
	})
	if !errors.Is(err, ErrTelegramApprovalUnavailable) {
		t.Fatalf("missing resolver err = %v, want ErrTelegramApprovalUnavailable", err)
	}

	b = New(Config{AllowedChatID: 42, ApprovalResolver: resolver}, client, nil)
	_, err = b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChatID:     "99",
		Command:    "rm -rf /important",
		SessionKey: "telegram:99:blocked",
	})
	if !errors.Is(err, ErrTelegramApprovalUnavailable) {
		t.Fatalf("unregistered chat err = %v, want ErrTelegramApprovalUnavailable", err)
	}
	if sent := client.sentMessages(); len(sent) != 0 {
		t.Fatalf("sent = %+v, want no prompt when unavailable", sent)
	}
	if calls := resolver.snapshot(); len(calls) != 0 {
		t.Fatalf("resolver calls = %+v, want no resolution on unavailable prompt", calls)
	}
}

func TestTelegramApprovalCallback_ResolvesOnceAndEditsPrompt(t *testing.T) {
	client := newMockClient()
	resolver := &approvalRecorder{}
	b := New(Config{
		AllowedChatID:    42,
		ApprovalResolver: resolver,
	}, client, nil)
	data := sendTelegramApprovalPrompt(t, b, client, "telegram:42:once", 0)

	b.handleCallbackQuery(context.Background(), &tgbotapi.CallbackQuery{
		ID:   "callback-once-1",
		Data: data,
		From: &tgbotapi.User{ID: 111, FirstName: "Norbert"},
		Message: &tgbotapi.Message{
			MessageID: 1000,
			Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
		},
	})

	callback := lastCallbackAnswer(t, client)
	if !strings.Contains(callback.Text, "Approved once") {
		t.Fatalf("callback answer = %q, want approved once", callback.Text)
	}
	edits := telegramApprovalEdits(client)
	if len(edits) != 1 {
		t.Fatalf("edits = %+v, want one edit", edits)
	}
	if !strings.Contains(edits[0].Text, "Approved once by Norbert") {
		t.Fatalf("edit text = %q, want actor decision", edits[0].Text)
	}
	if edits[0].ReplyMarkup == nil || len(edits[0].ReplyMarkup.InlineKeyboard) != 0 {
		t.Fatalf("edit reply markup = %#v, want cleared inline keyboard", edits[0].ReplyMarkup)
	}

	resolved := resolver.snapshot()
	if len(resolved) != 1 {
		t.Fatalf("resolver calls = %+v, want one", resolved)
	}
	if resolved[0].SessionKey != "telegram:42:once" || resolved[0].Choice != gateway.ApprovalChoiceOnce {
		t.Fatalf("resolution = %+v, want once session key", resolved[0])
	}
	if resolved[0].Platform != "telegram" || resolved[0].ChatID != "42" || resolved[0].MessageID != "1000" || resolved[0].ActorID != "111" {
		t.Fatalf("resolution metadata = %+v, want redacted Telegram evidence", resolved[0])
	}
}

func TestTelegramApprovalCallback_AccountScopedPlatform(t *testing.T) {
	client := newMockClient()
	resolver := &approvalRecorder{}
	b := New(Config{
		AllowedChatID:    42,
		AccountID:        "ops",
		ApprovalResolver: resolver,
	}, client, nil)
	data := sendTelegramApprovalPrompt(t, b, client, "telegram:ops:42:once", 0)

	b.handleCallbackQuery(context.Background(), &tgbotapi.CallbackQuery{
		ID:   "callback-account-1",
		Data: data,
		From: &tgbotapi.User{ID: 111, FirstName: "Norbert"},
		Message: &tgbotapi.Message{
			MessageID: 1000,
			Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
		},
	})

	resolved := resolver.snapshot()
	if len(resolved) != 1 {
		t.Fatalf("resolver calls = %+v, want one", resolved)
	}
	if resolved[0].Platform != "telegram:ops" {
		t.Fatalf("Platform = %q, want telegram:ops", resolved[0].Platform)
	}
}

func TestTelegramApprovalCallback_DoubleClickAckedWithoutSecondResolution(t *testing.T) {
	client := newMockClient()
	resolver := &approvalRecorder{}
	b := New(Config{
		AllowedChatID:    42,
		ApprovalResolver: resolver,
	}, client, nil)
	data := sendTelegramApprovalPrompt(t, b, client, "telegram:42:session", 1)

	click := func(callbackID string) {
		b.handleCallbackQuery(context.Background(), &tgbotapi.CallbackQuery{
			ID:   callbackID,
			Data: data,
			From: &tgbotapi.User{ID: 111, FirstName: "Ada"},
			Message: &tgbotapi.Message{
				MessageID: 1000,
				Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
			},
		})
	}
	click("callback-double-1")
	click("callback-double-2")

	if got := len(resolver.snapshot()); got != 1 {
		t.Fatalf("resolver calls = %d, want 1", got)
	}
	if edits := telegramApprovalEdits(client); len(edits) != 1 {
		t.Fatalf("edits = %+v, want one prompt edit", edits)
	}
	callbacks := callbackAnswers(client)
	if len(callbacks) != 2 {
		t.Fatalf("callback answers = %+v, want two acks", callbacks)
	}
	if !strings.Contains(callbacks[1].Text, "already been resolved") {
		t.Fatalf("second callback answer = %q, want already resolved", callbacks[1].Text)
	}
}

func TestTelegramApprovalCallback_UnauthorizedUserLeavesApprovalPending(t *testing.T) {
	client := newMockClient()
	resolver := &approvalRecorder{}
	b := New(Config{
		AllowedChatID:    42,
		AllowedUserIDs:   []int64{111},
		ApprovalResolver: resolver,
	}, client, nil)
	data := sendTelegramApprovalPrompt(t, b, client, "telegram:42:unauthorized", 0)

	b.handleCallbackQuery(context.Background(), &tgbotapi.CallbackQuery{
		ID:   "callback-unauthorized",
		Data: data,
		From: &tgbotapi.User{ID: 222, FirstName: "Mallory"},
		Message: &tgbotapi.Message{
			MessageID: 1000,
			Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
		},
	})

	if got := len(resolver.snapshot()); got != 0 {
		t.Fatalf("resolver calls = %d, want 0", got)
	}
	if edits := telegramApprovalEdits(client); len(edits) != 0 {
		t.Fatalf("edits = %+v, want no edit for unauthorized callback", edits)
	}
	if answer := lastCallbackAnswer(t, client); !strings.Contains(strings.ToLower(answer.Text), "not authorized") {
		t.Fatalf("callback answer = %q, want not authorized", answer.Text)
	}
	approvalID, ok := b.approvalIDFromCallbackData(data)
	if !ok {
		t.Fatalf("approval id parse failed for %q", data)
	}
	if got, ok := b.lookupApprovalSessionKey(approvalID); !ok || got != "telegram:42:unauthorized" {
		t.Fatalf("approval state = %q/%v, want pending session key", got, ok)
	}
}

func TestTelegramApprovalCallback_DenyMapsToGatewayChoice(t *testing.T) {
	client := newMockClient()
	resolver := &approvalRecorder{}
	b := New(Config{
		AllowedChatID:    42,
		ApprovalResolver: resolver,
	}, client, nil)
	data := sendTelegramApprovalPrompt(t, b, client, "telegram:42:deny", 3)

	b.handleCallbackQuery(context.Background(), &tgbotapi.CallbackQuery{
		ID:   "callback-deny",
		Data: data,
		From: &tgbotapi.User{ID: 111, FirstName: "Ada"},
		Message: &tgbotapi.Message{
			MessageID: 1000,
			Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
		},
	})

	resolved := resolver.snapshot()
	if len(resolved) != 1 {
		t.Fatalf("resolver calls = %+v, want one", resolved)
	}
	if resolved[0].Choice != gateway.ApprovalChoiceDeny {
		t.Fatalf("choice = %q, want deny", resolved[0].Choice)
	}
	if edit := telegramApprovalEdits(client)[0]; !strings.Contains(edit.Text, "Denied by Ada") {
		t.Fatalf("edit text = %q, want Denied by Ada", edit.Text)
	}
}

func TestTelegramApprovalCallback_UnrelatedPrefixesDoNotResolveApproval(t *testing.T) {
	client := newMockClient()
	resolver := &approvalRecorder{}
	b := New(Config{
		AllowedChatID:    42,
		ApprovalResolver: resolver,
	}, client, nil)
	approvalData := sendTelegramApprovalPrompt(t, b, client, "telegram:42:pending", 0)

	for _, data := range []string{"mp:some_provider", "update_prompt:y", "sc:once:confirm"} {
		handled := b.handleCallbackQuery(context.Background(), &tgbotapi.CallbackQuery{
			ID:   "callback-" + data,
			Data: data,
			From: &tgbotapi.User{ID: 111, FirstName: "Ada"},
			Message: &tgbotapi.Message{
				MessageID: 1000,
				Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
			},
		})
		if handled {
			t.Fatalf("handleCallbackQuery(%q) = true, want false for non-approval callback family", data)
		}
	}

	if got := len(resolver.snapshot()); got != 0 {
		t.Fatalf("resolver calls = %d, want 0", got)
	}
	approvalID, ok := b.approvalIDFromCallbackData(approvalData)
	if !ok {
		t.Fatalf("approval id parse failed for %q", approvalData)
	}
	if _, ok := b.lookupApprovalSessionKey(approvalID); !ok {
		t.Fatal("approval state was removed by unrelated callback family")
	}
}

func sendTelegramApprovalPrompt(t *testing.T, b *Bot, client *mockClient, sessionKey string, buttonIndex int) string {
	t.Helper()
	_, err := b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChatID:      "42",
		Command:     "rm -rf /important",
		SessionKey:  sessionKey,
		Description: "dangerous command",
	})
	if err != nil {
		t.Fatalf("SendExecApproval: %v", err)
	}
	sent := client.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(sent))
	}
	msg, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent[0] = %T, want MessageConfig", sent[0])
	}
	markup := telegramApprovalMarkup(t, msg.ReplyMarkup)
	var buttons []tgbotapi.InlineKeyboardButton
	for _, row := range markup.InlineKeyboard {
		buttons = append(buttons, row...)
	}
	if buttonIndex < 0 || buttonIndex >= len(buttons) {
		t.Fatalf("button index %d out of range for %#v", buttonIndex, buttons)
	}
	if buttons[buttonIndex].CallbackData == nil {
		t.Fatalf("button %d callback data nil", buttonIndex)
	}
	return *buttons[buttonIndex].CallbackData
}

func telegramApprovalMarkup(t *testing.T, value any) tgbotapi.InlineKeyboardMarkup {
	t.Helper()
	markup, ok := value.(tgbotapi.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("reply markup = %T, want InlineKeyboardMarkup", value)
	}
	if len(markup.InlineKeyboard) != 2 {
		t.Fatalf("keyboard rows = %d, want 2: %#v", len(markup.InlineKeyboard), markup.InlineKeyboard)
	}
	return markup
}

func lastCallbackAnswer(t *testing.T, client *mockClient) tgbotapi.CallbackConfig {
	t.Helper()
	answers := callbackAnswers(client)
	if len(answers) == 0 {
		t.Fatal("no callback answers")
	}
	return answers[len(answers)-1]
}

func callbackAnswers(client *mockClient) []tgbotapi.CallbackConfig {
	var out []tgbotapi.CallbackConfig
	for _, req := range client.requestMessages() {
		if cfg, ok := req.(tgbotapi.CallbackConfig); ok {
			out = append(out, cfg)
		}
	}
	return out
}

func telegramApprovalEdits(client *mockClient) []tgbotapi.EditMessageTextConfig {
	var out []tgbotapi.EditMessageTextConfig
	for _, sent := range client.sentMessages() {
		if cfg, ok := sent.(tgbotapi.EditMessageTextConfig); ok {
			out = append(out, cfg)
		}
	}
	return out
}
