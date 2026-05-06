package telegram

import (
	"context"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func (b *Bot) enqueuePhotoBatch(ctx context.Context, inbox chan<- gateway.InboundEvent, ev gateway.InboundEvent, msg *tgbotapi.Message) bool {
	if msg == nil || !telegramInboundEventHasPhoto(ev) {
		return false
	}
	key := telegramPhotoBatchKey(ev, msg.MediaGroupID)
	delay := b.telegramMediaBatchDelay()

	b.photoMu.Lock()
	defer b.photoMu.Unlock()
	if b.photoBatches == nil {
		b.photoBatches = map[string]*telegramPhotoBatchEntry{}
	}
	b.photoSeq++
	seq := b.photoSeq
	if existing := b.photoBatches[key]; existing != nil {
		existing.event = telegramMergePhotoBatch(existing.event, ev)
		if existing.timer != nil {
			existing.timer.Stop()
		}
		existing.seq = seq
		existing.timer = time.AfterFunc(delay, func() {
			b.flushPhotoBatch(ctx, inbox, key, seq)
		})
		return true
	}
	b.photoBatches[key] = &telegramPhotoBatchEntry{
		event: ev,
		seq:   seq,
		timer: time.AfterFunc(delay, func() {
			b.flushPhotoBatch(ctx, inbox, key, seq)
		}),
	}
	return true
}

func (b *Bot) flushPhotoBatch(ctx context.Context, inbox chan<- gateway.InboundEvent, key string, seq uint64) {
	b.photoMu.Lock()
	entry := b.photoBatches[key]
	if entry == nil || entry.seq != seq {
		b.photoMu.Unlock()
		return
	}
	delete(b.photoBatches, key)
	ev := entry.event
	b.photoMu.Unlock()

	select {
	case inbox <- ev:
	case <-ctx.Done():
	}
}

func (b *Bot) cancelPhotoBatches() {
	b.photoMu.Lock()
	defer b.photoMu.Unlock()
	for _, entry := range b.photoBatches {
		if entry != nil && entry.timer != nil {
			entry.timer.Stop()
		}
	}
	b.photoBatches = map[string]*telegramPhotoBatchEntry{}
}

func (b *Bot) pendingPhotoBatchCount() int {
	b.photoMu.Lock()
	defer b.photoMu.Unlock()
	return len(b.photoBatches)
}

func (b *Bot) Disconnect(_ context.Context) error {
	b.cancelPhotoBatches()
	if b.client != nil {
		b.client.StopReceivingUpdates()
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
	for _, attachment := range ev.Attachments {
		if strings.EqualFold(strings.TrimSpace(attachment.Kind), "photo") {
			return true
		}
	}
	return false
}

func telegramPhotoBatchKey(ev gateway.InboundEvent, mediaGroupID string) string {
	chatID := strings.TrimSpace(ev.ChatID)
	if chatID == "" {
		chatID = "unknown-chat"
	}
	if mediaGroupID = strings.TrimSpace(mediaGroupID); mediaGroupID != "" {
		return "album:" + chatID + ":" + mediaGroupID
	}
	userID := strings.TrimSpace(ev.UserID)
	if userID == "" {
		userID = "unknown-user"
	}
	return "burst:" + chatID + ":" + userID
}

func telegramMergePhotoBatch(first, next gateway.InboundEvent) gateway.InboundEvent {
	if strings.TrimSpace(first.Text) == "" && strings.TrimSpace(next.Text) != "" {
		first.Text = next.Text
	}
	first.Attachments = append(first.Attachments, next.Attachments...)
	return first
}
