package yuanbao

import (
	"context"
	"log/slog"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// PlatformName is the channel-neutral platform identifier emitted by the
// Yuanbao gateway runtime.
const PlatformName = "yuanbao"

// InboundPush is the runtime-side push payload a Client hands to the Channel.
// It is the platform-neutral lift of a parsed Yuanbao envelope; protocol or
// markdown decoding remains the responsibility of upstream parsers in this
// package.
type InboundPush struct {
	ConversationID string
	MessageID      string
	AuthorRole     string
	Text           string
}

// SentMessage records a single outbound delivery. Tests assert against this
// shape; production clients fill it in when implementing the live transport.
type SentMessage struct {
	ConversationID string
	Text           string
}

// Client is the seam between the Yuanbao Channel and its transport. The
// runtime row binds a fake Client only; live login/QR/websocket implementation
// is intentionally deferred to a separate slice.
type Client interface {
	Connect(ctx context.Context) error
	Run(ctx context.Context, deliver func(context.Context, InboundPush)) error
	Send(ctx context.Context, conversationID, text string) (string, error)
}

// Config bounds Yuanbao runtime behavior. AllowedConversationID, when set,
// filters inbound deliveries to a single conversation in lockstep with the
// gateway manager's allowed-chats policy.
type Config struct {
	AllowedConversationID string
}

// Channel adapts a Yuanbao Client onto the shared gateway.Channel contract.
type Channel struct {
	cfg    Config
	client Client
	log    *slog.Logger
}

var _ gateway.Channel = (*Channel)(nil)

// NewChannel returns a Yuanbao gateway channel wrapping client.
func NewChannel(cfg Config, client Client, log *slog.Logger) *Channel {
	if log == nil {
		log = slog.Default()
	}
	return &Channel{cfg: cfg, client: client, log: log}
}

// Name returns the stable Yuanbao platform identifier.
func (c *Channel) Name() string { return PlatformName }

// Run connects the underlying client and pumps inbound pushes into inbox until
// ctx is cancelled. Connect failures are surfaced before any inbound delivery
// so callers can record degraded evidence.
func (c *Channel) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	if err := c.client.Connect(ctx); err != nil {
		return err
	}
	return c.client.Run(ctx, func(ctx context.Context, push InboundPush) {
		ev, ok := c.toInboundEvent(push)
		if !ok {
			return
		}
		select {
		case inbox <- ev:
		case <-ctx.Done():
		}
	})
}

// Send forwards text to conversationID through the underlying client.
func (c *Channel) Send(ctx context.Context, chatID, text string) (string, error) {
	return c.client.Send(ctx, chatID, text)
}

func (c *Channel) toInboundEvent(push InboundPush) (gateway.InboundEvent, bool) {
	convID := strings.TrimSpace(push.ConversationID)
	if convID == "" {
		return gateway.InboundEvent{}, false
	}
	if allowed := strings.TrimSpace(c.cfg.AllowedConversationID); allowed != "" && allowed != convID {
		return gateway.InboundEvent{}, false
	}
	text := strings.TrimSpace(push.Text)
	if text == "" {
		return gateway.InboundEvent{}, false
	}
	return gateway.InboundEvent{
		Platform:  PlatformName,
		ChatID:    convID,
		UserID:    convID,
		MsgID:     strings.TrimSpace(push.MessageID),
		MessageID: strings.TrimSpace(push.MessageID),
		Kind:      gateway.EventSubmit,
		Text:      text,
	}, true
}
