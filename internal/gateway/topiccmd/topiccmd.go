package topiccmd

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

// Capabilities mirrors the bounded BotFather topic settings checked before
// enabling Telegram private-chat topic mode.
type Capabilities struct {
	Checked                   bool
	HasTopicsEnabled          bool
	AllowsUsersToCreateTopics bool
}

// ModeRecord is the durable topic-mode row needed to enable Telegram private-chat topics.
type ModeRecord struct {
	ChatID                    string
	UserID                    string
	HasTopicsEnabled          bool
	AllowsUsersToCreateTopics bool
	CapabilityChecked         bool
}

// PrivateChat reports whether /topic can operate on this Telegram conversation.
func PrivateChat(platform string, isDirectMessage bool, chatType, chatID string) bool {
	if !isTelegramPlatform(platform) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(chatType)) {
	case "group", "supergroup", "channel":
		return false
	case "private", "private_chat", "dm", "direct":
		return true
	case "":
		if strings.HasPrefix(strings.TrimSpace(chatID), "-") {
			return false
		}
		if isDirectMessage {
			return true
		}
	default:
		return false
	}
	// Older adapter fixtures may not carry ChatType. Telegram private chat IDs
	// are positive, while group/supergroup/channel IDs are negative.
	return !strings.HasPrefix(strings.TrimSpace(chatID), "-")
}

// HelpText renders operator guidance for Telegram topic mode.
func HelpText() string {
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

// CapabilityGuidance renders BotFather setup instructions for a failed capability check.
func CapabilityGuidance(reason string) string {
	return fmt.Sprintf(`%s

How to enable them:
1. Open @BotFather.
2. Choose your bot.
3. Open Bot Settings > Threads Settings.
4. Turn on Threaded Mode and make sure users are allowed to create new threads.

Then send /topic again.`, guidanceLine(reason))
}

func guidanceLine(value string) string {
	value = collapseRedactedReasonAssignments(redaction.RedactSecrets(value))
	value = strings.NewReplacer("`", "'", "*", "'").Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func collapseRedactedReasonAssignments(value string) string {
	replacer := strings.NewReplacer(
		"api_key=[redacted]", "[redacted]",
		"api-key=[redacted]", "[redacted]",
		"authorization=[redacted]", "[redacted]",
		"bearer=[redacted]", "[redacted]",
		"token=[redacted]", "[redacted]",
		"secret=[redacted]", "[redacted]",
		"password=[redacted]", "[redacted]",
	)
	fields := strings.Fields(replacer.Replace(value))
	out := make([]string, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		lower := strings.ToLower(field)
		nextRedacted := i+1 < len(fields) && strings.Contains(strings.ToLower(fields[i+1]), "[redacted]")
		if topicSecretField(lower) && (strings.Contains(lower, "[redacted]") || nextRedacted) {
			out = append(out, "[redacted]")
			if nextRedacted {
				i++
			}
			continue
		}
		out = append(out, field)
	}
	return strings.Join(out, " ")
}

func topicSecretField(value string) bool {
	return strings.Contains(value, "api_key") || strings.Contains(value, "api-key") || strings.Contains(value, "apikey") || strings.Contains(value, "authorization") || strings.Contains(value, "bearer") || strings.Contains(value, "token") || strings.Contains(value, "secret") || strings.Contains(value, "password")
}

// CapabilityDebouncedText is the bounded evidence reply when setup guidance was recently sent.
func CapabilityDebouncedText() string {
	return "telegram_topic_capability_hint_debounced: Topic setup guidance was sent recently. Try again in a few minutes."
}

// RestoreGuidance renders the current restore guidance for /topic <session-id>.
func RestoreGuidance(threadID string) string {
	if strings.TrimSpace(threadID) == "" {
		return "To restore a session, first create or open a Telegram topic, then send /topic <session-id> inside that topic."
	}
	return "telegram_topic_unavailable: Session restore into a Telegram topic is not available in this build yet."
}

func isTelegramPlatform(platform string) bool {
	base := strings.ToLower(strings.TrimSpace(platform))
	if base == "" {
		return false
	}
	for _, sep := range []string{":", "/", ".", "#"} {
		if i := strings.Index(base, sep); i >= 0 {
			base = base[:i]
		}
	}
	return base == "telegram"
}
