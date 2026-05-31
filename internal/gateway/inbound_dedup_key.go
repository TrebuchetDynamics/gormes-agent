package gateway

import gatewaydedup "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/dedup"

const MessageDeduplicatorEvidenceMissingMessageID MessageDeduplicatorEvidence = gatewaydedup.EvidenceMissingMessageID

// InboundMessageIDSource names the inbound event field that supplied the
// deduplication candidate.
type InboundMessageIDSource = gatewaydedup.InboundMessageIDSource

const (
	InboundMessageIDSourceMessageID InboundMessageIDSource = gatewaydedup.InboundMessageIDSourceMessageID
	InboundMessageIDSourceMsgID     InboundMessageIDSource = gatewaydedup.InboundMessageIDSourceMsgID
)

// InboundMessageIdentity is the normalized inbound message candidate used for
// deduplication before it is scoped by platform/chat/thread.
type InboundMessageIdentity = gatewaydedup.InboundMessageIdentity

// InboundDedupKeyResult reports the bounded-deduplicator tracking key or why
// one cannot be derived for an inbound event.
type InboundDedupKeyResult = gatewaydedup.InboundDedupKeyResult

// ResolveInboundMessageIdentity chooses the stable platform message identifier
// used for inbound deduplication, preferring MessageID and falling back to MsgID.
func ResolveInboundMessageIdentity(ev InboundEvent) InboundMessageIdentity {
	return gatewaydedup.ResolveInboundMessageIdentity(gatewaydedup.InboundEventKeyParts{
		MsgID:     ev.MsgID,
		MessageID: ev.MessageID,
	})
}

// InboundDedupKey derives the key used to track inbound platform message IDs.
func InboundDedupKey(ev InboundEvent) InboundDedupKeyResult {
	return gatewaydedup.InboundDedupKey(gatewaydedup.InboundEventKeyParts{
		Platform:  ev.Platform,
		AccountID: ev.AccountID,
		ChatID:    ev.ChatID,
		ThreadID:  ev.ThreadID,
		MsgID:     ev.MsgID,
		MessageID: ev.MessageID,
	})
}
