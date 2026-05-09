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
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return gateway.InboundEvent{}, false
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return gateway.InboundEvent{}, false
	}
	msg, spacePayload, ok := googleChatMessageAndSpace(envelope, raw)
	if !ok {
		return gateway.InboundEvent{}, false
	}
	space := firstNonEmpty(msg.Space.Name, spacePayload.Name)
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
		ChatType:  googleChatType(firstNonEmpty(msg.Space.Type, spacePayload.Type)),
		UserID:    senderID,
		UserName:  strings.TrimSpace(msg.Sender.DisplayName),
		ThreadID:  thread,
		MsgID:     msgID,
		MessageID: msgID,
		Kind:      kind,
		Text:      body,
	}, true
}

func googleChatMessageAndSpace(envelope googleChatEnvelope, raw map[string]json.RawMessage) (googleChatMessage, googleChatSpace, bool) {
	msg := envelope.Chat.MessagePayload.Message
	space := envelope.Chat.MessagePayload.Space
	if googleChatMessagePresent(msg) || strings.TrimSpace(space.Name) != "" {
		return msg, space, true
	}

	if _, ok := raw["message"]; ok {
		if strings.ToUpper(strings.TrimSpace(envelope.Type)) != "MESSAGE" {
			return googleChatMessage{}, googleChatSpace{}, false
		}
		msg := envelope.Message
		space := envelope.Space
		if strings.TrimSpace(space.Name) == "" {
			space = msg.Space
		}
		return msg, space, true
	}

	_, hasEventType := raw["event_type"]
	_, hasSenderEmail := raw["sender_email"]
	if !hasEventType && !hasSenderEmail {
		return googleChatMessage{}, googleChatSpace{}, false
	}
	eventType := strings.ToUpper(strings.TrimSpace(envelope.EventType))
	if eventType == "" {
		eventType = "MESSAGE"
	}
	if eventType != "MESSAGE" {
		return googleChatMessage{}, googleChatSpace{}, false
	}
	senderEmail := strings.TrimSpace(envelope.SenderEmail)
	senderDisplay := strings.TrimSpace(firstNonEmpty(envelope.SenderDisplayName, senderEmail, "Unknown"))
	senderName := "users/relay-" + strings.ReplaceAll(strings.ReplaceAll(firstNonEmpty(senderEmail, "unknown"), "@", "_at_"), ".", "_")
	msg = googleChatMessage{
		Name:         strings.TrimSpace(envelope.MessageName),
		Text:         envelope.Text,
		ArgumentText: envelope.Text,
		Sender: googleChatSender{
			Name:        senderName,
			Email:       senderEmail,
			DisplayName: senderDisplay,
			Type:        googleChatSenderType(envelope.SenderType),
		},
		Space: googleChatSpace{
			Name: strings.TrimSpace(envelope.SpaceName),
			Type: strings.TrimSpace(envelope.SpaceType),
		},
	}
	msg.Thread.Name = strings.TrimSpace(envelope.ThreadName)
	return msg, msg.Space, true
}

func googleChatMessagePresent(msg googleChatMessage) bool {
	return firstNonEmpty(msg.Name, msg.Text, msg.ArgumentText, msg.Sender.Name, msg.Space.Name) != ""
}

func googleChatSenderType(value string) string {
	senderType := strings.ToUpper(strings.TrimSpace(value))
	switch senderType {
	case "BOT", "HUMAN":
		return senderType
	default:
		return "HUMAN"
	}
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
	Type              string            `json:"type"`
	Message           googleChatMessage `json:"message"`
	Space             googleChatSpace   `json:"space"`
	EventType         string            `json:"event_type"`
	SenderEmail       string            `json:"sender_email"`
	SenderDisplayName string            `json:"sender_display_name"`
	SenderType        string            `json:"sender_type"`
	Text              string            `json:"text"`
	SpaceName         string            `json:"space_name"`
	SpaceType         string            `json:"space_type"`
	ThreadName        string            `json:"thread_name"`
	MessageName       string            `json:"message_name"`
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
