package browserartifacts

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/pagination"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// FormatBrowserArtifactTelegram renders a browser result envelope as a bounded
// Telegram MarkdownV2 artifact notice. It keeps browser/tool progress separate
// from final assistant text and never exposes local screenshot paths or raw
// persisted artifact bytes.
func Telegram(envelope tools.BrowserResultEnvelope) string {
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

func truncate(s string) string {
	runes := []rune(s)
	if len(runes) <= pagination.MaxMessageLen {
		return s
	}
	end := pagination.MaxMessageLen - 1
	for end > 0 && runes[end-1] == '\\' {
		end--
	}
	return string(runes[:end]) + "…"
}
