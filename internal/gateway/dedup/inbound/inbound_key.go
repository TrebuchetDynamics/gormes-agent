package inbound

import (
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/dedup/evidence"
)

const (
	EvidenceMissingMessageID evidence.Reason = evidence.MissingMessageID
	EvidenceMissingScope     evidence.Reason = evidence.MissingScope
)

type EventKeyParts struct {
	Platform  string
	AccountID string
	ChatID    string
	ThreadID  string
	MsgID     string
	MessageID string
}

// MessageIDSource names the inbound event field that supplied the
// deduplication candidate. It keeps fallback provenance explicit when adapters
// do not expose a platform-native MessageID.
type MessageIDSource string

const (
	MessageIDSourceMessageID MessageIDSource = "message_id"
	MessageIDSourceMsgID     MessageIDSource = "msg_id"
)

// MessageIdentity is the normalized inbound message candidate used for
// deduplication before it is scoped by platform/chat/thread.
type MessageIdentity struct {
	ID     string
	Source MessageIDSource
}

// Scope names the platform surface that must stay isolated when two chats or
// threads reuse the same platform-native message identifier.
type Scope struct {
	Platform  string
	AccountID string
	ChatID    string
	ThreadID  string
}

// KeyResult reports the bounded-deduplicator tracking key or why one cannot be
// derived for an inbound event.
type KeyResult struct {
	Key      string
	Evidence evidence.Reason
	Identity MessageIdentity
	Scope    Scope
}

// ResolveMessageIdentity chooses the stable platform message identifier used
// for inbound deduplication, preferring MessageID and falling back to MsgID.
func ResolveMessageIdentity(ev EventKeyParts) MessageIdentity {
	if messageID := strings.TrimSpace(ev.MessageID); messageID != "" {
		return MessageIdentity{ID: messageID, Source: MessageIDSourceMessageID}
	}
	if msgID := strings.TrimSpace(ev.MsgID); msgID != "" {
		return MessageIdentity{ID: msgID, Source: MessageIDSourceMsgID}
	}
	return MessageIdentity{}
}

// Key derives the key used to track inbound platform message IDs.
func Key(ev EventKeyParts) KeyResult {
	identity := ResolveMessageIdentity(ev)
	scope := Scope{Platform: ev.Platform, AccountID: ev.AccountID, ChatID: ev.ChatID, ThreadID: ev.ThreadID}
	if identity.ID == "" {
		return KeyResult{Evidence: EvidenceMissingMessageID, Scope: scope}
	}
	if strings.TrimSpace(scope.Platform) == "" || strings.TrimSpace(scope.ChatID) == "" {
		return KeyResult{Evidence: EvidenceMissingScope, Identity: identity, Scope: scope}
	}
	return KeyResult{
		Key:      scope.TrackingKey(identity),
		Identity: identity,
		Scope:    scope,
	}
}

// TrackingKey returns the length-prefixed key used by the bounded deduplicator.
// AccountID is part of the scope so multiple configured accounts on the same
// platform cannot suppress each other's coincident chat/thread message IDs.
// Scope components are intentionally byte-exact: admission and platform
// adapters own their own normalization before this point.
func (s Scope) TrackingKey(identity MessageIdentity) string {
	if identity.ID == "" {
		return ""
	}
	return keyParts(s.Platform, s.AccountID, s.ChatID, s.ThreadID, identity.ID)
}

func keyParts(parts ...string) string {
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
