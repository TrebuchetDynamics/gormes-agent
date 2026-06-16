package batching

import (
	"context"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// MergeFunc combines a pending batch event with the next event for the same
// key. The returned event replaces the pending value.
type MergeFunc func(first, next gateway.InboundEvent) gateway.InboundEvent

// EventBuffer owns the stateful debounce machinery shared by Telegram text and
// photo batching. Keying and merge policy stay in the caller/pure helpers; this
// type only owns timer, sequence, flush, cancel, and drain semantics.
type EventBuffer struct {
	mu      sync.Mutex
	seq     uint64
	entries map[string]*eventBufferEntry
	merge   MergeFunc
}

type eventBufferEntry struct {
	event gateway.InboundEvent
	seq   uint64
	timer *time.Timer
	inbox chan<- gateway.InboundEvent
}

func NewEventBuffer(merge MergeFunc) *EventBuffer {
	if merge == nil {
		merge = func(first, _ gateway.InboundEvent) gateway.InboundEvent { return first }
	}
	return &EventBuffer{
		entries: map[string]*eventBufferEntry{},
		merge:   merge,
	}
}

func (b *EventBuffer) Enqueue(ctx context.Context, inbox chan<- gateway.InboundEvent, key string, ev gateway.InboundEvent, delay time.Duration) bool {
	if b == nil || delay <= 0 {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.entries == nil {
		b.entries = map[string]*eventBufferEntry{}
	}
	b.seq++
	seq := b.seq
	if existing := b.entries[key]; existing != nil {
		existing.event = b.merge(existing.event, ev)
		if existing.timer != nil {
			existing.timer.Stop()
		}
		existing.seq = seq
		existing.inbox = inbox
		existing.timer = time.AfterFunc(delay, func() {
			b.Flush(ctx, inbox, key, seq)
		})
		return true
	}
	b.entries[key] = &eventBufferEntry{
		event: ev,
		seq:   seq,
		inbox: inbox,
		timer: time.AfterFunc(delay, func() {
			b.Flush(ctx, inbox, key, seq)
		}),
	}
	return true
}

func (b *EventBuffer) Flush(ctx context.Context, inbox chan<- gateway.InboundEvent, key string, seq uint64) {
	if b == nil {
		return
	}
	b.mu.Lock()
	entry := b.entries[key]
	if entry == nil || entry.seq != seq {
		b.mu.Unlock()
		return
	}
	delete(b.entries, key)
	ev := entry.event
	b.mu.Unlock()

	select {
	case inbox <- ev:
	case <-ctx.Done():
	}
}

func (b *EventBuffer) Cancel() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, entry := range b.entries {
		if entry != nil && entry.timer != nil {
			entry.timer.Stop()
		}
	}
	b.entries = map[string]*eventBufferEntry{}
}

func (b *EventBuffer) CancelAndDrain(ctx context.Context) {
	if b == nil {
		return
	}
	b.mu.Lock()
	entries := make([]eventBufferEntry, 0, len(b.entries))
	for _, entry := range b.entries {
		if entry == nil {
			continue
		}
		if entry.timer != nil {
			entry.timer.Stop()
		}
		entries = append(entries, *entry)
	}
	b.entries = map[string]*eventBufferEntry{}
	b.mu.Unlock()

	for _, entry := range entries {
		select {
		case entry.inbox <- entry.event:
		case <-ctx.Done():
			return
		}
	}
}

func (b *EventBuffer) Len() int {
	if b == nil {
		return 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}
