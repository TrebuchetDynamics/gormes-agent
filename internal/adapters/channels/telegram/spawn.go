package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const telegramCreateForumTopicEndpoint = "createForumTopic"

type telegramForumTopic struct {
	MessageThreadID int    `json:"message_thread_id"`
	Name            string `json:"name"`
}

func (b *Bot) CreateForumTopic(ctx context.Context, chatID, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	id, err := parseChatID(chatID)
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("telegram: CreateForumTopic requires non-empty name")
	}
	params := tgbotapi.Params{}
	params.AddNonZero64("chat_id", id)
	params.AddNonEmpty("name", name)
	resp, err := b.client.UploadFiles(telegramCreateForumTopicEndpoint, params, nil)
	if err != nil {
		return "", err
	}
	var topic telegramForumTopic
	if resp != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, &topic); err != nil {
			return "", fmt.Errorf("telegram: decode createForumTopic response: %w", err)
		}
	}
	if topic.MessageThreadID == 0 {
		return "", errors.New("telegram: createForumTopic response missing message_thread_id")
	}
	return strconv.Itoa(topic.MessageThreadID), nil
}
