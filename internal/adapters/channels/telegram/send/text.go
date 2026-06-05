package send

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TextReplyID(replyToMsgID string, required bool) (int, error) {
	if !required && strings.TrimSpace(replyToMsgID) == "" {
		return 0, nil
	}
	replyID, err := strconv.Atoi(replyToMsgID)
	if err != nil {
		return 0, fmt.Errorf("telegram: invalid reply msgID %q: %w", replyToMsgID, err)
	}
	return replyID, nil
}

func ThreadIDForTextSend(threadID, generalTopicThreadID string) (int, bool, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" || threadID == generalTopicThreadID {
		return 0, false, nil
	}
	thread, err := strconv.Atoi(threadID)
	if err != nil {
		return 0, false, fmt.Errorf("telegram: invalid thread ID %q: %w", threadID, err)
	}
	return thread, true, nil
}

func ThreadIDForAction(threadID string) (int, bool, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return 0, false, nil
	}
	thread, err := strconv.Atoi(threadID)
	if err != nil {
		return 0, false, fmt.Errorf("telegram: invalid thread ID %q: %w", threadID, err)
	}
	return thread, true, nil
}

func MessageParams(chatID int64, replyID int, text, parseMode string) tgbotapi.Params {
	params := tgbotapi.Params{}
	params.AddNonZero64("chat_id", chatID)
	params.AddNonZero("reply_to_message_id", replyID)
	params.AddNonEmpty("text", text)
	params.AddNonEmpty("parse_mode", parseMode)
	return params
}

func MessageFromAPIResponse(resp *tgbotapi.APIResponse) (tgbotapi.Message, error) {
	var msg tgbotapi.Message
	if resp != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, &msg); err != nil {
			return tgbotapi.Message{}, err
		}
	}
	return msg, nil
}
