package gateway

import gatewaydedup "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/dedup"

// MessageDeduplicatorEvidence is the stable reason emitted by Track when the
// deduplicator suppresses or degrades normal tracking.
type MessageDeduplicatorEvidence = gatewaydedup.MessageDeduplicatorEvidence

const (
	MessageDeduplicatorEvidenceDuplicate MessageDeduplicatorEvidence = gatewaydedup.MessageDeduplicatorEvidenceDuplicate
	MessageDeduplicatorEvidenceEvicted   MessageDeduplicatorEvidence = gatewaydedup.MessageDeduplicatorEvidenceEvicted
	MessageDeduplicatorEvidenceDisabled  MessageDeduplicatorEvidence = gatewaydedup.MessageDeduplicatorEvidenceDisabled
)

// MessageDeduplicatorResult reports whether a message was already tracked and
// any bounded-cache evidence a caller may surface later.
type MessageDeduplicatorResult = gatewaydedup.MessageDeduplicatorResult

// MessageDeduplicator tracks recently seen platform message IDs in memory.
type MessageDeduplicator = gatewaydedup.MessageDeduplicator

// NewMessageDeduplicator constructs a bounded in-memory message ID tracker. A
// maxSize of zero or less disables deduplication without allocating history.
func NewMessageDeduplicator(maxSize int) *MessageDeduplicator {
	return gatewaydedup.NewMessageDeduplicator(maxSize)
}
