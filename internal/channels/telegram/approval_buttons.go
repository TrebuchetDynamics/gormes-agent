package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const telegramApprovalTextLimit = 4096

var ErrTelegramApprovalUnavailable = errors.New("telegram_approval_unavailable")

type ApprovalPrompt struct {
	ChatID      string
	ThreadID    string
	Command     string
	SessionKey  string
	Description string
}

type telegramApprovalState struct {
	SessionKey string
	ChatID     int64
	MessageID  int
	ThreadID   string
}

type telegramApprovalUnavailableError struct {
	reason string
	err    error
}

func (e telegramApprovalUnavailableError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %s: %v", ErrTelegramApprovalUnavailable, e.reason, e.err)
	}
	return fmt.Sprintf("%s: %s", ErrTelegramApprovalUnavailable, e.reason)
}

func (e telegramApprovalUnavailableError) Unwrap() error {
	return ErrTelegramApprovalUnavailable
}

func (b *Bot) SendExecApproval(ctx context.Context, prompt ApprovalPrompt) (string, error) {
	if b.client == nil {
		return "", telegramApprovalUnavailable("not_connected", nil)
	}
	if b.cfg.ApprovalResolver == nil {
		return "", telegramApprovalUnavailable("approval_storage_unavailable", nil)
	}
	chatID, err := parseChatID(prompt.ChatID)
	if err != nil {
		return "", telegramApprovalUnavailable("chat_id_invalid", err)
	}
	if b.cfg.AllowedChatID != 0 && chatID != b.cfg.AllowedChatID {
		return "", telegramApprovalUnavailable("unregistered_chat", nil)
	}
	sessionKey := strings.TrimSpace(prompt.SessionKey)
	if sessionKey == "" {
		return "", telegramApprovalUnavailable("session_key_required", nil)
	}

	approvalID := b.nextApprovalID()
	text := telegramApprovalText(prompt.Command, prompt.Description)
	markup := telegramApprovalKeyboard(approvalID)
	msg, err := b.sendApprovalMessage(ctx, chatID, strings.TrimSpace(prompt.ThreadID), text, markup)
	if err != nil {
		return "", telegramApprovalUnavailable("post_failed", err)
	}
	b.storeApprovalSessionKey(approvalID, telegramApprovalState{
		SessionKey: sessionKey,
		ChatID:     chatID,
		MessageID:  msg.MessageID,
		ThreadID:   strings.TrimSpace(prompt.ThreadID),
	})
	return strconv.Itoa(msg.MessageID), nil
}

func (b *Bot) handleCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery) bool {
	if query == nil {
		return false
	}
	if prefix, value, ok := parseModelPickerCallback(query.Data); ok {
		return b.handleModelPickerCallback(ctx, query, prefix, value)
	}
	choice, approvalID, ok := parseTelegramApprovalCallbackData(query.Data)
	if !ok {
		return false
	}

	if !b.telegramApprovalCallbackAuthorized(query) {
		_ = b.answerCallback(query.ID, "⛔ You are not authorized to approve commands.")
		return true
	}

	state, claimed := b.claimApproval(approvalID)
	if !claimed {
		_ = b.answerCallback(query.ID, "This approval has already been resolved.")
		return true
	}

	label := telegramApprovalDecisionLabel(choice)
	actorID, actorName := telegramCallbackActor(query)
	if actorName == "" {
		actorName = actorID
	}
	if actorName == "" {
		actorName = "User"
	}
	_ = b.answerCallback(query.ID, label)
	_ = b.editApprovalMessage(query, label+" by "+actorName)

	if b.cfg.ApprovalResolver == nil {
		return true
	}
	if err := b.cfg.ApprovalResolver.ResolveGatewayApproval(ctx, gateway.ApprovalResolution{
		SessionKey: state.SessionKey,
		Choice:     choice,
		Platform:   "telegram",
		ChatID:     strconv.FormatInt(state.ChatID, 10),
		MessageID:  strconv.Itoa(state.MessageID),
		ActorID:    actorID,
		Evidence: map[string]string{
			"telegram_approval_id": strconv.FormatUint(approvalID, 10),
			"telegram_callback":    "ea",
		},
	}); err != nil {
		b.log.Warn("telegram approval resolver failed", "approval_id", approvalID, "err", err)
	}
	return true
}

func telegramApprovalUnavailable(reason string, err error) error {
	return telegramApprovalUnavailableError{reason: reason, err: err}
}

func (b *Bot) nextApprovalID() uint64 {
	b.approvalMu.Lock()
	defer b.approvalMu.Unlock()
	b.approvalNextID++
	return b.approvalNextID
}

func (b *Bot) storeApprovalSessionKey(id uint64, state telegramApprovalState) {
	b.approvalMu.Lock()
	defer b.approvalMu.Unlock()
	if b.approvalState == nil {
		b.approvalState = map[uint64]telegramApprovalState{}
	}
	b.approvalState[id] = state
}

func (b *Bot) lookupApprovalSessionKey(id uint64) (string, bool) {
	b.approvalMu.Lock()
	defer b.approvalMu.Unlock()
	state, ok := b.approvalState[id]
	return state.SessionKey, ok
}

func (b *Bot) claimApproval(id uint64) (telegramApprovalState, bool) {
	b.approvalMu.Lock()
	defer b.approvalMu.Unlock()
	state, ok := b.approvalState[id]
	if !ok {
		return telegramApprovalState{}, false
	}
	delete(b.approvalState, id)
	return state, true
}

func (b *Bot) approvalIDFromCallbackData(data string) (uint64, bool) {
	_, id, ok := parseTelegramApprovalCallbackData(data)
	return id, ok
}

func (b *Bot) sendApprovalMessage(_ context.Context, chatID int64, threadID, text string, markup tgbotapi.InlineKeyboardMarkup) (tgbotapi.Message, error) {
	if threadID == "" {
		msgCfg := tgbotapi.NewMessage(chatID, text)
		msgCfg.ParseMode = tgbotapi.ModeHTML
		msgCfg.ReplyMarkup = markup
		return b.client.Send(msgCfg)
	}
	thread, err := strconv.Atoi(threadID)
	if err != nil {
		return tgbotapi.Message{}, fmt.Errorf("telegram: invalid thread ID %q: %w", threadID, err)
	}
	markupJSON, err := json.Marshal(markup)
	if err != nil {
		return tgbotapi.Message{}, err
	}
	params := tgbotapi.Params{
		"chat_id":           strconv.FormatInt(chatID, 10),
		"message_thread_id": strconv.Itoa(thread),
		"text":              text,
		"parse_mode":        tgbotapi.ModeHTML,
		"reply_markup":      string(markupJSON),
	}
	resp, err := b.client.UploadFiles("sendMessage", params, nil)
	if err != nil {
		return tgbotapi.Message{}, err
	}
	var msg tgbotapi.Message
	if resp != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, &msg); err != nil {
			return tgbotapi.Message{}, err
		}
	}
	return msg, nil
}

func telegramApprovalText(command, description string) string {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		cmd = "(empty command)"
	}
	desc := strings.TrimSpace(description)
	if desc == "" {
		desc = "dangerous command"
	}
	desc = truncateTelegramApprovalRunes(desc, 500)
	cmd = truncateTelegramApprovalRunes(cmd, 3800)

	for {
		text := fmt.Sprintf(
			"⚠️ <b>Command Approval Required</b>\n\n<pre>%s</pre>\n\nReason: %s",
			html.EscapeString(cmd),
			html.EscapeString(desc),
		)
		if len([]rune(text)) <= telegramApprovalTextLimit {
			return text
		}
		if len([]rune(desc)) > 120 {
			desc = truncateTelegramApprovalRunes(desc, 120)
			continue
		}
		cmd = truncateTelegramApprovalRunes(cmd, len([]rune(cmd))-128)
	}
}

func telegramApprovalKeyboard(approvalID uint64) tgbotapi.InlineKeyboardMarkup {
	id := strconv.FormatUint(approvalID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Allow Once", "ea:once:"+id),
			tgbotapi.NewInlineKeyboardButtonData("✅ Session", "ea:session:"+id),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Always", "ea:always:"+id),
			tgbotapi.NewInlineKeyboardButtonData("❌ Deny", "ea:deny:"+id),
		),
	)
}

func parseTelegramApprovalCallbackData(data string) (gateway.ApprovalChoice, uint64, bool) {
	parts := strings.Split(strings.TrimSpace(data), ":")
	if len(parts) != 3 || parts[0] != "ea" {
		return "", 0, false
	}
	choice, ok := gateway.ParseApprovalChoice(parts[1])
	if !ok {
		return "", 0, false
	}
	id, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil || id == 0 {
		return "", 0, false
	}
	return choice, id, true
}

func (b *Bot) telegramApprovalCallbackAuthorized(query *tgbotapi.CallbackQuery) bool {
	if query == nil || query.From == nil {
		return false
	}
	if b.cfg.AllowedChatID != 0 {
		if query.Message == nil || query.Message.Chat == nil || query.Message.Chat.ID != b.cfg.AllowedChatID {
			return false
		}
	}
	if len(b.cfg.AllowedUserIDs) == 0 {
		return true
	}
	for _, id := range b.cfg.AllowedUserIDs {
		if query.From.ID == id {
			return true
		}
	}
	return false
}

func (b *Bot) answerCallback(callbackID, text string) error {
	if b.client == nil || strings.TrimSpace(callbackID) == "" {
		return nil
	}
	_, err := b.client.Request(tgbotapi.NewCallback(callbackID, text))
	return err
}

func (b *Bot) editApprovalMessage(query *tgbotapi.CallbackQuery, text string) error {
	if b.client == nil || query == nil || query.Message == nil || query.Message.Chat == nil {
		return nil
	}
	edit := tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID,
		query.Message.MessageID,
		text,
		tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}},
	)
	_, err := b.client.Send(edit)
	return err
}

func telegramApprovalDecisionLabel(choice gateway.ApprovalChoice) string {
	switch choice {
	case gateway.ApprovalChoiceOnce:
		return "✅ Approved once"
	case gateway.ApprovalChoiceSession:
		return "✅ Approved for session"
	case gateway.ApprovalChoiceAlways:
		return "✅ Approved permanently"
	case gateway.ApprovalChoiceDeny:
		return "❌ Denied"
	default:
		return "Resolved"
	}
}

func telegramCallbackActor(query *tgbotapi.CallbackQuery) (string, string) {
	if query == nil || query.From == nil {
		return "", ""
	}
	actorID := strconv.FormatInt(query.From.ID, 10)
	name := strings.TrimSpace(query.From.FirstName)
	if name == "" {
		name = strings.TrimSpace(query.From.UserName)
	}
	return actorID, name
}

func truncateTelegramApprovalRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func (b *Bot) handleModelPickerCallback(ctx context.Context, query *tgbotapi.CallbackQuery, prefix, value string) bool {
	if b.cfg.ModelPickerResolver == nil {
		return false
	}
	resp, err := b.cfg.ModelPickerResolver.HandleModelPickerCallback(ctx, gateway.ModelPickerCallback{
		ChatID:    strconv.FormatInt(query.Message.Chat.ID, 10),
		Prefix:    prefix,
		Value:     value,
		MessageID: query.Message.MessageID,
	})
	if err != nil {
		_ = b.answerCallback(query.ID, "Picker error")
		return true
	}
	if resp.Finished {
		if resp.Changed {
			_ = b.answerCallback(query.ID, "Model updated")
		} else {
			_ = b.answerCallback(query.ID, "Cancelled")
		}
	}
	if resp.Text != "" {
		edit := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, resp.Text)
		edit.ParseMode = "Markdown"
		if b.client != nil {
			if _, doErr := b.client.Request(edit); doErr != nil {
				b.log.Warn("model picker edit failed", "err", doErr)
			}
		}
	}
	return true
}
