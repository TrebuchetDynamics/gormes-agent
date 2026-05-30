package telegram

import (
	"context"
	"strconv"
	"strings"

	telegramsend "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/send"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type telegramTextSendRequest struct {
	ChatID       string
	ReplyToMsgID string
	RequireReply bool
	ThreadID     string
	Text         string
	ParseMode    string
}

func (b *Bot) sendTelegramText(ctx context.Context, req telegramTextSendRequest) (string, error) {
	chatID, err := parseChatID(req.ChatID)
	if err != nil {
		return "", err
	}
	replyID, err := telegramTextReplyID(req.ReplyToMsgID, req.RequireReply)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(req.ThreadID) == "" {
		msgCfg := tgbotapi.NewMessage(chatID, req.Text)
		msgCfg.ParseMode = req.ParseMode
		msgCfg.ReplyToMessageID = replyID
		msg, err := b.sendMessageConfig(msgCfg)
		if err != nil {
			return "", err
		}
		return strconv.Itoa(msg.MessageID), nil
	}

	thread, includeThread, err := telegramThreadIDForTextSend(req.ThreadID)
	if err != nil {
		return "", err
	}
	params := telegramSendMessageParams(chatID, replyID, req.Text, req.ParseMode)
	if includeThread {
		params.AddNonZero("message_thread_id", thread)
	}
	return b.sendThreadParamsWithRetry(ctx, params, includeThread)
}

func telegramTextReplyID(replyToMsgID string, required bool) (int, error) {
	return telegramsend.TextReplyID(replyToMsgID, required)
}

func (b *Bot) sendMessageConfig(msgCfg tgbotapi.MessageConfig) (tgbotapi.Message, error) {
	if msgCfg.ParseMode == "" {
		return b.client.Send(msgCfg)
	}
	return b.sendWithParseFallback(msgCfg)
}
