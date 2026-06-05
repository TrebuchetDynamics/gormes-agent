package weixin

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	ChatTypeDirect = "direct"
	ChatTypeGroup  = "group"
)

// Config captures the first-pass Weixin contract surface.
type Config struct {
	DMPolicy       string
	AllowFrom      []string
	GroupPolicy    string
	GroupAllowFrom []string
}

// InboundMessage is the SDK-neutral Weixin poll event shape.
type InboundMessage struct {
	ChatType     string
	ChatID       string
	UserID       string
	UserName     string
	MessageID    string
	Text         string
	ContextToken string
}

// Client is the minimal Weixin surface used by the adapter.
type Client interface {
	Events() <-chan InboundMessage
	SendWithContext(ctx context.Context, chatID, contextToken, text string) (string, error)
	Close() error
}

type sendContext struct {
	chatType     string
	contextToken string
}

// Bot adapts Weixin traffic into the shared gateway channel contract.
type Bot struct {
	cfg           Config
	client        Client
	log           *slog.Logger
	allowedDMs    map[string]struct{}
	allowedGroups map[string]struct{}

	mu       sync.Mutex
	contexts map[string]sendContext
}

var _ gateway.Channel = (*Bot)(nil)

func New(cfg Config, client Client, log *slog.Logger) *Bot {
	if log == nil {
		log = slog.Default()
	}
	return &Bot{
		cfg:           cfg,
		client:        client,
		log:           log,
		allowedDMs:    channelutil.ToSet(cfg.AllowFrom),
		allowedGroups: channelutil.ToSet(cfg.GroupAllowFrom),
		contexts:      map[string]sendContext{},
	}
}

func (b *Bot) Name() string { return "weixin" }

func (b *Bot) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	return channelutil.RunInboundLoop(ctx, b.client, inbox, b.toInboundEvent)
}

func (b *Bot) Send(ctx context.Context, chatID, text string) (string, error) {
	meta, ok := b.lookupContext(chatID)
	if !ok || meta.contextToken == "" {
		return "", fmt.Errorf("weixin: no context token for chat %q", chatID)
	}
	return b.client.SendWithContext(ctx, chatID, meta.contextToken, text)
}

func (b *Bot) toInboundEvent(msg InboundMessage) (gateway.InboundEvent, bool) {
	text := strings.TrimSpace(msg.Text)
	chatID := strings.TrimSpace(msg.ChatID)
	userID := strings.TrimSpace(msg.UserID)
	if text == "" || chatID == "" || userID == "" {
		return gateway.InboundEvent{}, false
	}

	switch strings.TrimSpace(msg.ChatType) {
	case ChatTypeDirect:
		if !allowedByPolicy(b.cfg.DMPolicy, b.allowedDMs, userID, true) {
			return gateway.InboundEvent{}, false
		}
	case ChatTypeGroup:
		if !allowedByPolicy(b.cfg.GroupPolicy, b.allowedGroups, chatID, false) {
			return gateway.InboundEvent{}, false
		}
	default:
		return gateway.InboundEvent{}, false
	}

	b.rememberContext(chatID, sendContext{
		chatType:     strings.TrimSpace(msg.ChatType),
		contextToken: strings.TrimSpace(msg.ContextToken),
	})

	kind, body := gateway.ParseInboundText(text)
	return gateway.InboundEvent{
		Platform:  "weixin",
		ChatID:    chatID,
		ChatType:  strings.TrimSpace(msg.ChatType),
		UserID:    userID,
		UserName:  strings.TrimSpace(msg.UserName),
		MsgID:     strings.TrimSpace(msg.MessageID),
		MessageID: strings.TrimSpace(msg.MessageID),
		Kind:      kind,
		Text:      body,
	}, true
}

func (b *Bot) rememberContext(chatID string, meta sendContext) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.contexts[chatID] = meta
}

func (b *Bot) lookupContext(chatID string) (sendContext, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	meta, ok := b.contexts[chatID]
	return meta, ok
}

func allowedByPolicy(policy string, allowed map[string]struct{}, value string, isDM bool) bool {
	// weixin defaults: empty policy = open for DM, disabled for group
	if channelutil.NormalizedPolicy(policy) == "open" && !isDM {
		return false
	}
	return channelutil.AllowedByPolicy(policy, allowed, value)
}
