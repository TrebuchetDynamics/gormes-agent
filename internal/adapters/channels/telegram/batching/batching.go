package batching

import (
	"strings"

	telegramcontent "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/content"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func InboundEventHasPhoto(ev gateway.InboundEvent) bool {
	for _, attachment := range ev.Attachments {
		if strings.EqualFold(strings.TrimSpace(attachment.Kind), "photo") {
			return true
		}
	}
	return false
}

func PhotoBatchKey(ev gateway.InboundEvent, mediaGroupID string) string {
	chatID := strings.TrimSpace(ev.ChatID)
	if chatID == "" {
		chatID = "unknown-chat"
	}
	if mediaGroupID = strings.TrimSpace(mediaGroupID); mediaGroupID != "" {
		return "album:" + chatID + ":" + mediaGroupID
	}
	userID := strings.TrimSpace(ev.UserID)
	if userID == "" {
		userID = "unknown-user"
	}
	return "burst:" + chatID + ":" + userID
}

func MergePhotoBatch(first, next gateway.InboundEvent) gateway.InboundEvent {
	first.Text = telegramcontent.MergeCaption(first.Text, next.Text)
	first.Attachments = append(first.Attachments, next.Attachments...)
	return first
}

func InboundEventIsBatchableText(ev gateway.InboundEvent) bool {
	return ev.Kind == gateway.EventSubmit && strings.TrimSpace(ev.Text) != "" && len(ev.Attachments) == 0
}

func TextBatchKey(ev gateway.InboundEvent) string {
	parts := []string{
		strings.TrimSpace(ev.Platform),
		strings.TrimSpace(ev.ChatID),
		strings.TrimSpace(ev.ChatType),
		strings.TrimSpace(ev.ThreadID),
		strings.TrimSpace(ev.UserID),
	}
	for i, part := range parts {
		if part == "" {
			parts[i] = "-"
		}
	}
	return strings.Join(parts, ":")
}

func MergeTextBatch(first, next gateway.InboundEvent) gateway.InboundEvent {
	text := strings.TrimSpace(next.Text)
	if text == "" {
		return first
	}
	if strings.TrimSpace(first.Text) == "" {
		first.Text = text
	} else {
		first.Text += "\n" + text
	}
	return first
}
