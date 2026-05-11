package gateway

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
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
			b.WriteString(line)
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
				b.WriteByte('`')
				b.WriteString(s[i+1 : contentEnd])
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
	label := strings.TrimSpace(s[start+1 : textEnd])
	url := strings.TrimSpace(s[urlStart:urlEnd])
	if label == "" || url == "" || strings.ContainsAny(url, " \n\r\t") {
		return "", start, false
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
		return "<" + url + "|" + label + ">", urlEnd + 1, true
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

// FormatFinalDiscordText renders final assistant text for Discord markdown.
func FormatFinalDiscordText(text string) string {
	return FormatFinalMarkdown(text, FormatTargetDiscord)
}

// FormatFinalSlackText renders final assistant text for Slack mrkdwn.
func FormatFinalSlackText(text string) string {
	return FormatFinalMarkdown(text, FormatTargetSlack)
}

// FormatFinalDiscord renders the final assistant message for Discord.
func FormatFinalDiscord(f kernel.RenderFrame) string {
	return FormatFinalDiscordText(FinalAssistantText(f))
}

// FormatFinalSlack renders the final assistant message for Slack.
func FormatFinalSlack(f kernel.RenderFrame) string {
	return FormatFinalSlackText(FinalAssistantText(f))
}

