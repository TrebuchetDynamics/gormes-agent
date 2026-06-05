package messaging

import "strings"

// ReplyReferenceKey builds the in-memory de-duplication key for Discord reply references.
func ReplyReferenceKey(chatID, replyToMsgID string) string {
	return strings.TrimSpace(chatID) + ":" + strings.TrimSpace(replyToMsgID)
}
