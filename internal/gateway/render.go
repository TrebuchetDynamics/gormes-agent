package gateway

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tooltrace"
)

const (
	maxMessageLen       = 4000
	streamPreviewCursor = " ▉"
)

// FormatStreamPlain renders a streaming frame as plain text.
func FormatStreamPlain(f kernel.RenderFrame) string {
	body := f.DraftText
	if streamPreviewCursorActive(f) {
		body += streamPreviewCursor
	}
	tail := ""
	if f.Phase == kernel.PhaseReconnecting {
		tail += "\n\nreconnecting…"
	}
	return truncate(body + tail)
}

// FormatToolProgressPlain renders the persistent Hermes-style tool progress
// transcript for gateway platforms that can edit progress messages.
func FormatToolProgressPlain(f kernel.RenderFrame) string {
	return FormatToolProgressPlainMode(f, "all")
}

// FormatToolProgressPlainMode renders tool progress with Hermes gateway
// display.tool_progress semantics for the compact progress transcript.
func FormatToolProgressPlainMode(f kernel.RenderFrame, mode string) string {
	mode = normalizeGatewayToolProgressMode(mode)
	if mode == "off" {
		return ""
	}
	return truncate(formatToolTraceBlockPlainMode(f.SoulEvents, mode))
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
	return text
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
	if streamPreviewCursorActive(f) {
		body += streamPreviewCursor
	}
	tail := ""
	if f.Phase == kernel.PhaseReconnecting {
		tail += "\n\n_reconnecting…_"
	}
	return truncate(body + tail)
}

// FormatToolProgressTelegram renders escaped MarkdownV2 text for Telegram's
// persistent tool-progress message.
func FormatToolProgressTelegram(f kernel.RenderFrame) string {
	return FormatToolProgressTelegramMode(f, "all")
}

func FormatToolProgressTelegramMode(f kernel.RenderFrame, mode string) string {
	progress := FormatToolProgressPlainMode(f, mode)
	if strings.TrimSpace(progress) == "" {
		return ""
	}
	return truncate(tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, progress))
}

// FormatFinalTelegram renders the final assistant message for Telegram.
func FormatFinalTelegram(f kernel.RenderFrame) string {
	return FormatFinalTelegramText(FinalAssistantText(f))
}

func FormatFinalTelegramText(text string) string {
	if strings.TrimSpace(text) == "" {
		return "_\\(empty reply\\)_"
	}
	return renderTelegramFinalMarkdown(text)
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

func renderTelegramFinalMarkdown(text string) string {
	text = stripOuterMarkdownFence(strings.ReplaceAll(text, "\r\n", "\n"))
	lines := strings.Split(text, "\n")
	var b strings.Builder
	inCode := false

	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inCode {
				b.WriteString("```")
				inCode = false
				continue
			}
			b.WriteString("```")
			inCode = true
			continue
		}
		if inCode {
			b.WriteString(escapeTelegramCode(line))
			continue
		}
		if heading, ok := parseMarkdownHeading(line); ok {
			b.WriteByte('*')
			b.WriteString(escapeTelegramMarkdown(heading))
			b.WriteByte('*')
			continue
		}
		if indent, body, ok := parseMarkdownBullet(line); ok {
			b.WriteString(indent)
			b.WriteString("• ")
			b.WriteString(renderTelegramInlineMarkdown(body))
			continue
		}
		if indent, number, body, ok := parseMarkdownNumberedItem(line); ok {
			b.WriteString(indent)
			b.WriteString(number)
			b.WriteString(`\. `)
			b.WriteString(renderTelegramInlineMarkdown(body))
			continue
		}
		if body, ok := parseMarkdownBlockquote(line); ok {
			b.WriteString("❯ ")
			b.WriteString(renderTelegramInlineMarkdown(body))
			continue
		}
		b.WriteString(renderTelegramInlineMarkdown(line))
	}
	if inCode {
		b.WriteString("\n```")
	}
	return b.String()
}

func stripOuterMarkdownFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") || !strings.HasSuffix(trimmed, "```") {
		return text
	}
	firstLineEnd := strings.Index(trimmed, "\n")
	if firstLineEnd < 0 {
		return text
	}
	lang := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed[:firstLineEnd], "```")))
	switch lang {
	case "markdown", "md", "mdx":
	default:
		return text
	}
	body := strings.TrimSuffix(trimmed[firstLineEnd+1:], "```")
	return strings.Trim(body, "\n")
}

func parseMarkdownHeading(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || level >= len(trimmed) || trimmed[level] != ' ' {
		return "", false
	}
	heading := strings.TrimSpace(trimmed[level:])
	if heading == "" {
		return "", false
	}
	return heading, true
}

func parseMarkdownBullet(line string) (string, string, bool) {
	indentLen := leadingWhitespaceLen(line)
	if indentLen >= len(line) {
		return "", "", false
	}
	marker := line[indentLen]
	if marker != '-' && marker != '*' && marker != '+' {
		return "", "", false
	}
	if indentLen+1 >= len(line) || line[indentLen+1] != ' ' {
		return "", "", false
	}
	body := strings.TrimSpace(line[indentLen+2:])
	if body == "" {
		return "", "", false
	}
	return line[:indentLen], body, true
}

func parseMarkdownNumberedItem(line string) (string, string, string, bool) {
	indentLen := leadingWhitespaceLen(line)
	i := indentLen
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == indentLen || i >= len(line) {
		return "", "", "", false
	}
	if line[i] != '.' && line[i] != ')' {
		return "", "", "", false
	}
	if i+1 >= len(line) || line[i+1] != ' ' {
		return "", "", "", false
	}
	body := strings.TrimSpace(line[i+2:])
	if body == "" {
		return "", "", "", false
	}
	return line[:indentLen], line[indentLen:i], body, true
}

func parseMarkdownBlockquote(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, ">") {
		return "", false
	}
	body := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
	return body, body != ""
}

func leadingWhitespaceLen(s string) int {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}

func renderTelegramInlineMarkdown(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '`' {
			if end := strings.IndexByte(s[i+1:], '`'); end >= 0 {
				contentEnd := i + 1 + end
				b.WriteByte('`')
				b.WriteString(escapeTelegramCode(s[i+1 : contentEnd]))
				b.WriteByte('`')
				i = contentEnd + 1
				continue
			}
		}
		if s[i] == '[' {
			if rendered, next, ok := renderTelegramInlineLink(s, i); ok {
				b.WriteString(rendered)
				i = next
				continue
			}
		}
		if strings.HasPrefix(s[i:], "**") {
			if end := strings.Index(s[i+2:], "**"); end >= 0 {
				contentEnd := i + 2 + end
				content := strings.TrimSpace(s[i+2 : contentEnd])
				if content != "" {
					b.WriteByte('*')
					b.WriteString(escapeTelegramMarkdown(content))
					b.WriteByte('*')
					i = contentEnd + 2
					continue
				}
			}
		}
		if s[i] == '_' {
			if end := strings.IndexByte(s[i+1:], '_'); end >= 0 {
				contentEnd := i + 1 + end
				content := s[i+1 : contentEnd]
				if inlineDelimiterBounded(s, i, contentEnd) && strings.TrimSpace(content) != "" {
					b.WriteByte('_')
					b.WriteString(escapeTelegramMarkdown(content))
					b.WriteByte('_')
					i = contentEnd + 1
					continue
				}
			}
		}
		if s[i] == '*' {
			if end := strings.IndexByte(s[i+1:], '*'); end >= 0 {
				contentEnd := i + 1 + end
				content := s[i+1 : contentEnd]
				if inlineDelimiterBounded(s, i, contentEnd) && strings.TrimSpace(content) != "" {
					b.WriteByte('_')
					b.WriteString(escapeTelegramMarkdown(content))
					b.WriteByte('_')
					i = contentEnd + 1
					continue
				}
			}
		}

		next := nextInlineSpecial(s, i+1)
		b.WriteString(escapeTelegramMarkdown(s[i:next]))
		i = next
	}
	return b.String()
}

func renderTelegramInlineLink(s string, start int) (string, int, bool) {
	closeText := strings.Index(s[start+1:], "](")
	if closeText < 0 {
		return "", start, false
	}
	textEnd := start + 1 + closeText
	urlStart := textEnd + 2
	closeURL := strings.IndexByte(s[urlStart:], ')')
	if closeURL < 0 {
		return "", start, false
	}
	urlEnd := urlStart + closeURL
	label := strings.TrimSpace(s[start+1 : textEnd])
	url := strings.TrimSpace(s[urlStart:urlEnd])
	if label == "" || url == "" || strings.ContainsAny(url, " \n\r\t") {
		return "", start, false
	}
	if !telegramLinkURLAllowed(url) {
		return escapeTelegramMarkdown(label + " (" + url + ")"), urlEnd + 1, true
	}
	return "[" + escapeTelegramMarkdown(label) + "](" + escapeTelegramLinkURL(url) + ")", urlEnd + 1, true
}

func telegramLinkURLAllowed(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "tg://")
}

func escapeTelegramLinkURL(url string) string {
	return escapeTelegramMarkdown(url)
}

func escapeTelegramCode(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, "`", "\\`")
}

func inlineDelimiterBounded(s string, start, end int) bool {
	if start+1 >= end {
		return false
	}
	if isASCIISpace(s[start+1]) || isASCIISpace(s[end-1]) {
		return false
	}
	beforeOK := start == 0 || isInlineBoundaryByte(s[start-1])
	afterOK := end+1 >= len(s) || isInlineBoundaryByte(s[end+1])
	return beforeOK && afterOK
}

func isInlineBoundaryByte(b byte) bool {
	return isASCIISpace(b) || strings.ContainsRune(".,;:!?)]}\"'", rune(b))
}

func isASCIISpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func nextInlineSpecial(s string, start int) int {
	next := len(s)
	for _, marker := range []string{"`", "[", "**", "_", "*"} {
		if idx := strings.Index(s[start:], marker); idx >= 0 && start+idx < next {
			next = start + idx
		}
	}
	return next
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

func streamPreviewCursorActive(f kernel.RenderFrame) bool {
	if strings.TrimSpace(f.DraftText) == "" {
		return false
	}
	switch f.Phase {
	case kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseFinalizing, kernel.PhaseReconnecting:
		return true
	default:
		return false
	}
}

func formatToolTracePlain(text string) string {
	return tooltrace.FormatPlain(text)
}

func formatToolTraceBlockPlain(events []kernel.SoulEntry) string {
	return formatToolTraceBlockPlainMode(events, "all")
}

func formatToolTraceBlockPlainMode(events []kernel.SoulEntry, mode string) string {
	texts := make([]string, 0, len(events))
	for _, event := range events {
		texts = append(texts, event.Text)
	}
	return tooltrace.FormatBlockMode(texts, mode)
}

func toolTraceName(text string) string {
	payload := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "tool: "))
	name, _, ok := strings.Cut(payload, ":")
	if ok {
		return strings.TrimSpace(name)
	}
	return payload
}

func normalizeGatewayToolProgressMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case "off", "new", "all", "verbose":
		return normalized
	default:
		return "all"
	}
}

func isKnownToolTraceName(name string) bool {
	switch strings.TrimSpace(name) {
	case "memory", "honcho_context", "honcho_search", "honcho_profile", "honcho_conclude", "session_search",
		"skill_view", "skills_list", "skill_manage", "todo", "cronjob",
		"search_files", "web_search", "web_extract", "web_crawl",
		"browser_navigate", "browser_snapshot", "browser_click", "browser_type", "browser_scroll",
		"browser_back", "browser_press", "browser_get_images", "browser_vision", "browser_cdp", "browser_dialog",
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
	case "skill_view", "skills_list", "skill_manage":
		return "📚"
	case "todo":
		return "📋"
	case "cronjob":
		return "⏰"
	case "search_files":
		return "🔎"
	case "web_search":
		return "🔍"
	case "web_extract":
		return "📄"
	case "web_crawl":
		return "🕸️"
	case "browser_navigate":
		return "🌐"
	case "browser_snapshot":
		return "📸"
	case "browser_click":
		return "👆"
	case "browser_type", "browser_press":
		return "⌨️"
	case "browser_scroll":
		return "📜"
	case "browser_back":
		return "◀️"
	case "browser_get_images":
		return "🖼️"
	case "browser_vision":
		return "👁️"
	case "browser_cdp", "browser_dialog":
		return "🖥️"
	case "read_file":
		return "📖"
	case "patch", "write_file":
		return "🔧"
	case "terminal", "process":
		return "💻"
	case "execute_code":
		return "🐍"
	default:
		return "🔧"
	}
}

func quoteAndTruncate(s string, limit int) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if limit > 0 {
		runes := []rune(s)
		if len(runes) > limit {
			if limit <= 3 {
				s = string(runes[:limit])
			} else {
				s = string(runes[:limit-3]) + "..."
			}
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

func paginatePlainText(s string) []string {
	return paginateOutboundText(s, plainPageMarker)
}

func paginateTelegramText(s string) []string {
	return paginateOutboundText(s, telegramPageMarker)
}

func paginateOutboundText(s string, marker func(int, int) string) []string {
	if marker == nil {
		marker = plainPageMarker
	}
	if len([]rune(s)) <= maxMessageLen {
		return []string{s}
	}
	const markerReserve = 32
	bodyLimit := maxMessageLen - markerReserve
	if bodyLimit < 1 {
		bodyLimit = maxMessageLen / 2
	}
	chunks := splitOutboundText(s, bodyLimit)
	if len(chunks) <= 1 {
		return chunks
	}
	pages := make([]string, len(chunks))
	total := len(chunks)
	for i, chunk := range chunks {
		pages[i] = chunk + marker(i+1, total)
	}
	return pages
}

func splitOutboundText(s string, limit int) []string {
	if limit <= 0 || len([]rune(s)) <= limit {
		return []string{s}
	}
	var chunks []string
	remaining := []rune(s)
	for len(remaining) > limit {
		split := outboundSplitIndex(remaining, limit)
		if split <= 0 || split > len(remaining) {
			split = limit
		}
		for split > 1 && remaining[split-1] == '\\' {
			split--
		}
		chunks = append(chunks, string(remaining[:split]))
		remaining = remaining[split:]
	}
	if len(remaining) > 0 {
		chunks = append(chunks, string(remaining))
	}
	return chunks
}

func outboundSplitIndex(runes []rune, limit int) int {
	if len(runes) <= limit {
		return len(runes)
	}
	window := runes[:limit]
	if idx := lastRuneIndex(window, '\n'); idx >= limit/2 {
		return idx + 1
	}
	if idx := lastRuneIndex(window, ' '); idx >= limit/2 {
		return idx + 1
	}
	return limit
}

func lastRuneIndex(runes []rune, needle rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == needle {
			return i
		}
	}
	return -1
}

func plainPageMarker(page, total int) string {
	return fmt.Sprintf("\n\n(%d/%d)", page, total)
}

func telegramPageMarker(page, total int) string {
	return fmt.Sprintf("\n\n\\(%d/%d\\)", page, total)
}
