package gateway

import (
	"context"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/topiccmd"
)

const telegramTopicCapabilityHintCooldown = 5 * time.Minute

// TelegramTopicCapabilitiesFunc checks Telegram private-chat topic settings
// without requiring the command handler to know Bot API details.
type TelegramTopicCapabilitiesFunc func(context.Context, InboundEvent) (TelegramTopicCapabilities, error)

// TelegramTopicCapabilities mirrors the bounded BotFather topic settings that
// Hermes checks before enabling Telegram private-chat topic mode.
type TelegramTopicCapabilities = topiccmd.Capabilities

// TelegramTopicModeRecord is the durable topic-mode row the gateway needs to
// enable for a Telegram private chat.
type TelegramTopicModeRecord = topiccmd.ModeRecord

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
	return topiccmd.PrivateChat(ev.Platform, ev.IsDirectMessage(), ev.ChatType, ev.ChatID)
}

func (m *Manager) telegramTopicAuthorized(ev InboundEvent) bool {
	if users := m.cfg.AllowedUsers["telegram"]; len(users) > 0 {
		return users[strings.TrimSpace(ev.UserID)]
	}
	return m.allowed(ev)
}

func telegramTopicHelpText() string {
	return topiccmd.HelpText()
}

func (m *Manager) disableTelegramTopicMode(ctx context.Context, store TelegramTopicStore, ev InboundEvent) string {
	enabled, err := store.IsTelegramTopicModeEnabled(ctx, ev.ChatID, ev.UserID)
	if err != nil {
		return "telegram_topic_unavailable: " + telegramTopicErrorText(err)
	}
	if !enabled {
		return "Multi-session topic mode is not currently enabled for this chat."
	}
	if err := store.DisableTelegramTopicMode(ctx, ev.ChatID); err != nil {
		return "telegram_topic_unavailable: failed to disable topic mode: " + telegramTopicErrorText(err)
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
			return "telegram_topic_unavailable: capability check failed: " + telegramTopicErrorText(err)
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
		return "telegram_topic_unavailable: failed to enable topic mode: " + telegramTopicErrorText(err)
	}
	if strings.TrimSpace(ev.ThreadID) != "" {
		return "Telegram multi-session topics are enabled. This topic will be used as an independent Gormes session."
	}
	return "Telegram multi-session topics are enabled.\n\nTo create a new Gormes chat, open All Messages at the top of this bot interface and send any message there."
}

func telegramTopicErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	compact := compactTelegramTopicSecretSeparators(lower)
	for _, marker := range []string{"token", "api_key", "apikey", "authorization", "bearer", "secret", "password"} {
		if strings.Contains(lower, marker) || strings.Contains(compact, marker) {
			return "[redacted]"
		}
	}
	replacer := strings.NewReplacer("`", "'", "*", "'", "#", "＃")
	return strings.Join(strings.Fields(replacer.Replace(msg)), " ")
}

func compactTelegramTopicSecretSeparators(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (m *Manager) telegramTopicCapabilityGuidance(chatID, reason string) string {
	if !m.shouldSendTelegramTopicCapabilityHint(chatID) {
		return topiccmd.CapabilityDebouncedText()
	}
	return topiccmd.CapabilityGuidance(reason)
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
	return topiccmd.RestoreGuidance(ev.ThreadID)
}
