package dedup

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/dedup/evidence"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/dedup/inbound"
)

const EvidenceMissingMessageID MessageDeduplicatorEvidence = evidence.MissingMessageID

type InboundEventKeyParts = inbound.EventKeyParts

// InboundMessageIDSource names the inbound event field that supplied the
// deduplication candidate. It keeps fallback provenance explicit when adapters
// do not expose a platform-native MessageID.
type InboundMessageIDSource = inbound.MessageIDSource

const (
	InboundMessageIDSourceMessageID InboundMessageIDSource = inbound.MessageIDSourceMessageID
	InboundMessageIDSourceMsgID     InboundMessageIDSource = inbound.MessageIDSourceMsgID
)

// InboundMessageIdentity is the normalized inbound message candidate used for
// deduplication before it is scoped by platform/chat/thread.
type InboundMessageIdentity = inbound.MessageIdentity

// InboundDedupScope names the platform surface that must stay isolated when
// two chats or threads reuse the same platform-native message identifier.
type InboundDedupScope = inbound.Scope

// InboundDedupKeyResult reports the bounded-deduplicator tracking key or why
// one cannot be derived for an inbound event.
type InboundDedupKeyResult = inbound.KeyResult

// ResolveInboundMessageIdentity chooses the stable platform message identifier
// used for inbound deduplication, preferring MessageID and falling back to MsgID.
func ResolveInboundMessageIdentity(ev InboundEventKeyParts) InboundMessageIdentity {
	return inbound.ResolveMessageIdentity(ev)
}

// InboundDedupKey derives the key used to track inbound platform message IDs.
func InboundDedupKey(ev InboundEventKeyParts) InboundDedupKeyResult {
	return inbound.Key(ev)
}
