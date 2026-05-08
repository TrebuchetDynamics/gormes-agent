// Package googlechat provides a fakeable Google Chat gateway seam. Live
// Pub/Sub, Chat REST, OAuth, and attachment delivery are intentionally
// deferred; this package owns the channel-neutral runtime contract first.
package googlechat

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	PlatformName     = "google_chat"
	MaxMessageLength = 4000
)

type Config struct{}

type Client interface {
	Events() <-chan []byte
	SendMessage(ctx context.Context, space, thread, text string) (string, error)
	Close() error
}

type Channel struct {
	cfg    Config
	client Client
	log    *slog.Logger
}

var _ gateway.Channel = (*Channel)(nil)

func NewChannel(cfg Config, client Client, log *slog.Logger) *Channel {
	if log == nil {
		log = slog.Default()
	}
	return &Channel{cfg: cfg, client: client, log: log}
}

func (c *Channel) Name() string { return PlatformName }

func (c *Channel) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	if c.client == nil {
		return errors.New("googlechat_transport_not_configured")
	}
	events := c.client.Events()
	for {
		select {
		case <-ctx.Done():
			_ = c.client.Close()
			return nil
		case payload, ok := <-events:
			if !ok {
				_ = c.client.Close()
				return nil
			}
			ev, ok := c.NormalizePubSubMessage(payload)
			if !ok {
				continue
			}
			select {
			case inbox <- ev:
			case <-ctx.Done():
				_ = c.client.Close()
				return nil
			}
		}
	}
}

func (c *Channel) Send(ctx context.Context, chatID, text string) (string, error) {
	return c.SendThread(ctx, chatID, "", text)
}

func (c *Channel) SendThread(ctx context.Context, chatID, threadID, text string) (string, error) {
	if c.client == nil {
		return "", errors.New("googlechat_transport_not_configured")
	}
	return c.client.SendMessage(ctx, chatID, threadID, text)
}

func (c *Channel) NormalizePubSubMessage(payload []byte) (gateway.InboundEvent, bool) {
	var envelope googleChatEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return gateway.InboundEvent{}, false
	}
	msg := envelope.Chat.MessagePayload.Message
	space := firstNonEmpty(msg.Space.Name, envelope.Chat.MessagePayload.Space.Name)
	thread := strings.TrimSpace(msg.Thread.Name)
	text := strings.TrimSpace(firstNonEmpty(msg.ArgumentText, msg.Text))
	senderID := strings.TrimSpace(msg.Sender.Name)
	msgID := strings.TrimSpace(msg.Name)
	if space == "" || senderID == "" || text == "" || msgID == "" {
		return gateway.InboundEvent{}, false
	}
	if strings.EqualFold(strings.TrimSpace(msg.Sender.Type), "BOT") {
		return gateway.InboundEvent{}, false
	}
	kind, body := gateway.ParseInboundText(text)
	return gateway.InboundEvent{
		Platform:  PlatformName,
		AccountID: strings.TrimSpace(msg.Sender.Email),
		ChatID:    space,
		ChatType:  googleChatType(firstNonEmpty(msg.Space.Type, envelope.Chat.MessagePayload.Space.Type)),
		UserID:    senderID,
		UserName:  strings.TrimSpace(msg.Sender.DisplayName),
		ThreadID:  thread,
		MsgID:     msgID,
		MessageID: msgID,
		Kind:      kind,
		Text:      body,
	}, true
}

type PluginInfo struct {
	Name             string
	Label            string
	RequiredEnv      []string
	AllowedUsersEnv  string
	AllowAllEnv      string
	InstallHint      string
	MaxMessageLength int
	SetupAvailable   bool
	PlatformHint     string
}

func PluginMetadata() PluginInfo {
	return PluginInfo{
		Name:             PlatformName,
		Label:            "Google Chat",
		RequiredEnv:      []string{"GOOGLE_CHAT_PROJECT_ID", "GOOGLE_CHAT_SUBSCRIPTION_NAME", "GOOGLE_CHAT_SERVICE_ACCOUNT_JSON"},
		AllowedUsersEnv:  "GOOGLE_CHAT_ALLOWED_USERS",
		AllowAllEnv:      "GOOGLE_CHAT_ALLOW_ALL_USERS",
		InstallHint:      "pip install 'hermes-agent[google_chat]'",
		MaxMessageLength: MaxMessageLength,
		SetupAvailable:   true,
		PlatformHint:     "You are on Google Chat. Keep responses concise and use the supported markdown subset.",
	}
}

type googleChatEnvelope struct {
	Chat struct {
		MessagePayload struct {
			Space   googleChatSpace   `json:"space"`
			Message googleChatMessage `json:"message"`
		} `json:"messagePayload"`
	} `json:"chat"`
}

type googleChatSpace struct {
	Name string `json:"name"`
	Type string `json:"spaceType"`
}

type googleChatMessage struct {
	Name         string           `json:"name"`
	Text         string           `json:"text"`
	ArgumentText string           `json:"argumentText"`
	Sender       googleChatSender `json:"sender"`
	Thread       struct {
		Name string `json:"name"`
	} `json:"thread"`
	Space googleChatSpace `json:"space"`
}

type googleChatSender struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
}

func googleChatType(spaceType string) string {
	switch strings.ToUpper(strings.TrimSpace(spaceType)) {
	case "DIRECT_MESSAGE":
		return "dm"
	case "GROUP_CHAT":
		return "group"
	default:
		return "space"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
