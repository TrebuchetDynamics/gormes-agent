package signal

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// Bot adapts Signal traffic into the shared gateway channel contract. The
// adapter is transport-agnostic: Transport owns the signal-cli/bridge
// lifecycle and reconnect policy while Bot translates events onto the
// gateway contract and records reply targets for outbound sends.
type Bot struct {
	transport Transport
	log       *slog.Logger

	mu           sync.RWMutex
	replyTargets map[string]replyTarget
}

type replyTarget struct {
	chatType    ChatType
	recipientID string
	options     SendOptions
}

var _ gateway.Channel = (*Bot)(nil)

func New(transport Transport, log *slog.Logger) *Bot {
	if log == nil {
		log = slog.Default()
	}
	return &Bot{
		transport:    transport,
		log:          log,
		replyTargets: map[string]replyTarget{},
	}
}

func (b *Bot) Name() string { return platformName }

func (b *Bot) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	if err := b.transport.Start(ctx); err != nil {
		return fmt.Errorf("signal: start: %w", err)
	}
	defer b.closeTransport()

	events := b.transport.Events()
	errs := b.transport.Errors()
	for {
		select {
		case <-ctx.Done():
			return nil
		case recvErr, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if recvErr.Fatal {
				return fmt.Errorf("signal: receive: %w", recvErr.Err)
			}
			b.log.Warn("signal receive transient error", "err", recvErr.Err)
		case msg, ok := <-events:
			if !ok {
				return nil
			}
			normalized, ok := NormalizeInbound(msg)
			if !ok {
				continue
			}
			b.recordReplyTarget(normalized)

			select {
			case inbox <- normalized.Event:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// Send issues a plain-text reply to a previously-seen chat. Attachments go
// through SendAttachments.
func (b *Bot) Send(ctx context.Context, chatID, text string) (string, error) {
	return b.sendWithAttachments(ctx, chatID, text, nil)
}

// SendAttachments issues a reply with one or more attachments to a
// previously-seen chat. The outbound envelope reuses the recorded reply
// metadata so thread/reply semantics are preserved.
func (b *Bot) SendAttachments(ctx context.Context, chatID, text string, attachments []Attachment) (string, error) {
	return b.sendWithAttachments(ctx, chatID, text, attachments)
}

func (b *Bot) sendWithAttachments(ctx context.Context, chatID, text string, attachments []Attachment) (string, error) {
	target, ok := b.lookupReplyTarget(chatID)
	if !ok {
		return "", fmt.Errorf("signal: no reply target for chat %q", chatID)
	}

	opts := target.options
	if len(attachments) > 0 {
		opts.Attachments = append([]Attachment(nil), attachments...)
	}

	switch target.chatType {
	case ChatTypeDirect:
		return b.transport.SendDirect(ctx, target.recipientID, text, opts)
	case ChatTypeGroup:
		return b.transport.SendGroup(ctx, target.recipientID, text, opts)
	default:
		return "", fmt.Errorf("signal: unsupported chat type %q", target.chatType)
	}
}

func (b *Bot) recordReplyTarget(normalized NormalizedInbound) {
	recipientID, ok := replyRecipientID(normalized.Identity)
	if !ok {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.replyTargets[normalized.Event.ChatID] = replyTarget{
		chatType:    normalized.Identity.ChatType,
		recipientID: recipientID,
		options: SendOptions{
			ReplyToMessageID: normalized.Event.MsgID,
		},
	}
}

func (b *Bot) lookupReplyTarget(chatID string) (replyTarget, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	target, ok := b.replyTargets[chatID]
	return target, ok
}

func replyRecipientID(identity SessionIdentity) (string, bool) {
	switch identity.ChatType {
	case ChatTypeDirect:
		if identity.UserID == "" {
			return "", false
		}
		return identity.UserID, true
	case ChatTypeGroup:
		if identity.ChatIDAlt == "" {
			return "", false
		}
		return identity.ChatIDAlt, true
	default:
		return "", false
	}
}

func (b *Bot) closeTransport() {
	_ = b.transport.Close()
}
