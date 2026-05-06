package gateway

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const telegramTopicCapabilityHintCooldown = 5 * time.Minute

// TelegramTopicCapabilitiesFunc checks Telegram private-chat topic settings
// without requiring the command handler to know Bot API details.
type TelegramTopicCapabilitiesFunc func(context.Context, InboundEvent) (TelegramTopicCapabilities, error)

// TelegramTopicCapabilities mirrors the bounded BotFather topic settings that
// Hermes checks before enabling Telegram private-chat topic mode.
type TelegramTopicCapabilities struct {
	Checked                   bool
	HasTopicsEnabled          bool
	AllowsUsersToCreateTopics bool
}

// TelegramTopicModeRecord is the durable topic-mode row the gateway needs to
// enable for a Telegram private chat.
type TelegramTopicModeRecord struct {
	ChatID                    string
	UserID                    string
	HasTopicsEnabled          bool
	AllowsUsersToCreateTopics bool
	CapabilityChecked         bool
}

// TelegramTopicStore is intentionally small: this slice owns only topic
// command state toggles, not broader topic/session routing.
type TelegramTopicStore interface {
	IsTelegramTopicModeEnabled(ctx context.Context, chatID, userID string) (bool, error)
	EnableTelegramTopicMode(ctx context.Context, record TelegramTopicModeRecord) error
	DisableTelegramTopicMode(ctx context.Context, chatID string) error
}

func (m *Manager) handleTelegramTopicCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, m.telegramTopicCommandReply(ctx, ev))
}

func (m *Manager) telegramTopicCommandReply(ctx context.Context, ev InboundEvent) string {
	if !telegramTopicPrivateChat(ev) {
		return "The /topic command is only available in Telegram private chats."
	}

	args := commandArgs(ev.Text)
	subcommand := ""
	if len(args) > 0 {
		subcommand = strings.ToLower(strings.TrimSpace(args[0]))
	}

	if subcommand == "help" || subcommand == "?" || subcommand == "-h" || subcommand == "--help" {
		return telegramTopicHelpText()
	}

	if !m.telegramTopicAuthorized(ev) {
		return "telegram_topic_auth_denied: You are not authorized to use /topic on this bot."
	}

	store := m.cfg.TelegramTopicStore
	if store == nil {
		return "telegram_topic_unavailable: Session database not available."
	}

	switch subcommand {
	case "off", "disable", "stop":
		return m.disableTelegramTopicMode(ctx, store, ev)
	case "":
		return m.enableTelegramTopicMode(ctx, store, ev)
	default:
		return m.telegramTopicRestoreGuidance(ev)
	}
}

func telegramTopicPrivateChat(ev InboundEvent) bool {
	if !strings.EqualFold(strings.TrimSpace(ev.Platform), "telegram") {
		return false
	}
	if ev.IsDirectMessage() {
		return true
	}
	chatType := strings.TrimSpace(ev.ChatType)
	if chatType != "" {
		return false
	}
	// Older adapter fixtures may not carry ChatType. Telegram private chat IDs
	// are positive, while group/supergroup/channel IDs are negative.
	return !strings.HasPrefix(strings.TrimSpace(ev.ChatID), "-")
}

func (m *Manager) telegramTopicAuthorized(ev InboundEvent) bool {
	if users := m.cfg.AllowedUsers["telegram"]; len(users) > 0 {
		return users[strings.TrimSpace(ev.UserID)]
	}
	return m.allowed(ev)
}

func telegramTopicHelpText() string {
	return strings.Join([]string{
		"/topic - enable multi-session DM mode (one bot, many parallel chats)",
		"",
		"Usage:",
		"  /topic             Enable topic mode, or show status if already on",
		"  /topic help        Show this message",
		"  /topic off         Disable topic mode and clear topic bindings",
		"  /topic <id>        Inside a topic: restore a previous session by ID",
		"",
		"How it works:",
		"1. Run /topic once in this DM. Gormes checks BotFather Threads Settings and flips on multi-session mode.",
		"2. Tap All Messages at the top of the bot and send any message. Telegram creates a new topic for that message.",
		"3. The root DM becomes a system lobby. Send /topic, /status, /help, and /usage there.",
		"4. /new inside a topic resets just that topic's session.",
		"5. /topic <id> inside a topic restores an old session into it.",
	}, "\n")
}

func (m *Manager) disableTelegramTopicMode(ctx context.Context, store TelegramTopicStore, ev InboundEvent) string {
	enabled, err := store.IsTelegramTopicModeEnabled(ctx, ev.ChatID, ev.UserID)
	if err != nil {
		return "telegram_topic_unavailable: " + err.Error()
	}
	if !enabled {
		return "Multi-session topic mode is not currently enabled for this chat."
	}
	if err := store.DisableTelegramTopicMode(ctx, ev.ChatID); err != nil {
		return "telegram_topic_unavailable: failed to disable topic mode: " + err.Error()
	}
	m.resetTelegramTopicDebounce(ev.ChatID)
	return strings.Join([]string{
		"Multi-session topic mode is now OFF for this chat.",
		"",
		"Existing topics in Telegram are not removed. They will stop being gated as independent sessions, and the root DM works as a normal Gormes chat again. Run /topic to re-enable later.",
	}, "\n")
}

func (m *Manager) enableTelegramTopicMode(ctx context.Context, store TelegramTopicStore, ev InboundEvent) string {
	capabilities := TelegramTopicCapabilities{}
	if m.cfg.TelegramTopicCapabilities != nil {
		var err error
		capabilities, err = m.cfg.TelegramTopicCapabilities(ctx, ev)
		if err != nil {
			return "telegram_topic_unavailable: capability check failed: " + err.Error()
		}
	}
	if capabilities.Checked {
		if !capabilities.HasTopicsEnabled {
			return m.telegramTopicCapabilityGuidance(ev.ChatID, "Telegram topics are not enabled for this bot yet.")
		}
		if !capabilities.AllowsUsersToCreateTopics {
			return m.telegramTopicCapabilityGuidance(ev.ChatID, "Telegram topics are enabled, but users are not allowed to create topics.")
		}
	}

	record := TelegramTopicModeRecord{
		ChatID:                    ev.ChatID,
		UserID:                    ev.UserID,
		HasTopicsEnabled:          capabilities.HasTopicsEnabled,
		AllowsUsersToCreateTopics: capabilities.AllowsUsersToCreateTopics,
		CapabilityChecked:         capabilities.Checked,
	}
	if err := store.EnableTelegramTopicMode(ctx, record); err != nil {
		return "telegram_topic_unavailable: failed to enable topic mode: " + err.Error()
	}
	if strings.TrimSpace(ev.ThreadID) != "" {
		return "Telegram multi-session topics are enabled. This topic will be used as an independent Gormes session."
	}
	return "Telegram multi-session topics are enabled.\n\nTo create a new Gormes chat, open All Messages at the top of this bot interface and send any message there."
}

func (m *Manager) telegramTopicCapabilityGuidance(chatID, reason string) string {
	if !m.shouldSendTelegramTopicCapabilityHint(chatID) {
		return "telegram_topic_capability_hint_debounced: Topic setup guidance was sent recently. Try again in a few minutes."
	}
	return fmt.Sprintf(`%s

How to enable them:
1. Open @BotFather.
2. Choose your bot.
3. Open Bot Settings > Threads Settings.
4. Turn on Threaded Mode and make sure users are allowed to create new threads.

Then send /topic again.`, reason)
}

func (m *Manager) shouldSendTelegramTopicCapabilityHint(chatID string) bool {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return true
	}
	now := m.now()
	m.telegramTopicMu.Lock()
	defer m.telegramTopicMu.Unlock()
	last, ok := m.telegramTopicCapabilityHint[chatID]
	if ok && now.Sub(last) < telegramTopicCapabilityHintCooldown {
		return false
	}
	m.telegramTopicCapabilityHint[chatID] = now
	return true
}

func (m *Manager) resetTelegramTopicDebounce(chatID string) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return
	}
	m.telegramTopicMu.Lock()
	defer m.telegramTopicMu.Unlock()
	delete(m.telegramTopicCapabilityHint, chatID)
}

func (m *Manager) telegramTopicRestoreGuidance(ev InboundEvent) string {
	if strings.TrimSpace(ev.ThreadID) == "" {
		return "To restore a session, first create or open a Telegram topic, then send /topic <session-id> inside that topic."
	}
	return "telegram_topic_unavailable: Session restore into a Telegram topic is not available in this build yet."
}
