package gateway

// MessageDeduplicatorEvidence is the stable reason emitted by Track when the
// deduplicator suppresses or degrades normal tracking.
type MessageDeduplicatorEvidence string

const (
	MessageDeduplicatorEvidenceDuplicate MessageDeduplicatorEvidence = "duplicate_message"
	MessageDeduplicatorEvidenceEvicted   MessageDeduplicatorEvidence = "deduplicator_evicted"
	MessageDeduplicatorEvidenceDisabled  MessageDeduplicatorEvidence = "deduplicator_disabled"
)

// MessageDeduplicatorResult reports whether a message was already tracked and
// any bounded-cache evidence a caller may surface later.
type MessageDeduplicatorResult struct {
	Duplicate bool
	Evidence  MessageDeduplicatorEvidence
	EvictedID string
}

// MessageDeduplicator tracks recently seen platform message IDs in memory.
type MessageDeduplicator struct {
	maxSize int
	seen    map[string]struct{}
	order   []string
}

// NewMessageDeduplicator constructs a bounded in-memory message ID tracker. A
// maxSize of zero or less disables deduplication without allocating history.
func NewMessageDeduplicator(maxSize int) *MessageDeduplicator {
	d := &MessageDeduplicator{maxSize: maxSize}
	if maxSize > 0 {
		d.seen = make(map[string]struct{}, maxSize)
		d.order = make([]string, 0, maxSize)
	}
	return d
}

// Track records messageID when it is new and reports duplicate or eviction
// evidence when the bounded cache changes observable behavior.
func (d *MessageDeduplicator) Track(messageID string) MessageDeduplicatorResult {
	if d == nil || d.maxSize <= 0 {
		return MessageDeduplicatorResult{Evidence: MessageDeduplicatorEvidenceDisabled}
	}
	if messageID == "" {
		return MessageDeduplicatorResult{}
	}
	if d.seen == nil {
		d.seen = make(map[string]struct{}, d.maxSize)
	}
	if _, ok := d.seen[messageID]; ok {
		return MessageDeduplicatorResult{
			Duplicate: true,
			Evidence:  MessageDeduplicatorEvidenceDuplicate,
		}
	}

	result := MessageDeduplicatorResult{}
	if len(d.order) >= d.maxSize {
		evictedID := d.order[0]
		delete(d.seen, evictedID)
		copy(d.order, d.order[1:])
		d.order = d.order[:len(d.order)-1]
		result.Evidence = MessageDeduplicatorEvidenceEvicted
		result.EvictedID = evictedID
	}
	d.seen[messageID] = struct{}{}
	d.order = append(d.order, messageID)
	return result
}
