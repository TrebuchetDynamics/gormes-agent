package yuanbao

import (
	"regexp"
	"strings"
)

// RenderPromptSafe flattens a Yuanbao Markdown fragment into plain text that
// can be forwarded to a kernel prompt without losing URLs, mentions, or list
// structure. The renderer is intentionally narrow: it strips emphasis markers,
// converts inline and image links to "label (url)", preserves fenced code
// content without the backtick fences, and keeps mention tokens (@user)
// verbatim.
func RenderPromptSafe(input string) string {
	if input == "" {
		return ""
	}

	var b strings.Builder
	lines := strings.Split(input, "\n")
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}
		b.WriteString(renderInline(line))
		b.WriteByte('\n')
	}

	out := strings.TrimRight(b.String(), "\n")
	return out
}

// RenderPromptSafeStrict behaves like RenderPromptSafe but returns a typed
// degraded error when the input is empty or contains no renderable content.
func RenderPromptSafeStrict(input string) (string, error) {
	out := RenderPromptSafe(input)
	if strings.TrimSpace(out) == "" {
		return "", newDegraded(DegradedMarkdownParseFailed, "no renderable content")
	}
	return out, nil
}

var (
	imageLinkRE  = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	inlineLinkRE = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	boldRE       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicRE     = regexp.MustCompile(`\*([^*]+)\*`)
	inlineCodeRE = regexp.MustCompile("`([^`]+)`")
)

func renderInline(line string) string {
	out := line

	out = imageLinkRE.ReplaceAllStringFunc(out, func(match string) string {
		m := imageLinkRE.FindStringSubmatch(match)
		alt := strings.TrimSpace(m[1])
		url := strings.TrimSpace(m[2])
		if alt == "" {
			return url
		}
		return alt + " (" + url + ")"
	})

	out = inlineLinkRE.ReplaceAllStringFunc(out, func(match string) string {
		m := inlineLinkRE.FindStringSubmatch(match)
		label := strings.TrimSpace(m[1])
		url := strings.TrimSpace(m[2])
		if label == "" {
			return url
		}
		if label == url {
			return url
		}
		return label + " (" + url + ")"
	})

	out = boldRE.ReplaceAllString(out, "$1")
	out = italicRE.ReplaceAllString(out, "$1")
	out = inlineCodeRE.ReplaceAllString(out, "$1")

	return out
}
