// Package gateway is the channel-agnostic messaging chassis for Gormes.
// Individual adapters translate SDK-specific traffic into InboundEvent and
// implement the Channel interface plus any capability sub-interfaces they
// support. The manager owns cross-channel mechanics like command
// normalization, session-map persistence, and outbound routing.
package gateway

import (
	"strconv"
	"strings"
)

// EventKind is the normalized command kind on an inbound message.
type EventKind int

const (
	// EventUnknown is an unrecognized slash command.
	EventUnknown EventKind = iota
	// EventSubmit carries user text for kernel.PlatformEventSubmit.
	EventSubmit
	// EventCancel maps to kernel.PlatformEventCancel.
	EventCancel
	// EventReset maps to kernel.PlatformEventResetSession.
	EventReset
	// EventStart is the help or welcome command.
	EventStart
	// EventRestart requests a graceful service-manager restart.
	EventRestart
	// EventSteer queues operator guidance for the active turn fallback path.
	EventSteer
	// EventQueue queues one full follow-up turn without interrupting the active run.
	EventQueue
	// EventUsage renders runtime and provider account-usage evidence.
	EventUsage
	// EventStatus renders Hermes-style gateway/session status directly in the
	// channel, without submitting the slash text to the model.
	EventStatus
	// EventTitle sets or reads the current session title directly in the channel,
	// without submitting the slash text to the model.
	EventTitle
	// EventVerbose cycles gateway tool-progress display for the calling platform.
	EventVerbose
	EventModel
	// EventGateway renders gateway status.
	EventGateway
	// EventThreadLifecycle carries normalized thread open/close/archive state.
	EventThreadLifecycle
	// EventSessions handles /sessions subcommands (list, show).
	EventSessions
	// EventProfile handles /profile subcommands (show, list).
	EventProfile
	// EventSkills handles /skills subcommands (list, inspect).
	EventSkills
	// EventCommands handles /commands [page] command and skill catalog.
	EventCommands
	// EventReasoning handles /reasoning subcommands (show, set, reset).
	EventReasoning
	// EventBusy handles /busy subcommands (queue, steer, interrupt, status).
	EventBusy
	// EventTTS handles /tts subcommands (on, off, speed, voice, engine, language).
	EventTTS
	// EventReload reloads gateway runtime config without restarting the process.
	EventReload
	// EventReloadSkills refreshes dynamic skill command catalogs without a model turn.
	EventReloadSkills
	// EventRetry handles /retry (retry the last message by resending to agent).
	EventRetry
	// EventUndo handles /undo (remove the last user/assistant exchange).
	EventUndo
	// EventGoal handles /goal state and continuation loop controls.
	EventGoal
	// EventTopic handles Telegram private-chat topic-mode controls.
	EventTopic
	// EventKanban handles /kanban subcommands (create, list, show, complete,
	// claim, block, unblock, link, init). All kanban operations bypass the
	// model and delegate to the local kanban.Store.
	EventKanban
	// EventSpawn handles channel-native dynamic agent spawn UX such as
	// Telegram forum topics and Discord threads.
	EventSpawn
	// EventPlatformControl handles `/platform <list|pause|resume> [name]`,
	// the operator slash handler for the per-platform circuit breaker
	// (distinct from EventGateway's read-only /platforms status display).
	EventPlatformControl
	// EventPersonality handles `/personality` subcommands (list, switch, none).
	EventPersonality
)

// String returns the stable log/test representation of an EventKind.
func (k EventKind) String() string {
	switch k {
	case EventSubmit:
		return "submit"
	case EventCancel:
		return "cancel"
	case EventReset:
		return "reset"
	case EventStart:
		return "start"
	case EventRestart:
		return "restart"
	case EventSteer:
		return "steer"
	case EventQueue:
		return "queue"
	case EventUsage:
		return "usage"
	case EventStatus:
		return "status"
	case EventTitle:
		return "title"
	case EventVerbose:
		return "verbose"
	case EventModel:
		return "model"
	case EventGateway:
		return "gateway"
	case EventThreadLifecycle:
		return "thread_lifecycle"
	case EventSessions:
		return "sessions"
	case EventProfile:
		return "profile"
	case EventSkills:
		return "skills"
	case EventCommands:
		return "commands"
	case EventReasoning:
		return "reasoning"
	case EventBusy:
		return "busy"
	case EventTTS:
		return "tts"
	case EventReload:
		return "reload"
	case EventReloadSkills:
		return "reload_skills"
	case EventRetry:
		return "retry"
	case EventUndo:
		return "undo"
	case EventGoal:
		return "goal"
	case EventTopic:
		return "topic"
	case EventKanban:
		return "kanban"
	case EventSpawn:
		return "spawn"
	case EventPlatformControl:
		return "platform_control"
	case EventPersonality:
		return "personality"
	default:
		return "unknown"
	}
}

// ThreadLifecycleState is the platform-neutral lifecycle state for a threaded
// conversation surface such as a Discord thread or forum post.
type ThreadLifecycleState string

const (
	ThreadLifecycleOpen     ThreadLifecycleState = "open"
	ThreadLifecycleClosed   ThreadLifecycleState = "closed"
	ThreadLifecycleArchived ThreadLifecycleState = "archived"
)

// ThreadLifecycleEvent carries normalized thread metadata alongside the
// channel-neutral InboundEvent envelope.
type ThreadLifecycleEvent struct {
	ID       string
	ParentID string
	Name     string
	State    ThreadLifecycleState
	Archived bool
	Locked   bool
}

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

// ChatKey returns the internal/session map key shape for this event.
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
	text := strings.TrimSpace(e.Text)
	if reply := strings.TrimSpace(e.ReplyToText); reply != "" {
		prefix := `[Replying to: "` + truncateRunes(reply, 500) + `"]`
		if text == "" {
			text = prefix
		} else {
			text = prefix + "\n\n" + text
		}
	}
	if len(e.Attachments) == 0 {
		return text
	}

	attachmentLines := make([]string, 0, len(e.Attachments))
	for _, att := range e.Attachments {
		if line := att.submitLine(); line != "" {
			attachmentLines = append(attachmentLines, line)
		}
	}
	if len(attachmentLines) == 0 {
		return text
	}
	audioHintLines := audioTranscriptionHintLines(e.Attachments)

	lines := make([]string, 0, len(attachmentLines)+len(audioHintLines)+5)
	if text != "" {
		lines = append(lines, text, "")
	}
	lines = append(lines, "Attachments:")
	lines = append(lines, attachmentLines...)
	if len(audioHintLines) > 0 {
		lines = append(lines, "", "Audio transcription:")
		lines = append(lines, audioHintLines...)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

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

// Attachment is the channel-neutral media descriptor attached to an inbound
// event. SourceID preserves the platform-side media identifier so failures can
// still be diagnosed even when URL resolution fails.
type Attachment struct {
	Kind      string `json:"kind"`
	URL       string `json:"url,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
	FileName  string `json:"fileName,omitempty"`
	SourceID  string `json:"sourceId,omitempty"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (a Attachment) submitLine() string {
	kind := strings.TrimSpace(a.Kind)
	if kind == "" {
		kind = "attachment"
	}
	label := kind
	if fileName := strings.TrimSpace(a.FileName); fileName != "" {
		label += " " + fileName
	}

	target := strings.TrimSpace(a.URL)
	if target == "" {
		target = strings.TrimSpace(a.SourceID)
	}
	if target == "" {
		return ""
	}

	var meta []string
	if mediaType := strings.TrimSpace(a.MediaType); mediaType != "" {
		meta = append(meta, "mediaType="+mediaType)
	}
	if sourceID := strings.TrimSpace(a.SourceID); sourceID != "" {
		meta = append(meta, "sourceId="+sourceID)
	}
	if a.SizeBytes > 0 {
		meta = append(meta, "sizeBytes="+strconv.FormatInt(a.SizeBytes, 10))
	}
	if errText := strings.TrimSpace(a.Error); errText != "" {
		meta = append(meta, "error="+errText)
	}

	line := "- " + label + ": " + target
	if len(meta) > 0 {
		line += " (" + strings.Join(meta, ", ") + ")"
	}
	return line
}

func audioTranscriptionHintLines(attachments []Attachment) []string {
	lines := make([]string, 0, len(attachments))
	for _, att := range attachments {
		if line := att.audioTranscriptionHintLine(); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func (a Attachment) audioTranscriptionHintLine() string {
	if !a.isAudioAttachment() {
		return ""
	}
	audioPath := strings.TrimSpace(a.URL)
	if audioPath == "" || !isLocalAttachmentPath(audioPath) {
		return ""
	}

	var meta []string
	if kind := strings.TrimSpace(a.Kind); kind != "" {
		meta = append(meta, "kind="+kind)
	}
	if fileName := strings.TrimSpace(a.FileName); fileName != "" {
		meta = append(meta, "fileName="+fileName)
	}
	if mediaType := strings.TrimSpace(a.MediaType); mediaType != "" {
		meta = append(meta, "mediaType="+mediaType)
	}

	line := "- transcribe_audio audio_path=" + strconv.Quote(audioPath)
	if len(meta) > 0 {
		line += " (" + strings.Join(meta, ", ") + ")"
	}
	return line
}

func (a Attachment) isAudioAttachment() bool {
	switch strings.ToLower(strings.TrimSpace(a.Kind)) {
	case "audio", "voice", "voice_message", "voice_note":
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.MediaType)), "audio/")
}

func isLocalAttachmentPath(target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	lower := strings.ToLower(target)
	if strings.Contains(lower, "://") || strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "transcript:") {
		return false
	}
	if strings.HasPrefix(target, "/") || strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../") || strings.HasPrefix(target, "~") {
		return true
	}
	if len(target) >= 3 && target[1] == ':' && (target[2] == '\\' || target[2] == '/') {
		return true
	}
	return strings.ContainsAny(target, `/\`)
}
