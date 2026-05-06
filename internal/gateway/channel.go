package gateway

import "context"

// Channel is the minimum every adapter implements. Additional capabilities are
// modeled as optional interfaces that the manager type-asserts at runtime.
type Channel interface {
	// Name returns the stable platform identifier ("telegram", "discord", ...).
	Name() string

	// Run starts the inbound loop and blocks until ctx cancellation. The
	// adapter must not close inbox; the manager owns the shared channel.
	Run(ctx context.Context, inbox chan<- InboundEvent) error

	// Send delivers a plain-text message to chatID and returns the platform
	// message ID when one exists.
	Send(ctx context.Context, chatID, text string) (msgID string, err error)
}

// ReplySender is implemented by channels that can send a message as a native
// reply/quote to an inbound platform message.
type ReplySender interface {
	SendReply(ctx context.Context, chatID, replyToMsgID, text string) (msgID string, err error)
}

// OutboundMediaKind classifies local files for native platform send paths.
type OutboundMediaKind string

const (
	OutboundMediaKindAudio    OutboundMediaKind = "audio"
	OutboundMediaKindDocument OutboundMediaKind = "document"
	OutboundMediaKindImage    OutboundMediaKind = "image"
	OutboundMediaKindVideo    OutboundMediaKind = "video"
)

// OutboundMedia is a local file that should be delivered through a platform's
// native media path instead of being shown as a raw MEDIA tag in assistant text.
type OutboundMedia struct {
	Path     string
	AsVoice  bool
	Kind     OutboundMediaKind
	ThreadID string
}

// MediaSender is implemented by channels that can send local files as native
// platform media. replyToMsgID is optional and preserves reply quoting when the
// channel supports it.
type MediaSender interface {
	SendMedia(ctx context.Context, chatID, replyToMsgID string, media OutboundMedia) (msgID string, err error)
}

// DisconnectCapable is implemented by channels that can release resources
// outside their Run loop after a failed startup.
type DisconnectCapable interface {
	Disconnect(ctx context.Context) error
}

// MessageEditor is implemented by channels that can edit an existing message.
type MessageEditor interface {
	EditMessage(ctx context.Context, chatID, msgID, text string) error
}

// FinalizingMessageEditor is implemented by channels whose streaming edit
// lifecycle needs an explicit terminal update.
type FinalizingMessageEditor interface {
	EditMessageFinal(ctx context.Context, chatID, msgID, text string, finalize bool) error
}

// MessageDeleter is implemented by channels that can remove an existing
// message after a replacement has been delivered.
type MessageDeleter interface {
	DeleteMessage(ctx context.Context, chatID, msgID string) error
}

// PlaceholderCapable is implemented by channels that can create a message
// placeholder for subsequent streaming edits.
type PlaceholderCapable interface {
	SendPlaceholder(ctx context.Context, chatID string) (msgID string, err error)
}

// ReplyPlaceholderCapable is implemented by editable channels that can create
// their initial streaming placeholder as a native reply to the inbound message.
type ReplyPlaceholderCapable interface {
	SendReplyPlaceholder(ctx context.Context, chatID, replyToMsgID string) (msgID string, err error)
}

// TypingCapable is implemented by channels that can show a typing indicator.
// The returned stop function must be idempotent.
type TypingCapable interface {
	StartTyping(ctx context.Context, chatID string) (stop func(), err error)
}

// TypingActionCapable is implemented by channels that expose one-shot typing
// actions such as Telegram sendChatAction.
type TypingActionCapable interface {
	SendChatAction(ctx context.Context, chatID, action string) error
}

// ProcessingOutcome is the channel-neutral terminal state for best-effort
// processing reactions.
type ProcessingOutcome string

const (
	ProcessingOutcomeSuccess   ProcessingOutcome = "success"
	ProcessingOutcomeFailure   ProcessingOutcome = "failure"
	ProcessingOutcomeCancelled ProcessingOutcome = "cancelled"
)

// ReactionCapable is implemented by channels that can react to inbound
// message processing lifecycle events. Implementations must treat missing IDs,
// disabled reaction config, and platform API failures as non-fatal.
type ReactionCapable interface {
	OnProcessingStart(ctx context.Context, chatID, msgID string) error
	OnProcessingComplete(ctx context.Context, chatID, msgID string, outcome ProcessingOutcome) error
}
