package gateway

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
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

// FormatBrowserArtifactTelegram renders a browser result envelope as a bounded
// Telegram MarkdownV2 artifact notice. It keeps browser/tool progress separate
// from final assistant text and never exposes local screenshot paths or raw
// persisted artifact bytes.
func FormatBrowserArtifactTelegram(envelope tools.BrowserResultEnvelope) string {
	var lines []string
	lines = append(lines, "🌐 *Browser artifact*")
	if title := strings.TrimSpace(envelope.State.Title); title != "" {
		lines = append(lines, "Title: "+escapeTelegramMarkdown(title))
	}
	if u := strings.TrimSpace(envelope.State.URL); u != "" {
		lines = append(lines, "URL: "+escapeTelegramMarkdown(u))
	}
	if artifact := strings.TrimSpace(envelope.Tool.Artifact); artifact != "" {
		detail := artifact
		if envelope.Tool.Bytes > 0 {
			detail = fmt.Sprintf("%s (%d bytes)", artifact, envelope.Tool.Bytes)
		}
		lines = append(lines, "Artifact: "+escapeTelegramMarkdown(detail))
	}
	if strings.TrimSpace(envelope.State.ScreenshotPath) != "" {
		lines = append(lines, "Screenshot: browser artifact available")
	}
	if console := joinBrowserArtifactLines(envelope.State.Console, 2); console != "" {
		lines = append(lines, "Console: "+escapeTelegramMarkdown(console))
	}
	if errs := joinBrowserArtifactLines(envelope.State.Errors, 2); errs != "" {
		lines = append(lines, "Errors: "+escapeTelegramMarkdown(errs))
	}
	if preview := firstNonEmptyBrowserPreview(envelope.Tool.Preview, envelope.State.Text, envelope.Text); preview != "" {
		lines = append(lines, "Preview: "+escapeTelegramMarkdown(preview))
	}
	if evidence := strings.TrimSpace(envelope.Evidence); evidence != "" {
		lines = append(lines, "Evidence: "+escapeTelegramMarkdown(evidence))
	}
	lines = append(lines, escapeTelegramMarkdown("browser_artifact_text_fallback"))
	return truncate(strings.Join(lines, "\n"))
}

func escapeTelegramMarkdown(text string) string {
	return tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, text)
}

func joinBrowserArtifactLines(lines []string, limit int) string {
	var kept []string
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			continue
		}
		kept = append(kept, line)
		if limit > 0 && len(kept) >= limit {
			break
		}
	}
	return strings.Join(kept, "; ")
}

func firstNonEmptyBrowserPreview(candidates ...string) string {
	for _, candidate := range candidates {
		candidate = strings.Join(strings.Fields(candidate), " ")
		if candidate != "" {
			return candidate
		}
	}
	return ""
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
			if !isKnownToolTraceName(name) {
				return "🔧 tool_progress: " + quoteAndTruncate(arg, 96)
			}
			return toolTraceIcon(name) + " " + name + ": " + quoteAndTruncate(arg, 96)
		}
	}
	return "🔧 " + text
}

func isKnownToolTraceName(name string) bool {
	switch strings.TrimSpace(name) {
	case "memory", "honcho_context", "honcho_search", "honcho_profile", "honcho_conclude", "session_search",
		"search_files", "browser_navigate", "browser_snapshot", "web_search",
		"read_file", "patch", "write_file", "terminal", "execute_code", "process":
		return true
	default:
		return false
	}
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
