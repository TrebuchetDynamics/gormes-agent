package dingtalk

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	dingtalkClient "github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
)

// StreamClient implements the Client interface using DingTalk Stream Mode.
// It wraps the official DingTalk Go SDK to provide real WebSocket-based
// message receive and session-webhook reply over HTTP.
type StreamClient struct {
	sdk    *dingtalkClient.StreamClient
	events chan InboundMessage
	log    *slog.Logger
	startOnce sync.Once
	closeOnce sync.Once
	closeCh   chan struct{}
}

// NewStreamClient creates a DingTalk Stream Mode client. Call Start to
// begin receiving messages; the returned struct is a zero-value Client that
// satisfies the existing adapter contract.
func NewStreamClient(clientID, clientSecret string, log *slog.Logger) *StreamClient {
	if log == nil {
		log = slog.Default()
	}

	sc := &StreamClient{
		events:  make(chan InboundMessage, 64),
		log:     log,
		closeCh: make(chan struct{}, 1),
	}

	sdk := dingtalkClient.NewStreamClient(
		dingtalkClient.WithAppCredential(
			dingtalkClient.NewAppCredentialConfig(clientID, clientSecret),
		),
	)

	sdk.RegisterChatBotCallbackRouter(func(ctx context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error) {
		msg := InboundMessage{
			MessageID:        data.MsgId,
			ConversationID:   data.ConversationId,
			ConversationType: data.ConversationType,
			SenderStaffID:    data.SenderStaffId,
			SenderID:         data.SenderId,
			SenderNick:       data.SenderNick,
			Text:             data.Text.Content,
			SessionWebhook:   data.SessionWebhook,
			Mentioned:        data.IsInAtList,
		}
		select {
		case sc.events <- msg:
		case <-sc.closeCh:
		}
		return nil, nil
	})

	sc.sdk = sdk
	return sc
}

// Start connects to DingTalk Stream Mode and begins forwarding messages to
// the channel returned by Events. It is safe to call multiple times — only
// the first call starts the connection.
func (sc *StreamClient) Start(ctx context.Context) error {
	var startErr error
	sc.startOnce.Do(func() {
		startErr = sc.sdk.Start(ctx)
	})
	return startErr
}

// Events returns a read-only channel of inbound DingTalk messages. The
// channel is created in NewStreamClient and closed by Close.
func (sc *StreamClient) Events() <-chan InboundMessage {
	return sc.events
}

// SendReply posts a text reply to the session webhook URL provided by
// DingTalk for the target conversation. It uses the SDK's ChatbotReplier
// which makes an HTTP POST to the webhook.
func (sc *StreamClient) SendReply(ctx context.Context, webhook, text string) (string, error) {
	if webhook == "" {
		return "", errors.New("dingtalk: empty session webhook")
	}
	replier := chatbot.NewChatbotReplier()
	if err := replier.SimpleReplyText(ctx, webhook, []byte(text)); err != nil {
		return "", err
	}
	return "ok", nil
}

// Close terminates the Stream Mode connection and stops message delivery.
func (sc *StreamClient) Close() error {
	sc.closeOnce.Do(func() {
		close(sc.closeCh)
	})
	sc.sdk.Close()
	return nil
}
