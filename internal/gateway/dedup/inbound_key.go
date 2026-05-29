package dedup

import (
	"strconv"
	"strings"
)

const EvidenceMissingMessageID MessageDeduplicatorEvidence = "dedup_unavailable_missing_message_id"

type InboundEventKeyParts struct {
	Platform  string
	ChatID    string
	ThreadID  string
	MsgID     string
	MessageID string
}

// InboundDedupKeyResult reports the bounded-deduplicator tracking key or why
// one cannot be derived for an inbound event.
type InboundDedupKeyResult struct {
	Key      string
	Evidence MessageDeduplicatorEvidence
}

// InboundDedupKey derives the key used to track inbound platform message IDs.
func InboundDedupKey(ev InboundEventKeyParts) InboundDedupKeyResult {
	messageID := strings.TrimSpace(ev.MessageID)
	if messageID == "" {
		messageID = strings.TrimSpace(ev.MsgID)
	}
	if messageID == "" {
		return InboundDedupKeyResult{Evidence: EvidenceMissingMessageID}
	}
	return InboundDedupKeyResult{Key: inboundDedupKeyParts(ev.Platform, ev.ChatID, ev.ThreadID, messageID)}
}

func inboundDedupKeyParts(parts ...string) string {
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(strconv.Itoa(len(part)))
		b.WriteByte(':')
		b.WriteString(part)
	}
	return b.String()
}
