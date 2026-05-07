package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// Config drives the Telegram channel. AllowedChatID and discovery are still
// kept here so SDK-specific entrypoints can reuse the typed values.
type Config struct {
	AllowedChatID     int64
	AllowedUserIDs    []int64
	FirstRunDiscovery bool
	// AttachmentCacheDir stores Telegram downloads before the channel emits
	// normalized attachments to the gateway. Empty uses the user cache dir.
	AttachmentCacheDir string
	// MediaBatchDelay is the debounce window for Telegram photo bursts and
	// media_group albums. Empty uses the Hermes-compatible default.
	MediaBatchDelay time.Duration
	// TextBatchDelay is the quiet period before plain text updates are
	// dispatched. Empty preserves the existing immediate-dispatch path.
	TextBatchDelay time.Duration
	// AudioTranscriber optionally turns Telegram voice/audio attachments into
	// text before they reach the gateway. When nil or degraded, the adapter
	// still emits deterministic attachment markers instead of blank turns.
	AudioTranscriber AudioTranscriber
	// StickerCachePath stores Hermes-compatible Telegram sticker descriptions.
	// Empty uses the user cache dir.
	StickerCachePath string
	// StickerVisionAnalyzer optionally describes static stickers on cache miss.
	// Tests inject a fake analyzer; production may leave it nil for degraded
	// placeholder behavior until a live vision provider is wired.
	StickerVisionAnalyzer StickerVisionAnalyzer
	// RequireMention gates group inbound messages so only those addressed to
	// BotUsername (mention or bot_command @suffix) reach the gateway.
	RequireMention bool
	// BotUsername is the bare bot handle used to recognise group mentions.
	BotUsername string
	// BotUserID is optional and lets Telegram text_mention entities target the
	// bot by user ID when Telegram does not emit an @username mention.
	BotUserID int64
	// DynamicCommands are optional runtime-discovered commands (for example
	// enabled skill slash commands) appended to the canonical Hermes menu.
	DynamicCommands  []gateway.PlatformCommand
	ApprovalResolver gateway.ApprovalResolver
	// TokenLockDir stores machine-local same-token polling locks. Empty uses
	// the gateway package default; cmd/gormes passes config.GatewayLockDir.
	TokenLockDir string
	// PollingConflictRetryDelay bounds tests and preserves Hermes' retry
	// ladder in production when zero.
	PollingConflictRetryDelay time.Duration
	tokenLocker               telegramTokenLocker
}

// Bot implements gateway.Channel plus the editing capabilities the shared
// manager uses for streamed responses.
type Bot struct {
	cfg    Config
	client telegramClient
	log    *slog.Logger

	photoMu      sync.Mutex
	photoSeq     uint64
	photoBatches map[string]*telegramPhotoBatchEntry

	textMu      sync.Mutex
	textSeq     uint64
	textBatches map[string]*telegramTextBatchEntry

	approvalMu     sync.Mutex
	approvalNextID uint64
	approvalState  map[uint64]telegramApprovalState

	startupMu            sync.Mutex
	startupLock          telegramTokenLock
	pollingConflictCount int
}

var _ gateway.Channel = (*Bot)(nil)
var _ gateway.MessageEditor = (*Bot)(nil)
var _ gateway.MessageDeleter = (*Bot)(nil)
var _ gateway.ThreadSender = (*Bot)(nil)
var _ gateway.ThreadReplySender = (*Bot)(nil)
var _ gateway.MediaSender = (*Bot)(nil)
var _ gateway.PlaceholderCapable = (*Bot)(nil)
var _ gateway.ReplyPlaceholderCapable = (*Bot)(nil)
var _ gateway.ThreadPlaceholderCapable = (*Bot)(nil)
var _ gateway.ThreadReplyPlaceholderCapable = (*Bot)(nil)
var _ gateway.TypingCapable = (*Bot)(nil)
var _ gateway.ThreadTypingActionCapable = (*Bot)(nil)
var _ gateway.DisconnectCapable = (*Bot)(nil)
var _ gateway.ReactionCapable = (*Bot)(nil)

const telegramCommandLimit = 100
const telegramTypingRefreshInterval = 4 * time.Second
const telegramReactionEndpoint = "setMessageReaction"
const telegramSendMessageEndpoint = "sendMessage"
const telegramSendChatActionEndpoint = "sendChatAction"
const telegramGeneralTopicThreadID = "1"
const maxSendRetries = 3

func New(cfg Config, client telegramClient, log *slog.Logger) *Bot {
	if log == nil {
		log = slog.Default()
	}
	return &Bot{
		cfg:           cfg,
		client:        client,
		log:           log,
		photoBatches:  map[string]*telegramPhotoBatchEntry{},
		textBatches:   map[string]*telegramTextBatchEntry{},
		approvalState: map[uint64]telegramApprovalState{},
	}
}

func (b *Bot) Name() string { return "telegram" }

func telegramReactionsEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("TELEGRAM_REACTIONS"))) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

func (b *Bot) OnProcessingStart(ctx context.Context, chatID, msgID string) error {
	if !telegramReactionsEnabled() {
		return nil
	}
	b.setReaction(ctx, chatID, msgID, "👀")
	return nil
}

func (b *Bot) OnProcessingComplete(ctx context.Context, chatID, msgID string, outcome gateway.ProcessingOutcome) error {
	if !telegramReactionsEnabled() || outcome == gateway.ProcessingOutcomeCancelled {
		return nil
	}
	emoji := "👎"
	if outcome == gateway.ProcessingOutcomeSuccess {
		emoji = "👍"
	}
	b.setReaction(ctx, chatID, msgID, emoji)
	return nil
}

func (b *Bot) setReaction(_ context.Context, chatID, msgID, emoji string) {
	chatID = strings.TrimSpace(chatID)
	msgID = strings.TrimSpace(msgID)
	if chatID == "" || msgID == "" || b.client == nil {
		return
	}
	chatInt, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		b.log.Debug("telegram reaction skipped: invalid chat id")
		return
	}
	msgInt, err := strconv.Atoi(msgID)
	if err != nil {
		b.log.Debug("telegram reaction skipped: invalid message id")
		return
	}
	params := tgbotapi.Params{}
	params.AddNonZero64("chat_id", chatInt)
	params.AddNonZero("message_id", msgInt)
	if err := params.AddInterface("reaction", []map[string]string{{
		"type":  "emoji",
		"emoji": emoji,
	}}); err != nil {
		b.log.Debug("telegram reaction skipped: encode reaction", "err", err)
		return
	}
	if _, err := b.client.UploadFiles(telegramReactionEndpoint, params, nil); err != nil {
		b.log.Debug("telegram reaction failed", "err", err)
	}
}

func (b *Bot) registerCommands() error {
	commands := gateway.TelegramBotCommandsWith(b.cfg.DynamicCommands)
	if hidden := len(commands) - telegramCommandLimit; hidden > 0 {
		b.log.Info("telegram setMyCommands capped at platform limit", "limit", telegramCommandLimit, "hidden_count", hidden)
	}
	botCommands := make([]tgbotapi.BotCommand, 0, len(commands))
	for _, cmd := range commands {
		if len(botCommands) >= telegramCommandLimit {
			break
		}
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
	if err := b.prepareStartup(ctx); err != nil {
		return err
	}
	defer b.releaseStartupTokenLock(context.Background())
	if err := b.registerCommands(); err != nil {
		b.log.Warn("telegram setMyCommands failed", "err", err)
	}
	defer b.cancelPhotoBatches()
	defer b.cancelTextBatches(ctx)

	ucfg := tgbotapi.NewUpdate(0)
	ucfg.Timeout = 30

	for {
		if err := ctx.Err(); err != nil {
			b.client.StopReceivingUpdates()
			return nil
		}
		updates, err := b.client.GetUpdates(ctx, ucfg)
		if err != nil {
			again, err := b.handlePollingError(ctx, err)
			if err != nil {
				b.client.StopReceivingUpdates()
				return err
			}
			if !again {
				b.client.StopReceivingUpdates()
				return nil
			}
			continue
		}
		if len(updates) > 0 {
			b.pollingConflictCount = 0
		}
		for _, u := range updates {
			if u.UpdateID >= ucfg.Offset {
				ucfg.Offset = u.UpdateID + 1
			}
			if err := b.handleUpdate(ctx, inbox, u); err != nil {
				b.client.StopReceivingUpdates()
				return err
			}
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, inbox chan<- gateway.InboundEvent, u tgbotapi.Update) error {
	if u.CallbackQuery != nil {
		if b.handleCallbackQuery(ctx, u.CallbackQuery) {
			return nil
		}
		return nil
	}
	if ev, ok := b.toInboundEvent(ctx, u); ok {
		if b.enqueuePhotoBatch(ctx, inbox, ev, u.Message) {
			return nil
		}
		if b.enqueueTextBatch(ctx, inbox, ev) {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case inbox <- ev:
		}
	}
	return nil
}

func (b *Bot) toInboundEvent(ctx context.Context, u tgbotapi.Update) (gateway.InboundEvent, bool) {
	if u.Message == nil {
		return gateway.InboundEvent{}, false
	}

	chatID := u.Message.Chat.ID
	text, attachments := b.telegramInboundTextAndAttachments(ctx, u.Message)

	if b.cfg.RequireMention && telegramIsGroupChat(u.Message.Chat) {
		if !telegramGroupMentionGateMessageAddressed(u.Message, b.cfg.BotUsername, b.cfg.BotUserID, true) {
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
		ChatType:    telegramChatType(u.Message.Chat),
		UserID:      userID,
		MsgID:       strconv.Itoa(u.Message.MessageID),
		MessageID:   strconv.Itoa(u.Message.MessageID),
		Kind:        kind,
		Text:        body,
		Attachments: attachments,
	}, true
}

func telegramChatType(chat *tgbotapi.Chat) string {
	if chat == nil {
		return ""
	}
	return strings.TrimSpace(chat.Type)
}

func (b *Bot) telegramInboundTextAndAttachments(ctx context.Context, msg *tgbotapi.Message) (string, []gateway.Attachment) {
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
		if transcript, err := b.transcribeTelegramAudio(ctx, AudioInput{
			Kind:      "voice",
			FileID:    attachment.SourceID,
			MediaType: attachment.MediaType,
			Duration:  time.Duration(msg.Voice.Duration) * time.Second,
		}); err == nil && strings.TrimSpace(transcript) != "" {
			markers = append(markers, telegramAudioTranscriptMarker("voice", transcript))
		} else if err != nil {
			attachment.Error = sanitizeTelegramAudioError(err)
			b.log.Warn("telegram audio transcription unavailable", "kind", "voice", "err", attachment.Error)
		}
		markers = append(markers, marker)
		attachments = append(attachments, attachment)
	}
	if msg.Audio != nil {
		marker, attachment := telegramAudioAttachment(msg.Audio)
		if transcript, err := b.transcribeTelegramAudio(ctx, AudioInput{
			Kind:      "audio",
			FileID:    attachment.SourceID,
			MediaType: attachment.MediaType,
			FileName:  attachment.FileName,
			Duration:  time.Duration(msg.Audio.Duration) * time.Second,
		}); err == nil && strings.TrimSpace(transcript) != "" {
			markers = append(markers, telegramAudioTranscriptMarker("audio", transcript))
		} else if err != nil {
			attachment.Error = sanitizeTelegramAudioError(err)
			b.log.Warn("telegram audio transcription unavailable", "kind", "audio", "err", attachment.Error)
		}
		markers = append(markers, marker)
		attachments = append(attachments, attachment)
	}
	var prefixes []string
	if msg.Document != nil {
		prefix, marker, attachment := b.telegramDocumentAttachment(ctx, msg.Document)
		if prefix != "" {
			prefixes = append(prefixes, prefix)
		}
		if marker != "" {
			markers = append(markers, marker)
		}
		if attachment != nil {
			attachments = append(attachments, *attachment)
		}
	}
	if msg.Video != nil {
		marker, attachment := b.telegramVideoMessageAttachment(ctx, msg.Video)
		if marker != "" {
			markers = append(markers, marker)
		}
		if attachment != nil {
			attachments = append(attachments, *attachment)
		}
	}
	if len(msg.Photo) > 0 {
		marker, attachment := b.telegramPhotoMessageAttachment(ctx, msg.Photo)
		if marker != "" {
			markers = append(markers, marker)
		}
		if attachment != nil {
			attachments = append(attachments, *attachment)
		}
	}
	if msg.Sticker != nil {
		marker := b.telegramStickerMarker(ctx, msg.Sticker)
		if marker != "" {
			markers = append(markers, marker)
		}
	}

	if len(prefixes) > 0 {
		prefix := strings.Join(prefixes, "\n\n")
		if text == "" {
			text = prefix
		} else {
			text = prefix + "\n\n" + text
		}
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

func telegramInboundTextAndAttachments(msg *tgbotapi.Message) (string, []gateway.Attachment) {
	return New(Config{}, nil, nil).telegramInboundTextAndAttachments(context.Background(), msg)
}

func telegramAudioTranscriptMarker(kind, transcript string) string {
	label := "voice message"
	if strings.EqualFold(strings.TrimSpace(kind), "audio") {
		label = "audio message"
	}
	return fmt.Sprintf("[The user sent a %s~ Here's what they said: %q]", label, strings.Join(strings.Fields(transcript), " "))
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
	msgCfg := tgbotapi.NewMessage(id, text)
	msgCfg.ParseMode = tgbotapi.ModeMarkdownV2
	msg, err := b.sendWithParseFallback(msgCfg)
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
	msgCfg.ParseMode = tgbotapi.ModeMarkdownV2
	msgCfg.ReplyToMessageID = replyID
	msg, err := b.sendWithParseFallback(msgCfg)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(msg.MessageID), nil
}

func (b *Bot) SendThread(ctx context.Context, chatID, threadID, text string) (string, error) {
	if strings.TrimSpace(threadID) == "" {
		return b.Send(ctx, chatID, text)
	}
	id, err := parseChatID(chatID)
	if err != nil {
		return "", err
	}
	thread, includeThread, err := telegramThreadIDForTextSend(threadID)
	if err != nil {
		return "", err
	}
	params := telegramSendMessageParams(id, 0, text, tgbotapi.ModeMarkdownV2)
	if includeThread {
		params.AddNonZero("message_thread_id", thread)
	}

	var lastErr error
	for attempt := 0; attempt < maxSendRetries; attempt++ {
		msg, err := b.sendRawMessageWithParseFallback(ctx, params)
		if err == nil {
			return strconv.Itoa(msg.MessageID), nil
		}
		lastErr = err

		if isThreadNotFoundError(err) && includeThread {
			delete(params, "message_thread_id")
			includeThread = false
			continue
		}
		if isTimedOutError(err) {
			return "", err
		}
		if !isTransientNetworkError(err) {
			return "", err
		}
	}
	return "", lastErr
}

func (b *Bot) SendThreadReply(ctx context.Context, chatID, threadID, replyToMsgID, text string) (string, error) {
	if strings.TrimSpace(threadID) == "" {
		return b.SendReply(ctx, chatID, replyToMsgID, text)
	}
	id, err := parseChatID(chatID)
	if err != nil {
		return "", err
	}
	replyID, err := strconv.Atoi(replyToMsgID)
	if err != nil {
		return "", fmt.Errorf("telegram: invalid reply msgID %q: %w", replyToMsgID, err)
	}
	thread, includeThread, err := telegramThreadIDForTextSend(threadID)
	if err != nil {
		return "", err
	}
	params := telegramSendMessageParams(id, replyID, text, tgbotapi.ModeMarkdownV2)
	if includeThread {
		params.AddNonZero("message_thread_id", thread)
	}

	var lastErr error
	for attempt := 0; attempt < maxSendRetries; attempt++ {
		msg, err := b.sendRawMessageWithParseFallback(ctx, params)
		if err == nil {
			return strconv.Itoa(msg.MessageID), nil
		}
		lastErr = err

		if isThreadNotFoundError(err) && includeThread {
			delete(params, "message_thread_id")
			includeThread = false
			continue
		}
		if isTimedOutError(err) {
			return "", err
		}
		if !isTransientNetworkError(err) {
			return "", err
		}
	}
	return "", lastErr
}

func (b *Bot) SendMedia(ctx context.Context, chatID, replyToMsgID string, media gateway.OutboundMedia) (string, error) {
	_ = ctx
	id, err := parseChatID(chatID)
	if err != nil {
		return "", err
	}
	mediaPath := strings.TrimSpace(media.Path)
	if mediaPath == "" {
		return "", fmt.Errorf("telegram: media path is required")
	}
	replyID := 0
	if strings.TrimSpace(replyToMsgID) != "" {
		replyID, err = strconv.Atoi(replyToMsgID)
		if err != nil {
			return "", fmt.Errorf("telegram: invalid reply msgID %q: %w", replyToMsgID, err)
		}
	}
	if strings.TrimSpace(media.ThreadID) != "" {
		return b.sendMediaWithThread(id, replyID, media)
	}
	var msg tgbotapi.Message
	switch gateway.ClassifyOutboundMedia(media) {
	case gateway.OutboundMediaKindAudio:
		if media.AsVoice {
			cfg := tgbotapi.NewVoice(id, tgbotapi.FilePath(mediaPath))
			cfg.ReplyToMessageID = replyID
			msg, err = b.client.Send(cfg)
		} else {
			cfg := tgbotapi.NewAudio(id, tgbotapi.FilePath(mediaPath))
			cfg.ReplyToMessageID = replyID
			msg, err = b.client.Send(cfg)
		}
	case gateway.OutboundMediaKindImage:
		cfg := tgbotapi.NewPhoto(id, tgbotapi.FilePath(mediaPath))
		cfg.ReplyToMessageID = replyID
		msg, err = b.client.Send(cfg)
	case gateway.OutboundMediaKindDocument:
		cfg := tgbotapi.NewDocument(id, tgbotapi.FilePath(mediaPath))
		cfg.ReplyToMessageID = replyID
		msg, err = b.client.Send(cfg)
	case gateway.OutboundMediaKindVideo:
		cfg := tgbotapi.NewVideo(id, tgbotapi.FilePath(mediaPath))
		cfg.ReplyToMessageID = replyID
		msg, err = b.client.Send(cfg)
	default:
		return "", fmt.Errorf("telegram: unsupported media type")
	}
	if err != nil {
		return "", err
	}
	return strconv.Itoa(msg.MessageID), nil
}

func (b *Bot) sendMediaWithThread(chatID int64, replyID int, media gateway.OutboundMedia) (string, error) {
	threadID, err := strconv.Atoi(strings.TrimSpace(media.ThreadID))
	if err != nil {
		return "", fmt.Errorf("telegram: invalid thread ID %q: %w", media.ThreadID, err)
	}
	mediaPath := strings.TrimSpace(media.Path)
	endpoint, field, err := telegramMediaUploadEndpoint(media)
	if err != nil {
		return "", err
	}
	params := tgbotapi.Params{}
	params.AddNonZero64("chat_id", chatID)
	params.AddNonZero("reply_to_message_id", replyID)
	params.AddNonZero("message_thread_id", threadID)
	if gateway.ClassifyOutboundMedia(media) == gateway.OutboundMediaKindVideo {
		params.AddBool("supports_streaming", true)
	}
	resp, err := b.client.UploadFiles(endpoint, params, []tgbotapi.RequestFile{{
		Name: field,
		Data: tgbotapi.FilePath(mediaPath),
	}})
	if err != nil {
		return "", err
	}
	var msg tgbotapi.Message
	if resp != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, &msg); err != nil {
			return "", err
		}
	}
	return strconv.Itoa(msg.MessageID), nil
}

func telegramThreadIDForTextSend(threadID string) (int, bool, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" || threadID == telegramGeneralTopicThreadID {
		return 0, false, nil
	}
	thread, err := strconv.Atoi(threadID)
	if err != nil {
		return 0, false, fmt.Errorf("telegram: invalid thread ID %q: %w", threadID, err)
	}
	return thread, true, nil
}

func telegramThreadIDForAction(threadID string) (int, bool, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return 0, false, nil
	}
	thread, err := strconv.Atoi(threadID)
	if err != nil {
		return 0, false, fmt.Errorf("telegram: invalid thread ID %q: %w", threadID, err)
	}
	return thread, true, nil
}

func telegramSendMessageParams(chatID int64, replyID int, text, parseMode string) tgbotapi.Params {
	params := tgbotapi.Params{}
	params.AddNonZero64("chat_id", chatID)
	params.AddNonZero("reply_to_message_id", replyID)
	params.AddNonEmpty("text", text)
	params.AddNonEmpty("parse_mode", parseMode)
	return params
}

func (b *Bot) sendRawMessageWithParseFallback(_ context.Context, params tgbotapi.Params) (tgbotapi.Message, error) {
	resp, err := b.client.UploadFiles(telegramSendMessageEndpoint, params, nil)
	if err != nil && isMarkdownParseError(err) {
		b.log.Warn("telegram MarkdownV2 parse failed, falling back to plain text", "err", err)
		retry := make(tgbotapi.Params, len(params))
		for key, value := range params {
			retry[key] = value
		}
		delete(retry, "parse_mode")
		retry["text"] = stripTelegramMarkdownV2(retry["text"])
		resp, err = b.client.UploadFiles(telegramSendMessageEndpoint, retry, nil)
	}
	if err != nil {
		return tgbotapi.Message{}, err
	}
	return telegramMessageFromAPIResponse(resp)
}

func telegramMessageFromAPIResponse(resp *tgbotapi.APIResponse) (tgbotapi.Message, error) {
	var msg tgbotapi.Message
	if resp != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, &msg); err != nil {
			return tgbotapi.Message{}, err
		}
	}
	return msg, nil
}

func telegramMediaUploadEndpoint(media gateway.OutboundMedia) (endpoint, field string, err error) {
	switch gateway.ClassifyOutboundMedia(media) {
	case gateway.OutboundMediaKindAudio:
		if media.AsVoice {
			return "sendVoice", "voice", nil
		}
		return "sendAudio", "audio", nil
	case gateway.OutboundMediaKindImage:
		return "sendPhoto", "photo", nil
	case gateway.OutboundMediaKindDocument:
		return "sendDocument", "document", nil
	case gateway.OutboundMediaKindVideo:
		return "sendVideo", "video", nil
	default:
		return "", "", fmt.Errorf("telegram: unsupported media type")
	}
}

func (b *Bot) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	return b.Send(ctx, chatID, "⏳")
}

func (b *Bot) SendThreadPlaceholder(ctx context.Context, chatID, threadID string) (string, error) {
	return b.SendThread(ctx, chatID, threadID, "⏳")
}

func (b *Bot) SendReplyPlaceholder(ctx context.Context, chatID, replyToMsgID string) (string, error) {
	return b.SendReply(ctx, chatID, replyToMsgID, "⏳")
}

func (b *Bot) SendThreadReplyPlaceholder(ctx context.Context, chatID, threadID, replyToMsgID string) (string, error) {
	return b.SendThreadReply(ctx, chatID, threadID, replyToMsgID, "⏳")
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
	editCfg := tgbotapi.NewEditMessageText(cid, mid, text)
	editCfg.ParseMode = tgbotapi.ModeMarkdownV2
	if _, err := b.client.Send(editCfg); err != nil {
		if isMarkdownParseError(err) {
			b.log.Warn("telegram MarkdownV2 parse failed on edit, falling back to plain text", "err", err)
			editCfg.ParseMode = ""
			editCfg.Text = stripTelegramMarkdownV2(editCfg.Text)
			if _, retryErr := b.client.Send(editCfg); retryErr != nil {
				return retryErr
			}
			return nil
		}
		return err
	}
	return nil
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

// sendWithParseFallback wraps b.client.Send for MessageConfig payloads with
// the Hermes parity fallback at gateway/platforms/telegram.py:1117-1129. If
// Telegram rejects MarkdownV2 with a parse-entity error, the bot retries
// once with ParseMode unset and the original body unchanged. The render
// layer (internal/gateway/render.go) is the only place that touches escape
// behavior; bot.go must hand the body through byte-identically.
func (b *Bot) sendWithParseFallback(msgCfg tgbotapi.MessageConfig) (tgbotapi.Message, error) {
	msg, err := b.client.Send(msgCfg)
	if err == nil {
		return msg, nil
	}
	if !isMarkdownParseError(err) {
		return tgbotapi.Message{}, err
	}
	b.log.Warn("telegram MarkdownV2 parse failed, falling back to plain text", "err", err)
	msgCfg.ParseMode = ""
	msgCfg.Text = stripTelegramMarkdownV2(msgCfg.Text)
	return b.client.Send(msgCfg)
}

// isMarkdownParseError matches the Hermes heuristic: any send error whose
// message mentions "parse" or "markdown" is treated as a malformed-entity
// rejection, not a transient network failure.
func isMarkdownParseError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "parse") || strings.Contains(lower, "markdown")
}

func isThreadNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "thread not found")
}

func isTimedOutError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "timedout") ||
		strings.Contains(lower, "timed out") ||
		strings.Contains(lower, "i/o timeout")
}

func isTransientNetworkError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "network") ||
		strings.Contains(lower, "connection") ||
		strings.Contains(lower, "eof") ||
		strings.Contains(lower, "reset") ||
		strings.Contains(lower, "broken pipe")
}

// SendChatAction issues a Telegram sendChatAction request. action is one of
// the documented Telegram chat actions ("typing", "upload_photo", etc.).
// Failures are returned to the caller; this method does not retry, log, or
// spawn goroutines. Caller is responsible for redacted evidence on failure.
func (b *Bot) SendChatAction(_ context.Context, chatID, action string) error {
	chatID = strings.TrimSpace(chatID)
	action = strings.TrimSpace(action)
	if chatID == "" {
		return errors.New("telegram: SendChatAction requires non-empty chat_id")
	}
	if action == "" {
		return errors.New("telegram: SendChatAction requires non-empty action")
	}
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("telegram: SendChatAction invalid chat_id %q: %w", chatID, err)
	}
	cfg := tgbotapi.NewChatAction(id, action)
	if _, err := b.client.Request(cfg); err != nil {
		return fmt.Errorf("telegram: SendChatAction: %w", err)
	}
	return nil
}

func (b *Bot) SendThreadChatAction(ctx context.Context, chatID, threadID, action string) error {
	if strings.TrimSpace(threadID) == "" {
		return b.SendChatAction(ctx, chatID, action)
	}
	chatID = strings.TrimSpace(chatID)
	action = strings.TrimSpace(action)
	if chatID == "" {
		return errors.New("telegram: SendThreadChatAction requires non-empty chat_id")
	}
	if action == "" {
		return errors.New("telegram: SendThreadChatAction requires non-empty action")
	}
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("telegram: SendThreadChatAction invalid chat_id %q: %w", chatID, err)
	}
	thread, includeThread, err := telegramThreadIDForAction(threadID)
	if err != nil {
		return err
	}
	params := tgbotapi.Params{}
	params.AddNonZero64("chat_id", id)
	params.AddNonEmpty("action", action)
	if includeThread {
		params.AddNonZero("message_thread_id", thread)
	}

	var lastErr error
	for attempt := 0; attempt < maxSendRetries; attempt++ {
		_, err := b.client.UploadFiles(telegramSendChatActionEndpoint, params, nil)
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("telegram: SendThreadChatAction: %w", err)

		if isThreadNotFoundError(err) && includeThread {
			delete(params, "message_thread_id")
			includeThread = false
			continue
		}
		if isTimedOutError(err) {
			return lastErr
		}
		if !isTransientNetworkError(err) {
			return lastErr
		}
	}
	return lastErr
}

// StartTyping starts Telegram's transient typing indicator and refreshes it no
// more frequently than Telegram's documented five-second display window needs.
// The returned stop function is idempotent and cancels future refreshes.
func (b *Bot) StartTyping(ctx context.Context, chatID string) (func(), error) {
	if err := b.SendChatAction(ctx, chatID, "typing"); err != nil {
		return nil, err
	}
	typingCtx, cancel := context.WithCancel(ctx)
	var once sync.Once
	stop := func() { once.Do(cancel) }
	go func() {
		ticker := time.NewTicker(telegramTypingRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				if err := b.SendChatAction(typingCtx, chatID, "typing"); err != nil {
					b.log.Debug("telegram typing action refresh failed", "err", err)
				}
			}
		}
	}()
	return stop, nil
}

func parseChatID(s string) (int64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("telegram: invalid chat ID %q: %w", s, err)
	}
	return v, nil
}
