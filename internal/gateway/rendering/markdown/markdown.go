package markdown

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/telegramtext"
)

// FormatTarget specifies the messaging platform for markdown rendering.
type FormatTarget int

const (
	FormatTargetTelegram FormatTarget = iota
	FormatTargetDiscord
	FormatTargetSlack
)

// FormatFinalMarkdown renders the final assistant text as formatted markdown
// for the given platform target. Handles headings, lists, code blocks,
// blockquotes, tables, bold, italic, strikethrough, spoiler, links, and code.
func FormatFinalMarkdown(text string, target FormatTarget) string {
	if strings.TrimSpace(text) == "" {
		switch target {
		case FormatTargetTelegram:
			return "_\\(empty reply\\)_"
		case FormatTargetDiscord:
			return "*empty reply*"
		case FormatTargetSlack:
			return "_empty reply_"
		}
		return "(empty reply)"
	}

	text = stripOuterMarkdownFence(strings.ReplaceAll(text, "\r\n", "\n"))

	if target == FormatTargetTelegram {
		text = wrapTelegramMarkdownTables(text)
	}

	lines := strings.Split(text, "\n")
	var b strings.Builder
	inCode := false

	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			b.WriteString(codeFence(target))
			inCode = !inCode
			continue
		}
		if inCode {
			if target == FormatTargetTelegram {
				b.WriteString(escapeTelegramCode(line))
			} else {
				b.WriteString(line)
			}
			continue
		}
		if heading, ok := parseMarkdownHeading(line); ok {
			b.WriteString(bold(target, heading))
			continue
		}
		if indent, body, ok := parseMarkdownBullet(line); ok {
			b.WriteString(indent)
			b.WriteString(bullet(target))
			b.WriteString(renderInline(body, target))
			continue
		}
		if indent, number, body, ok := parseMarkdownNumberedItem(line); ok {
			b.WriteString(indent)
			b.WriteString(number)
			b.WriteString(numberedSep(target))
			b.WriteString(renderInline(body, target))
			continue
		}
		if prefix, body, ok := parseMarkdownBlockquote(line); ok {
			b.WriteString(prefix)
			if body != "" {
				b.WriteByte(' ')
			}
			b.WriteString(renderInline(body, target))
			continue
		}
		b.WriteString(renderInline(line, target))
	}
	if inCode {
		if target == FormatTargetTelegram {
			b.WriteByte('\n')
		}
		b.WriteString(codeFence(target))
	}
	return b.String()
}

// renderInline renders inline formatting for the target platform.
func renderInline(s string, target FormatTarget) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '`' {
			if end := strings.IndexByte(s[i+1:], '`'); end >= 0 {
				contentEnd := i + 1 + end
				content := s[i+1 : contentEnd]
				if target == FormatTargetTelegram {
					content = escapeTelegramCode(content)
				}
				b.WriteByte('`')
				b.WriteString(content)
				b.WriteByte('`')
				i = contentEnd + 1
				continue
			}
		}
		if s[i] == '[' {
			if rendered, next, ok := link(s, i, target); ok {
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
					b.WriteString(bold(target, content))
					i = contentEnd + 2
					continue
				}
			}
		}
		if strings.HasPrefix(s[i:], "~~") {
			if end := strings.Index(s[i+2:], "~~"); end >= 0 {
				contentEnd := i + 2 + end
				content := strings.TrimSpace(s[i+2 : contentEnd])
				if content != "" {
					b.WriteString(strike(target, content))
					i = contentEnd + 2
					continue
				}
			}
		}
		if target == FormatTargetTelegram && strings.HasPrefix(s[i:], "||") {
			if end := strings.Index(s[i+2:], "||"); end >= 0 {
				contentEnd := i + 2 + end
				content := strings.TrimSpace(s[i+2 : contentEnd])
				if content != "" {
					b.WriteString("||")
					b.WriteString(esc(target, content))
					b.WriteString("||")
					i = contentEnd + 2
					continue
				}
			}
		}
		if s[i] == '_' || (s[i] == '*' && target != FormatTargetSlack) {
			closer := byte('_')
			if s[i] == '*' {
				closer = '*'
			}
			if end := strings.IndexByte(s[i+1:], closer); end >= 0 {
				contentEnd := i + 1 + end
				content := s[i+1 : contentEnd]
				if inlineDelimiterBounded(s, i, contentEnd) && strings.TrimSpace(content) != "" {
					b.WriteString(italic(target, content))
					i = contentEnd + 1
					continue
				}
			}
		}
		next := nextInlineSpecial(s, i+1)
		b.WriteString(esc(target, s[i:next]))
		i = next
	}
	return b.String()
}

func link(s string, start int, target FormatTarget) (string, int, bool) {
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
	label := sanitizeLinkLabel(s[start+1 : textEnd])
	url := strings.TrimSpace(s[urlStart:urlEnd])
	if label == "" || url == "" || strings.ContainsAny(url, " \n\r\t") {
		return "", start, false
	}
	if unsafeLinkURL(url) {
		return esc(target, label), urlEnd + 1, true
	}
	switch target {
	case FormatTargetTelegram:
		if !telegramLinkURLAllowed(url) {
			return esc(target, label+" ("+url+")"), urlEnd + 1, true
		}
		return "[" + esc(target, label) + "](" + esc(target, url) + ")", urlEnd + 1, true
	case FormatTargetDiscord:
		return "[" + label + "](" + url + ")", urlEnd + 1, true
	case FormatTargetSlack:
		return "<" + esc(target, url) + "|" + esc(target, label) + ">", urlEnd + 1, true
	}
	return "", start, false
}

// Platform formatting helpers.

func bold(target FormatTarget, text string) string {
	switch target {
	case FormatTargetTelegram:
		return "*" + esc(target, text) + "*"
	case FormatTargetDiscord:
		return "**" + text + "**"
	case FormatTargetSlack:
		return "*" + text + "*"
	}
	return text
}

func italic(target FormatTarget, text string) string {
	switch target {
	case FormatTargetTelegram:
		return "_" + esc(target, text) + "_"
	case FormatTargetDiscord:
		return "*" + text + "*"
	case FormatTargetSlack:
		return "_" + text + "_"
	}
	return text
}

func strike(target FormatTarget, text string) string {
	switch target {
	case FormatTargetTelegram:
		return "~" + esc(target, text) + "~"
	case FormatTargetDiscord:
		return "~~" + text + "~~"
	case FormatTargetSlack:
		return "~" + text + "~"
	}
	return text
}

func bullet(target FormatTarget) string {
	switch target {
	case FormatTargetTelegram:
		return "• "
	case FormatTargetDiscord, FormatTargetSlack:
		return "- "
	}
	return "- "
}

func numberedSep(target FormatTarget) string {
	switch target {
	case FormatTargetTelegram:
		return `\. `
	case FormatTargetDiscord, FormatTargetSlack:
		return ". "
	}
	return ". "
}

func codeFence(target FormatTarget) string {
	return "```"
}

func escapeTelegramCode(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, "`", "\\`")
}

func escapeTelegramMarkdown(text string) string {
	return telegramtext.EscapeMarkdownV2(text)
}

func esc(target FormatTarget, text string) string {
	switch target {
	case FormatTargetTelegram:
		return escapeTelegramMarkdown(text)
	case FormatTargetSlack:
		text = strings.ReplaceAll(text, "&", "&amp;")
		text = strings.ReplaceAll(text, "<", "&lt;")
		text = strings.ReplaceAll(text, ">", "&gt;")
		return text
	default:
		return text
	}
}

func wrapTelegramMarkdownTables(text string) string {
	if !strings.Contains(text, "|") || !strings.Contains(text, "-") {
		return text
	}
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inFence := false
	for i := 0; i < len(lines); {
		line := lines[i]
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "```") {
			inFence = !inFence
			out = append(out, line)
			i++
			continue
		}
		if inFence {
			out = append(out, line)
			i++
			continue
		}
		if strings.Contains(line, "|") && i+1 < len(lines) && isTelegramMarkdownTableSeparator(lines[i+1]) {
			block := []string{line, lines[i+1]}
			j := i + 2
			for j < len(lines) && isTelegramMarkdownTableRow(lines[j]) {
				block = append(block, lines[j])
				j++
			}
			out = append(out, renderTelegramMarkdownTable(block))
			i = j
			continue
		}
		out = append(out, line)
		i++
	}
	return strings.Join(out, "\n")
}

func isTelegramMarkdownTableRow(line string) bool {
	return strings.TrimSpace(line) != "" && strings.Contains(line, "|")
}

func isTelegramMarkdownTableSeparator(line string) bool {
	cells := splitTelegramMarkdownTableRow(line)
	if len(cells) < 2 {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			return false
		}
		if strings.HasPrefix(cell, ":") {
			cell = strings.TrimPrefix(cell, ":")
		}
		if strings.HasSuffix(cell, ":") {
			cell = strings.TrimSuffix(cell, ":")
		}
		if len(cell) == 0 {
			return false
		}
		for _, r := range cell {
			if r != '-' {
				return false
			}
		}
	}
	return true
}

func splitTelegramMarkdownTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "|") {
		trimmed = strings.TrimPrefix(trimmed, "|")
	}
	if strings.HasSuffix(trimmed, "|") {
		trimmed = strings.TrimSuffix(trimmed, "|")
	}
	parts := strings.Split(trimmed, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func renderTelegramMarkdownTable(block []string) string {
	if len(block) < 3 {
		return strings.Join(block, "\n")
	}
	headers := splitTelegramMarkdownTableRow(block[0])
	if len(headers) < 2 {
		return strings.Join(block, "\n")
	}
	firstDataRow := []string(nil)
	if len(block) > 2 {
		firstDataRow = splitTelegramMarkdownTableRow(block[2])
	}
	hasRowLabelColumn := len(firstDataRow) == len(headers)+1
	var rows []string
	for index, line := range block[2:] {
		cells := splitTelegramMarkdownTableRow(line)
		heading := fmt.Sprintf("Row %d", index+1)
		dataCells := cells
		if hasRowLabelColumn {
			if len(cells) > 0 && strings.TrimSpace(cells[0]) != "" {
				heading = cells[0]
			}
			if len(cells) > 1 {
				dataCells = cells[1:]
			} else {
				dataCells = nil
			}
		} else {
			for _, cell := range cells {
				if strings.TrimSpace(cell) != "" {
					heading = cell
					break
				}
			}
		}
		for len(dataCells) < len(headers) {
			dataCells = append(dataCells, "")
		}
		if len(dataCells) > len(headers) {
			dataCells = dataCells[:len(headers)]
		}

		lines := []string{"**" + heading + "**"}
		for i, header := range headers {
			lines = append(lines, "• "+header+": "+dataCells[i])
		}
		rows = append(rows, strings.Join(lines, "\n"))
	}
	return strings.Join(rows, "\n\n")
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

func parseMarkdownBlockquote(line string) (string, string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, ">") {
		return "", "", false
	}
	i := 0
	for i < len(trimmed) && trimmed[i] == '>' {
		i++
	}
	body := strings.TrimSpace(trimmed[i:])
	return strings.Repeat(">", i), body, body != ""
}

func leadingWhitespaceLen(s string) int {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}

func sanitizeLinkLabel(label string) string {
	fields := strings.Fields(label)
	for i, field := range fields {
		lower := strings.ToLower(field)
		switch lower {
		case "@everyone", "@here", "@channel":
			fields[i] = strings.TrimPrefix(field, "@")
		}
	}
	return strings.Join(fields, " ")
}

func telegramLinkURLAllowed(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "tg://")
}

func unsafeLinkURL(url string) bool {
	lower := strings.ToLower(strings.TrimSpace(url))
	if containsLinkControl(lower) || containsLinkMention(lower) {
		return true
	}
	if idx := strings.Index(lower, ":"); idx > 0 {
		scheme := lower[:idx]
		return scheme != "http" && scheme != "https" && scheme != "tg"
	}
	return false
}

func containsLinkControl(value string) bool {
	return strings.ContainsFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) })
}

func containsLinkMention(value string) bool {
	return strings.Contains(value, "@everyone") || strings.Contains(value, "@here") || strings.Contains(value, "@channel")
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
	for _, marker := range []string{"`", "[", "**", "~~", "||", "_", "*"} {
		if idx := strings.Index(s[start:], marker); idx >= 0 && start+idx < next {
			next = start + idx
		}
	}
	return next
}

func demoteBroadcastMentions(text string) string {
	lower := strings.ToLower(text)
	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(text); {
		switch {
		case strings.HasPrefix(lower[i:], "@everyone"):
			b.WriteString("everyone")
			i += len("@everyone")
		case strings.HasPrefix(lower[i:], "@channel"):
			b.WriteString("channel")
			i += len("@channel")
		case strings.HasPrefix(lower[i:], "@here"):
			b.WriteString("here")
			i += len("@here")
		default:
			b.WriteByte(text[i])
			i++
		}
	}
	return b.String()
}

// FormatFinalDiscordText renders final assistant text for Discord markdown.
func FormatFinalDiscordText(text string) string {
	return demoteBroadcastMentions(FormatFinalMarkdown(text, FormatTargetDiscord))
}

// FormatFinalSlackText renders final assistant text for Slack mrkdwn.
func FormatFinalSlackText(text string) string {
	return demoteBroadcastMentions(FormatFinalMarkdown(text, FormatTargetSlack))
}
