package gateway

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

const maxMessageLen = 4000

// FormatStreamPlain renders a streaming frame as plain text.
func FormatStreamPlain(f kernel.RenderFrame) string {
	body := f.DraftText
	tail := ""
	if len(f.SoulEvents) > 0 {
		last := f.SoulEvents[len(f.SoulEvents)-1]
		if last.Text != "" && last.Text != "idle" {
			tail = "\n\n" + formatToolTracePlain(last.Text)
		}
	}
	if f.Phase == kernel.PhaseReconnecting {
		tail += "\n\nreconnecting…"
	}
	return truncate(body + tail)
}

// FormatFinalPlain returns the final assistant text from render history.
func FormatFinalPlain(f kernel.RenderFrame) string {
	return FormatFinalPlainText(FinalAssistantText(f))
}

// FinalAssistantText returns the raw final assistant text from render history.
func FinalAssistantText(f kernel.RenderFrame) string {
	for i := len(f.History) - 1; i >= 0; i-- {
		if f.History[i].Role == "assistant" {
			return f.History[i].Content
		}
	}
	return ""
}

func FormatFinalPlainText(text string) string {
	if strings.TrimSpace(text) == "" {
		return "(empty reply)"
	}
	return truncate(text)
}

// FormatErrorPlain renders a terminal error frame.
func FormatErrorPlain(f kernel.RenderFrame) string {
	text := "❌ " + sanitizeProviderErrorText(f.LastError)
	if f.LastError == "" {
		text = "❌ cancelled"
	}
	return truncate(text)
}

// FormatStreamTelegram renders a streaming frame using Telegram MarkdownV2.
func FormatStreamTelegram(f kernel.RenderFrame) string {
	body := tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, f.DraftText)
	body = truncate(body)
	tail := ""
	if len(f.SoulEvents) > 0 {
		last := f.SoulEvents[len(f.SoulEvents)-1]
		if last.Text != "" && last.Text != "idle" {
			tail = "\n\n_" + tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, formatToolTracePlain(last.Text)) + "_"
		}
	}
	if f.Phase == kernel.PhaseReconnecting {
		tail += "\n\n_reconnecting…_"
	}
	return body + tail
}

// FormatFinalTelegram renders the final assistant message for Telegram.
func FormatFinalTelegram(f kernel.RenderFrame) string {
	return FormatFinalTelegramText(FinalAssistantText(f))
}

func FormatFinalTelegramText(text string) string {
	if strings.TrimSpace(text) == "" {
		return "_\\(empty reply\\)_"
	}
	return truncate(tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, text))
}

// FormatErrorTelegram renders an error frame for Telegram MarkdownV2.
func FormatErrorTelegram(f kernel.RenderFrame) string {
	text := "❌ " + sanitizeProviderErrorText(f.LastError)
	if f.LastError == "" {
		text = "❌ cancelled"
	}
	return truncate(tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, text))
}

func formatToolTracePlain(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if strings.HasPrefix(text, "tool: ") {
		payload := strings.TrimSpace(strings.TrimPrefix(text, "tool: "))
		name, arg, ok := strings.Cut(payload, ":")
		if ok {
			name = strings.TrimSpace(name)
			arg = strings.TrimSpace(arg)
			return toolTraceIcon(name) + " " + name + ": " + quoteAndTruncate(arg, 96)
		}
	}
	return "🔧 " + text
}

func toolTraceIcon(name string) string {
	switch strings.TrimSpace(name) {
	case "memory", "honcho_context", "honcho_search", "honcho_profile", "honcho_conclude", "session_search":
		return "🧠"
	case "search_files", "browser_navigate", "browser_snapshot", "web_search":
		return "🔎"
	case "read_file":
		return "📖"
	case "patch", "write_file":
		return "🔧"
	case "terminal", "execute_code", "process":
		return "🖥"
	default:
		return "🔧"
	}
}

func quoteAndTruncate(s string, limit int) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if limit > 0 {
		runes := []rune(s)
		if len(runes) > limit {
			s = string(runes[:limit-1]) + "…"
		}
	}
	return `"` + s + `"`
}

func sanitizeProviderErrorText(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html") || strings.Contains(lower, "<svg") {
		if idx := strings.Index(trimmed, ":"); idx > 0 {
			prefix := strings.TrimSpace(trimmed[:idx])
			if prefix != "" && !strings.ContainsAny(prefix, "<>\n\r") {
				return prefix + ": provider returned HTML error body"
			}
		}
		return "provider returned HTML error body"
	}
	return trimmed
}

func truncate(s string) string {
	runes := []rune(s)
	if len(runes) <= maxMessageLen {
		return s
	}
	return string(runes[:maxMessageLen-1]) + "…"
}
