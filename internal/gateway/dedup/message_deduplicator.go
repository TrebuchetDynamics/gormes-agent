package dedup

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/dedup/evidence"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/dedup/tracker"
)

// MessageDeduplicatorEvidence is the stable reason emitted by Track when the
// deduplicator suppresses or degrades normal tracking.
type MessageDeduplicatorEvidence = evidence.Reason

const (
	MessageDeduplicatorEvidenceDuplicate MessageDeduplicatorEvidence = evidence.Duplicate
	MessageDeduplicatorEvidenceEvicted   MessageDeduplicatorEvidence = evidence.Evicted
	MessageDeduplicatorEvidenceDisabled  MessageDeduplicatorEvidence = evidence.Disabled
)

// MessageDeduplicatorResult reports whether a message was already tracked and
// any bounded-cache evidence a caller may surface later.
type MessageDeduplicatorResult = tracker.Result

// MessageDeduplicator tracks recently seen platform message IDs in memory.
type MessageDeduplicator = tracker.MessageDeduplicator

// NewMessageDeduplicator constructs a bounded in-memory message ID tracker. A
// maxSize of zero or less disables deduplication without allocating history.
func NewMessageDeduplicator(maxSize int) *MessageDeduplicator {
	return tracker.NewMessageDeduplicator(maxSize)
}
