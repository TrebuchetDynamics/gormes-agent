package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// Config drives the Telegram channel. AllowedChatID and discovery are still
// kept here so SDK-specific entrypoints can reuse the typed values.
type Config struct {
	AllowedChatID     int64
	FirstRunDiscovery bool
	// RequireMention gates group inbound messages so only those addressed to
	// BotUsername (mention or bot_command @suffix) reach the gateway.
	RequireMention bool
	// BotUsername is the bare bot handle used to recognise group mentions.
	BotUsername string
}

// Bot implements gateway.Channel plus the editing capabilities the shared
// manager uses for streamed responses.
type Bot struct {
	cfg    Config
	client telegramClient
	log    *slog.Logger
}

var _ gateway.Channel = (*Bot)(nil)
var _ gateway.MessageEditor = (*Bot)(nil)
var _ gateway.MessageDeleter = (*Bot)(nil)
var _ gateway.PlaceholderCapable = (*Bot)(nil)

func New(cfg Config, client telegramClient, log *slog.Logger) *Bot {
	if log == nil {
		log = slog.Default()
	}
	return &Bot{cfg: cfg, client: client, log: log}
}

func (b *Bot) Name() string { return "telegram" }

func (b *Bot) registerCommands() error {
	commands := gateway.TelegramBotCommands()
	botCommands := make([]tgbotapi.BotCommand, 0, len(commands))
	for _, cmd := range commands {
		name := strings.TrimSpace(cmd.Name)
		desc := strings.TrimSpace(cmd.Description)
		if name == "" || desc == "" {
			continue
		}
		botCommands = append(botCommands, tgbotapi.BotCommand{Command: name, Description: desc})
	}
	if len(botCommands) == 0 {
		return nil
	}
	_, err := b.client.Request(tgbotapi.NewSetMyCommands(botCommands...))
	return err
}

func (b *Bot) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	if err := b.registerCommands(); err != nil {
		b.log.Warn("telegram setMyCommands failed", "err", err)
	}

	ucfg := tgbotapi.NewUpdate(0)
	ucfg.Timeout = 30
	updates := b.client.GetUpdatesChan(ucfg)

	for {
		select {
		case <-ctx.Done():
			b.client.StopReceivingUpdates()
			return nil
		case u, ok := <-updates:
			if !ok {
				return nil
			}
			if ev, ok := b.toInboundEvent(u); ok {
				select {
				case inbox <- ev:
				case <-ctx.Done():
					return nil
				}
			}
		}
	}
}

func (b *Bot) toInboundEvent(u tgbotapi.Update) (gateway.InboundEvent, bool) {
	if u.Message == nil {
		return gateway.InboundEvent{}, false
	}

	chatID := u.Message.Chat.ID
	text, attachments := telegramInboundTextAndAttachments(u.Message)

	if b.cfg.RequireMention && telegramIsGroupChat(u.Message.Chat) {
		if !telegramGroupMentionGateAddressed(text, u.Message.Entities, b.cfg.BotUsername, true) {
			return gateway.InboundEvent{}, false
		}
	}

	kind, body := gateway.ParseInboundText(text)

	var userID string
	if u.Message.From != nil {
		userID = strconv.FormatInt(u.Message.From.ID, 10)
	}

	return gateway.InboundEvent{
		Platform:    "telegram",
		ChatID:      strconv.FormatInt(chatID, 10),
		UserID:      userID,
		MsgID:       strconv.Itoa(u.Message.MessageID),
		Kind:        kind,
		Text:        body,
		Attachments: attachments,
	}, true
}

func telegramInboundTextAndAttachments(msg *tgbotapi.Message) (string, []gateway.Attachment) {
	if msg == nil {
		return "", nil
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		text = strings.TrimSpace(msg.Caption)
	}

	var attachments []gateway.Attachment
	var markers []string
	if msg.Voice != nil {
		marker, attachment := telegramVoiceAttachment(msg.Voice)
		markers = append(markers, marker)
		attachments = append(attachments, attachment)
	}
	if msg.Audio != nil {
		marker, attachment := telegramAudioAttachment(msg.Audio)
		markers = append(markers, marker)
		attachments = append(attachments, attachment)
	}

	for _, marker := range markers {
		if marker == "" {
			continue
		}
		if text == "" {
			text = marker
		} else {
			text += "\n\n" + marker
		}
	}
	return text, attachments
}

func telegramVoiceAttachment(voice *tgbotapi.Voice) (string, gateway.Attachment) {
	if voice == nil {
		return "", gateway.Attachment{}
	}
	mediaType := strings.TrimSpace(voice.MimeType)
	attachment := gateway.Attachment{
		Kind:      "voice",
		MediaType: mediaType,
		SourceID:  strings.TrimSpace(voice.FileID),
	}
	return telegramAudioMarker("voice", voice.Duration, mediaType, ""), attachment
}

func telegramAudioAttachment(audio *tgbotapi.Audio) (string, gateway.Attachment) {
	if audio == nil {
		return "", gateway.Attachment{}
	}
	mediaType := strings.TrimSpace(audio.MimeType)
	fileName := strings.TrimSpace(audio.FileName)
	attachment := gateway.Attachment{
		Kind:      "audio",
		MediaType: mediaType,
		FileName:  fileName,
		SourceID:  strings.TrimSpace(audio.FileID),
	}
	return telegramAudioMarker("audio", audio.Duration, mediaType, fileName), attachment
}

func telegramAudioMarker(kind string, duration int, mediaType, fileName string) string {
	var parts []string
	if duration > 0 {
		parts = append(parts, fmt.Sprintf("duration=%ds", duration))
	}
	if mediaType = strings.TrimSpace(mediaType); mediaType != "" {
		parts = append(parts, "mime="+mediaType)
	}
	if fileName = strings.TrimSpace(fileName); fileName != "" {
		parts = append(parts, "file="+fileName)
	}
	if len(parts) == 0 {
		return fmt.Sprintf("[Telegram %s message attached]", kind)
	}
	return fmt.Sprintf("[Telegram %s message attached: %s]", kind, strings.Join(parts, ", "))
}

func (b *Bot) Send(ctx context.Context, chatID, text string) (string, error) {
	_ = ctx
	id, err := parseChatID(chatID)
	if err != nil {
		return "", err
	}
	msg, err := b.client.Send(tgbotapi.NewMessage(id, text))
	if err != nil {
		return "", err
	}
	return strconv.Itoa(msg.MessageID), nil
}

func (b *Bot) SendReply(ctx context.Context, chatID, replyToMsgID, text string) (string, error) {
	_ = ctx
	id, err := parseChatID(chatID)
	if err != nil {
		return "", err
	}
	replyID, err := strconv.Atoi(replyToMsgID)
	if err != nil {
		return "", fmt.Errorf("telegram: invalid reply msgID %q: %w", replyToMsgID, err)
	}
	msgCfg := tgbotapi.NewMessage(id, text)
	msgCfg.ReplyToMessageID = replyID
	msg, err := b.client.Send(msgCfg)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(msg.MessageID), nil
}

func (b *Bot) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	return b.Send(ctx, chatID, "⏳")
}

func (b *Bot) SendReplyPlaceholder(ctx context.Context, chatID, replyToMsgID string) (string, error) {
	return b.SendReply(ctx, chatID, replyToMsgID, "⏳")
}

func (b *Bot) EditMessage(ctx context.Context, chatID, msgID, text string) error {
	_ = ctx
	cid, err := parseChatID(chatID)
	if err != nil {
		return err
	}
	mid, err := strconv.Atoi(msgID)
	if err != nil {
		return fmt.Errorf("telegram: invalid msgID %q: %w", msgID, err)
	}
	_, err = b.client.Send(tgbotapi.NewEditMessageText(cid, mid, text))
	return err
}

func (b *Bot) DeleteMessage(ctx context.Context, chatID, msgID string) error {
	_ = ctx
	cid, err := parseChatID(chatID)
	if err != nil {
		return err
	}
	mid, err := strconv.Atoi(msgID)
	if err != nil {
		return fmt.Errorf("telegram: invalid msgID %q: %w", msgID, err)
	}
	if err := b.client.DeleteMessage(cid, mid); err != nil {
		b.log.Debug("telegram delete message failed", "chat_id", cid, "message_id", mid, "err", err)
		return err
	}
	return nil
}

// SendToChat is retained for the cron delivery sink, which addresses Telegram
// using the native int64 chat identifier.
func (b *Bot) SendToChat(ctx context.Context, chatID int64, text string) error {
	_ = ctx
	_, err := b.client.Send(tgbotapi.NewMessage(chatID, text))
	return err
}

func parseChatID(s string) (int64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("telegram: invalid chat ID %q: %w", s, err)
	}
	return v, nil
}
