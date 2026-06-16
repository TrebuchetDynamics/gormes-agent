package tracker

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/dedup/evidence"
)

// Result reports whether a message was already tracked and any bounded-cache
// evidence a caller may surface later.
type Result struct {
	Duplicate bool
	Evidence  evidence.Reason
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
func (d *MessageDeduplicator) Track(messageID string) Result {
	if d == nil || d.maxSize <= 0 {
		return Result{Evidence: evidence.Disabled}
	}
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return Result{}
	}
	if d.seen == nil {
		d.seen = make(map[string]struct{}, d.maxSize)
	}
	if _, ok := d.seen[messageID]; ok {
		d.refresh(messageID)
		return Result{
			Duplicate: true,
			Evidence:  evidence.Duplicate,
		}
	}

	result := Result{}
	if len(d.order) >= d.maxSize {
		evictedID := d.order[0]
		delete(d.seen, evictedID)
		copy(d.order, d.order[1:])
		d.order = d.order[:len(d.order)-1]
		result.Evidence = evidence.Evicted
		result.EvictedID = evictedID
	}
	d.seen[messageID] = struct{}{}
	d.order = append(d.order, messageID)
	return result
}

func (d *MessageDeduplicator) refresh(messageID string) {
	for i, id := range d.order {
		if id != messageID {
			continue
		}
		copy(d.order[i:], d.order[i+1:])
		d.order[len(d.order)-1] = messageID
		return
	}
}
