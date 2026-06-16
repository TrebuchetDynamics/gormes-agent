package telegram

import (
	"context"
	"time"

	telegrambatching "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/batching"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func (b *Bot) enqueueTextBatch(ctx context.Context, inbox chan<- gateway.InboundEvent, ev gateway.InboundEvent) bool {
	return b.enqueueTextBatchWithDelay(ctx, inbox, ev, b.telegramTextBatchDelay())
}

func (b *Bot) enqueueTextBatchWithDelay(ctx context.Context, inbox chan<- gateway.InboundEvent, ev gateway.InboundEvent, delay time.Duration) bool {
	if !telegramInboundEventIsBatchableText(ev) {
		return false
	}
	if delay <= 0 {
		return false
	}
	key := telegramTextBatchKey(ev)
	return b.textBatches.Enqueue(ctx, inbox, key, ev, delay)
}

func (b *Bot) cancelTextBatches(ctx context.Context) {
	b.textBatches.CancelAndDrain(ctx)
}

func (b *Bot) pendingTextBatchCount() int {
	return b.textBatches.Len()
}

func (b *Bot) telegramTextBatchDelay() time.Duration {
	return b.cfg.TextBatchDelay
}

func (b *Bot) telegramForwardedTextBatchDelay() time.Duration {
	if b.cfg.ForwardedTextBatchDelay > 0 {
		return b.cfg.ForwardedTextBatchDelay
	}
	return telegramDefaultForwardedTextBatchDelay
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
