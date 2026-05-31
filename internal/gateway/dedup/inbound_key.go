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

// InboundMessageIDSource names the inbound event field that supplied the
// deduplication candidate. It keeps fallback provenance explicit when adapters
// do not expose a platform-native MessageID.
type InboundMessageIDSource string

const (
	InboundMessageIDSourceMessageID InboundMessageIDSource = "message_id"
	InboundMessageIDSourceMsgID     InboundMessageIDSource = "msg_id"
)

// InboundMessageIdentity is the normalized inbound message candidate used for
// deduplication before it is scoped by platform/chat/thread.
type InboundMessageIdentity struct {
	ID     string
	Source InboundMessageIDSource
}

// InboundDedupKeyResult reports the bounded-deduplicator tracking key or why
// one cannot be derived for an inbound event.
type InboundDedupKeyResult struct {
	Key      string
	Evidence MessageDeduplicatorEvidence
	Identity InboundMessageIdentity
}

// ResolveInboundMessageIdentity chooses the stable platform message identifier
// used for inbound deduplication, preferring MessageID and falling back to MsgID.
func ResolveInboundMessageIdentity(ev InboundEventKeyParts) InboundMessageIdentity {
	if messageID := strings.TrimSpace(ev.MessageID); messageID != "" {
		return InboundMessageIdentity{ID: messageID, Source: InboundMessageIDSourceMessageID}
	}
	if msgID := strings.TrimSpace(ev.MsgID); msgID != "" {
		return InboundMessageIdentity{ID: msgID, Source: InboundMessageIDSourceMsgID}
	}
	return InboundMessageIdentity{}
}

// InboundDedupKey derives the key used to track inbound platform message IDs.
func InboundDedupKey(ev InboundEventKeyParts) InboundDedupKeyResult {
	identity := ResolveInboundMessageIdentity(ev)
	if identity.ID == "" {
		return InboundDedupKeyResult{Evidence: EvidenceMissingMessageID}
	}
	return InboundDedupKeyResult{
		Key:      inboundDedupKeyParts(ev.Platform, ev.ChatID, ev.ThreadID, identity.ID),
		Identity: identity,
	}
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
