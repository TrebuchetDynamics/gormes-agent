package signal

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/gateway"
)

// SendOptions captures Signal reply metadata for outbound sends.
type SendOptions struct {
	ReplyToMessageID string
}

// Client is the transport-neutral Signal surface used by the shared channel.
type Client interface {
	Events() <-chan InboundMessage
	SendDirect(ctx context.Context, recipientID, text string, opts SendOptions) (string, error)
	SendGroup(ctx context.Context, groupID, text string, opts SendOptions) (string, error)
	Close() error
}

// Bot adapts Signal traffic into the shared gateway channel contract.
type Bot struct {
	client Client
	log    *slog.Logger

	mu           sync.RWMutex
	replyTargets map[string]replyTarget
}

type replyTarget struct {
	chatType    ChatType
	recipientID string
	options     SendOptions
}

var _ gateway.Channel = (*Bot)(nil)

func New(client Client, log *slog.Logger) *Bot {
	if log == nil {
		log = slog.Default()
	}
	return &Bot{
		client:       client,
		log:          log,
		replyTargets: map[string]replyTarget{},
	}
}

func (b *Bot) Name() string { return platformName }

func (b *Bot) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	events := b.client.Events()
	for {
		select {
		case <-ctx.Done():
			b.closeClient()
			return nil
		case msg, ok := <-events:
			if !ok {
				b.closeClient()
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
				b.closeClient()
				return nil
			}
		}
	}
}

func (b *Bot) Send(ctx context.Context, chatID, text string) (string, error) {
	target, ok := b.lookupReplyTarget(chatID)
	if !ok {
		return "", fmt.Errorf("signal: no reply target for chat %q", chatID)
	}

	switch target.chatType {
	case ChatTypeDirect:
		return b.client.SendDirect(ctx, target.recipientID, text, target.options)
	case ChatTypeGroup:
		return b.client.SendGroup(ctx, target.recipientID, text, target.options)
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

func (b *Bot) closeClient() {
	_ = b.client.Close()
}
