// Package gateway is the channel-agnostic messaging chassis for Gormes.
// Individual adapters translate SDK-specific traffic into InboundEvent and
// implement the Channel interface plus any capability sub-interfaces they
// support. The manager owns cross-channel mechanics like command
// normalization, session-map persistence, and outbound routing.
package gateway

import (
	"strings"

	gatewayevents "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"
)

type EventKind = gatewayevents.EventKind

const (
	EventUnknown         = gatewayevents.EventUnknown
	EventSubmit          = gatewayevents.EventSubmit
	EventCancel          = gatewayevents.EventCancel
	EventReset           = gatewayevents.EventReset
	EventStart           = gatewayevents.EventStart
	EventRestart         = gatewayevents.EventRestart
	EventSteer           = gatewayevents.EventSteer
	EventQueue           = gatewayevents.EventQueue
	EventUsage           = gatewayevents.EventUsage
	EventStatus          = gatewayevents.EventStatus
	EventTitle           = gatewayevents.EventTitle
	EventVerbose         = gatewayevents.EventVerbose
	EventModel           = gatewayevents.EventModel
	EventGateway         = gatewayevents.EventGateway
	EventThreadLifecycle = gatewayevents.EventThreadLifecycle
	EventSessions        = gatewayevents.EventSessions
	EventProfile         = gatewayevents.EventProfile
	EventSkills          = gatewayevents.EventSkills
	EventCommands        = gatewayevents.EventCommands
	EventReasoning       = gatewayevents.EventReasoning
	EventBusy            = gatewayevents.EventBusy
	EventTTS             = gatewayevents.EventTTS
	EventReload          = gatewayevents.EventReload
	EventReloadSkills    = gatewayevents.EventReloadSkills
	EventRetry           = gatewayevents.EventRetry
	EventUndo            = gatewayevents.EventUndo
	EventGoal            = gatewayevents.EventGoal
	EventTopic           = gatewayevents.EventTopic
	EventKanban          = gatewayevents.EventKanban
	EventSpawn           = gatewayevents.EventSpawn
	EventPlatformControl = gatewayevents.EventPlatformControl
	EventPersonality     = gatewayevents.EventPersonality
)

// ThreadLifecycleState is the platform-neutral lifecycle state for a threaded
// conversation surface such as a Discord thread or forum post.
type ThreadLifecycleState = gatewayevents.ThreadLifecycleState

const (
	ThreadLifecycleOpen     = gatewayevents.ThreadLifecycleOpen
	ThreadLifecycleClosed   = gatewayevents.ThreadLifecycleClosed
	ThreadLifecycleArchived = gatewayevents.ThreadLifecycleArchived
)

// ThreadLifecycleEvent carries normalized thread metadata alongside the
// channel-neutral InboundEvent envelope.
type ThreadLifecycleEvent = gatewayevents.ThreadLifecycleEvent

// InboundEvent is the platform-neutral form every channel emits into the
// shared gateway manager.
type InboundEvent struct {
	Platform  string
	AccountID string
	ChatID    string
	ChatName  string
	ChatType  string
	UserID    string
	UserName  string
	ThreadID  string
	MsgID     string
	GuildID   string
	TeamID    string
	Roles     []string
	// ParentChatID preserves the containing channel/forum when ChatID and
	// ThreadID identify a threaded conversation surface.
	ParentChatID string
	// MessageID is source-context metadata for the triggering platform message.
	// MsgID remains the gateway's existing hook/reaction field.
	MessageID string
	// ReplyToText carries optional text from the platform message this event is
	// replying to. Adapters may leave it empty when parent lookup degrades.
	ReplyToText string
	Kind        EventKind
	Text        string
	// AutoSkills carries channel-scoped skills resolved by adapters from
	// Hermes-compatible channel_skill_bindings.
	AutoSkills []string
	// SkillSlashExpanded marks submit text that was already expanded from a
	// Hermes-compatible /skill-name invocation. The manager uses it to avoid
	// injecting a second automatic skill block for the same turn.
	SkillSlashExpanded bool
	// ChannelPrompt carries an ephemeral per-channel prompt resolved by the
	// adapter. It is injected for this turn only and never mutates global
	// prompt or skill configuration.
	ChannelPrompt string
	// AllowlistBypassReason is set only by adapters for source-backed,
	// policy-scoped admission exceptions that should pass the manager
	// allowlist while remaining auditable.
	AllowlistBypassReason string

	ThreadLifecycle *ThreadLifecycleEvent

	// Attachments carries platform-normalized inbound media references. The
	// shared gateway keeps this as metadata; adapters own platform-specific
	// download and fallback behavior.
	Attachments []Attachment
}

const AllowlistBypassTelegramGuestMention = "telegram_guest_mention"

// ChatKey returns the internal/persistence/session map key shape for this event.
func (e InboundEvent) ChatKey() string {
	return e.Platform + ":" + e.ChatID
}

// IsDirectMessage reports whether the adapter identified the source as a
// one-to-one chat. Telegram uses "private" while other adapters generally use
// "dm" or "direct".
func (e InboundEvent) IsDirectMessage() bool {
	switch strings.ToLower(strings.TrimSpace(e.ChatType)) {
	case "dm", "direct", "private", "private_chat":
		return true
	default:
		return false
	}
}

// PairingUserID returns the user identity eligible for pairing policy. Telegram
// private messages may omit from_user for service-like events; upstream Hermes
// falls back to chat.id only for private chats and never for groups/channels.
func (e InboundEvent) PairingUserID() string {
	if userID := strings.TrimSpace(e.UserID); userID != "" {
		return userID
	}
	if isTelegramPlatform(e.Platform) && e.IsDirectMessage() {
		return strings.TrimSpace(e.ChatID)
	}
	return ""
}

// SubmitText returns the text sent to the kernel for submit events, including
// deterministic attachment references when a channel supplied inbound media.
func (e InboundEvent) SubmitText() string {
	return gatewayevents.SubmitText(e.Text, e.ReplyToText, e.Attachments)
}

// Attachment is the channel-neutral media descriptor attached to an inbound
// event. SourceID preserves the platform-side media identifier so failures can
// still be diagnosed even when URL resolution fails.
type Attachment = gatewayevents.Attachment

func truncateRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	for i := range s {
		if limit == 0 {
			return s[:i]
		}
		limit--
	}
	return s
}
