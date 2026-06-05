package telegram

import (
	"context"
	"time"

	telegrambatching "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/batching"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

type telegramTextBatchEntry struct {
	event gateway.InboundEvent
	seq   uint64
	timer *time.Timer
	inbox chan<- gateway.InboundEvent
}

func (b *Bot) enqueueTextBatch(ctx context.Context, inbox chan<- gateway.InboundEvent, ev gateway.InboundEvent) bool {
	if !telegramInboundEventIsBatchableText(ev) {
		return false
	}
	delay := b.telegramTextBatchDelay()
	if delay <= 0 {
		return false
	}
	key := telegramTextBatchKey(ev)

	b.textMu.Lock()
	defer b.textMu.Unlock()
	if b.textBatches == nil {
		b.textBatches = map[string]*telegramTextBatchEntry{}
	}
	b.textSeq++
	seq := b.textSeq
	if existing := b.textBatches[key]; existing != nil {
		existing.event = telegramMergeTextBatch(existing.event, ev)
		if existing.timer != nil {
			existing.timer.Stop()
		}
		existing.seq = seq
		existing.inbox = inbox
		existing.timer = time.AfterFunc(delay, func() {
			b.flushTextBatch(ctx, inbox, key, seq)
		})
		return true
	}
	b.textBatches[key] = &telegramTextBatchEntry{
		event: ev,
		seq:   seq,
		inbox: inbox,
		timer: time.AfterFunc(delay, func() {
			b.flushTextBatch(ctx, inbox, key, seq)
		}),
	}
	return true
}

func (b *Bot) flushTextBatch(ctx context.Context, inbox chan<- gateway.InboundEvent, key string, seq uint64) {
	b.textMu.Lock()
	entry := b.textBatches[key]
	if entry == nil || entry.seq != seq {
		b.textMu.Unlock()
		return
	}
	delete(b.textBatches, key)
	ev := entry.event
	b.textMu.Unlock()

	select {
	case inbox <- ev:
	case <-ctx.Done():
	}
}

func (b *Bot) cancelTextBatches(ctx context.Context) {
	b.textMu.Lock()
	entries := make([]telegramTextBatchEntry, 0, len(b.textBatches))
	for _, entry := range b.textBatches {
		if entry == nil {
			continue
		}
		if entry.timer != nil {
			entry.timer.Stop()
		}
		entries = append(entries, *entry)
	}
	b.textBatches = map[string]*telegramTextBatchEntry{}
	b.textMu.Unlock()

	for _, entry := range entries {
		select {
		case entry.inbox <- entry.event:
		case <-ctx.Done():
			return
		}
	}
}

func (b *Bot) pendingTextBatchCount() int {
	b.textMu.Lock()
	defer b.textMu.Unlock()
	return len(b.textBatches)
}

func (b *Bot) telegramTextBatchDelay() time.Duration {
	return b.cfg.TextBatchDelay
}

func telegramInboundEventIsBatchableText(ev gateway.InboundEvent) bool {
	return telegrambatching.InboundEventIsBatchableText(ev)
}

func telegramTextBatchKey(ev gateway.InboundEvent) string {
	return telegrambatching.TextBatchKey(ev)
}

func telegramMergeTextBatch(first, next gateway.InboundEvent) gateway.InboundEvent {
	return telegrambatching.MergeTextBatch(first, next)
}
