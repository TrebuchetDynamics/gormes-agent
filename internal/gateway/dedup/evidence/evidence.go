package evidence

// Reason is the stable reason emitted when inbound deduplication suppresses or
// degrades normal tracking.
type Reason string

const (
	Duplicate        Reason = "duplicate_message"
	Evicted          Reason = "deduplicator_evicted"
	Disabled         Reason = "deduplicator_disabled"
	MissingMessageID Reason = "dedup_unavailable_missing_message_id"
)
