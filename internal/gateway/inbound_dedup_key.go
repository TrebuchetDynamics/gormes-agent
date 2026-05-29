package gateway

import gatewaydedup "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/dedup"

const MessageDeduplicatorEvidenceMissingMessageID MessageDeduplicatorEvidence = gatewaydedup.EvidenceMissingMessageID

// InboundDedupKeyResult reports the bounded-deduplicator tracking key or why
// one cannot be derived for an inbound event.
type InboundDedupKeyResult = gatewaydedup.InboundDedupKeyResult

// InboundDedupKey derives the key used to track inbound platform message IDs.
func InboundDedupKey(ev InboundEvent) InboundDedupKeyResult {
	return gatewaydedup.InboundDedupKey(gatewaydedup.InboundEventKeyParts{
		Platform:  ev.Platform,
		ChatID:    ev.ChatID,
		ThreadID:  ev.ThreadID,
		MsgID:     ev.MsgID,
		MessageID: ev.MessageID,
	})
}
