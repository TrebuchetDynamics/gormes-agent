package llm

import (
	"regexp"
	"strings"
)

type reasoningTagPattern struct {
	closed       *regexp.Regexp
	unterminated *regexp.Regexp
	orphan       *regexp.Regexp
}

var (
	reasoningTagNames = []string{
		"think",
		"thinking",
		"reasoning",
		"thought",
		"REASONING_SCRATCHPAD",
	}
	reasoningTagPatterns = []reasoningTagPattern{
		newReasoningTagPattern(reasoningTagNames[0]),
		newReasoningTagPattern(reasoningTagNames[1]),
		newReasoningTagPattern(reasoningTagNames[2]),
		newReasoningTagPattern(reasoningTagNames[3]),
		newReasoningTagPattern(reasoningTagNames[4]),
	}
	reasoningTagBlankLines = regexp.MustCompile(`[ \t]*\n[ \t]*\n+[ \t]*`)
)

func newReasoningTagPattern(tag string) reasoningTagPattern {
	name := regexp.QuoteMeta(tag)
	return reasoningTagPattern{
		closed: regexp.MustCompile(
			`(?is)<` + name + `\b[^>]*>.*?</` + name + `\s*>[ \t]*`,
		),
		unterminated: regexp.MustCompile(
			`(?is)(?:^|\n)[ \t]*<` + name + `\b[^>]*>.*$`,
		),
		orphan: regexp.MustCompile(
			`(?is)</?` + name + `\b[^>]*>[ \t]*`,
		),
	}
}

// SanitizeReasoningTags returns visible assistant text with inline reasoning
// XML blocks removed. Callers must keep raw stream/transcript text separately
// for audit rather than treating this sanitized copy as source evidence.
func SanitizeReasoningTags(text string) string {
	if text == "" {
		return ""
	}
	cleaned := text
	for _, pattern := range reasoningTagPatterns {
		cleaned = pattern.closed.ReplaceAllString(cleaned, "")
		cleaned = pattern.unterminated.ReplaceAllString(cleaned, "")
		cleaned = pattern.orphan.ReplaceAllString(cleaned, "")
	}
	cleaned = strings.TrimSpace(cleaned)
	cleaned = reasoningTagBlankLines.ReplaceAllString(cleaned, "\n")
	return strings.TrimSpace(cleaned)
}

// ContainsReasoningTagMarker reports whether text contains a provider-leaked
// reasoning XML marker. It is a cheap guard for the streaming sanitizer.
func ContainsReasoningTagMarker(text string) bool {
	lower := strings.ToLower(text)
	for _, tag := range reasoningTagNames {
		name := strings.ToLower(tag)
		if strings.Contains(lower, "<"+name) || strings.Contains(lower, "</"+name) {
			return true
		}
	}
	return false
}

// SanitizeReasoningStreamChunk removes provider-leaked reasoning XML blocks
// from one streamed content chunk while preserving state across chunks. The
// returned bool is true when the next chunk starts inside a reasoning block.
func SanitizeReasoningStreamChunk(text string, inReasoning bool) (string, bool) {
	if text == "" {
		return "", inReasoning
	}
	lower := strings.ToLower(text)
	var out strings.Builder
	pos := 0
	for pos < len(text) {
		if inReasoning {
			match, ok := nextReasoningTag(lower, pos, true)
			if !ok {
				return out.String(), true
			}
			end := reasoningTagEnd(text, match.start)
			if end < 0 {
				return out.String(), true
			}
			pos = end
			inReasoning = false
			continue
		}

		match, ok := nextReasoningTagAny(lower, pos)
		if !ok {
			out.WriteString(text[pos:])
			break
		}
		out.WriteString(text[pos:match.start])
		end := reasoningTagEnd(text, match.start)
		if end < 0 {
			if match.closing {
				break
			}
			return out.String(), true
		}
		pos = end
		inReasoning = !match.closing
	}
	cleaned := reasoningTagBlankLines.ReplaceAllString(out.String(), "\n")
	return cleaned, inReasoning
}

type reasoningTagStreamMatch struct {
	start   int
	closing bool
}

func nextReasoningTagAny(lower string, from int) (reasoningTagStreamMatch, bool) {
	open, hasOpen := nextReasoningTag(lower, from, false)
	close, hasClose := nextReasoningTag(lower, from, true)
	switch {
	case !hasOpen:
		return close, hasClose
	case !hasClose:
		return open, true
	case close.start < open.start:
		return close, true
	default:
		return open, true
	}
}

func nextReasoningTag(lower string, from int, closing bool) (reasoningTagStreamMatch, bool) {
	best := reasoningTagStreamMatch{start: -1, closing: closing}
	for _, tag := range reasoningTagNames {
		name := strings.ToLower(tag)
		prefix := "<" + name
		if closing {
			prefix = "</" + name
		}
		searchFrom := from
		for searchFrom < len(lower) {
			idx := strings.Index(lower[searchFrom:], prefix)
			if idx < 0 {
				break
			}
			idx += searchFrom
			if reasoningTagNameBoundary(lower, idx+len(prefix)) {
				if best.start < 0 || idx < best.start {
					best.start = idx
				}
				break
			}
			searchFrom = idx + 1
		}
	}
	if best.start < 0 {
		return reasoningTagStreamMatch{}, false
	}
	return best, true
}

func reasoningTagNameBoundary(lower string, idx int) bool {
	if idx >= len(lower) {
		return true
	}
	switch lower[idx] {
	case '>', '/', ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func reasoningTagEnd(text string, start int) int {
	if start < 0 || start >= len(text) {
		return -1
	}
	idx := strings.IndexByte(text[start:], '>')
	if idx < 0 {
		return -1
	}
	return start + idx + 1
}
