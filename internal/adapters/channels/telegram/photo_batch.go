package telegram

import (
	"context"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	telegrambatching "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/batching"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func (b *Bot) enqueuePhotoBatch(ctx context.Context, inbox chan<- gateway.InboundEvent, ev gateway.InboundEvent, msg *tgbotapi.Message) bool {
	if msg == nil || !telegramInboundEventHasPhoto(ev) {
		return false
	}
	key := telegramPhotoBatchKey(ev, msg.MediaGroupID)
	delay := b.telegramMediaBatchDelay()

	return b.photoBatches.Enqueue(ctx, inbox, key, ev, delay)
}

func (b *Bot) cancelPhotoBatches() {
	b.photoBatches.Cancel()
}

func (b *Bot) pendingPhotoBatchCount() int {
	return b.photoBatches.Len()
}

func guardDoubleClosePanic(fn func()) {
	defer func() {
		recover()
	}()
	fn()
}

func (b *Bot) Disconnect(ctx context.Context) error {
	b.cancelPhotoBatches()
	b.cancelTextBatches(ctx)
	if b.client != nil {
		guardDoubleClosePanic(b.client.StopReceivingUpdates)
	}
	return nil
}

func (b *Bot) telegramMediaBatchDelay() time.Duration {
	if b.cfg.MediaBatchDelay > 0 {
		return b.cfg.MediaBatchDelay
	}
	return telegramDefaultMediaBatchDelay
}

func telegramInboundEventHasPhoto(ev gateway.InboundEvent) bool {
	return telegrambatching.InboundEventHasPhoto(ev)
}

func telegramPhotoBatchKey(ev gateway.InboundEvent, mediaGroupID string) string {
	return telegrambatching.PhotoBatchKey(ev, mediaGroupID)
}

func telegramMergePhotoBatch(first, next gateway.InboundEvent) gateway.InboundEvent {
	return telegrambatching.MergePhotoBatch(first, next)
}
