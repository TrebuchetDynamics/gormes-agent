package browserartifacts

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/pagination"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/telegramtext"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/textlimit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// FormatBrowserArtifactTelegram renders a browser result envelope as a bounded
// Telegram MarkdownV2 artifact notice. It keeps browser/tool progress separate
// from final assistant text and never exposes local screenshot paths or raw
// persisted artifact bytes.
func Telegram(envelope tools.BrowserResultEnvelope) string {
	var lines []string
	lines = append(lines, "🌐 *Browser artifact*")
	if title := browserArtifactLineValue(envelope.State.Title); title != "" {
		lines = append(lines, "Title: "+escapeTelegramMarkdown(title))
	}
	if u := browserArtifactLineValue(envelope.State.URL); u != "" {
		lines = append(lines, "URL: "+escapeTelegramMarkdown(u))
	}
	if artifact := browserArtifactLineValue(envelope.Tool.Artifact); artifact != "" {
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
	if evidence := browserArtifactLineValue(envelope.Evidence); evidence != "" {
		lines = append(lines, "Evidence: "+escapeTelegramMarkdown(evidence))
	}
	lines = append(lines, escapeTelegramMarkdown("browser_artifact_text_fallback"))
	return truncate(strings.Join(lines, "\n"))
}

func escapeTelegramMarkdown(text string) string {
	return telegramtext.EscapeMarkdownV2(text)
}

func browserArtifactLineValue(value string) string {
	fields := strings.Fields(value)
	for i, field := range fields {
		trimmed := strings.Trim(field, "()[]{}.,;:'\"`")
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "~/") || strings.HasPrefix(lower, "file://") || strings.Contains(trimmed, ":\\") {
			fields[i] = "[path]"
		}
	}
	return strings.Join(fields, " ")
}

func joinBrowserArtifactLines(lines []string, limit int) string {
	var kept []string
	for _, line := range lines {
		line = browserArtifactLineValue(line)
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
		candidate = browserArtifactLineValue(candidate)
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func truncate(s string) string {
	return textlimit.TruncateMarkdownV2Safe(s, pagination.MaxMessageLen)
}
