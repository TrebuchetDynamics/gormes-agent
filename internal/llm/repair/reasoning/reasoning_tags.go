package reasoning

import (
	"regexp"
	"strings"
)

type tagPattern struct {
	closed       *regexp.Regexp
	unterminated *regexp.Regexp
	orphan       *regexp.Regexp
}

var (
	tagNames = []string{
		"think",
		"thinking",
		"reasoning",
		"thought",
		"REASONING_SCRATCHPAD",
	}
	tagPatterns = []tagPattern{
		newTagPattern(tagNames[0]),
		newTagPattern(tagNames[1]),
		newTagPattern(tagNames[2]),
		newTagPattern(tagNames[3]),
		newTagPattern(tagNames[4]),
	}
	blankLines = regexp.MustCompile(`[ \t]*\n[ \t]*\n+[ \t]*`)
)

func newTagPattern(tag string) tagPattern {
	name := regexp.QuoteMeta(tag)
	return tagPattern{
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

// SanitizeTags returns visible assistant text with inline reasoning XML blocks
// removed. Callers must keep raw stream/transcript text separately for audit
// rather than treating this sanitized copy as source evidence.
func SanitizeTags(text string) string {
	if text == "" {
		return ""
	}
	cleaned := text
	for _, pattern := range tagPatterns {
		cleaned = pattern.closed.ReplaceAllString(cleaned, "")
		cleaned = pattern.unterminated.ReplaceAllString(cleaned, "")
		cleaned = pattern.orphan.ReplaceAllString(cleaned, "")
	}
	cleaned = strings.TrimSpace(cleaned)
	cleaned = blankLines.ReplaceAllString(cleaned, "\n")
	return strings.TrimSpace(cleaned)
}

// ContainsTagMarker reports whether text contains a provider-leaked reasoning
// XML marker. It is a cheap guard for the streaming sanitizer.
func ContainsTagMarker(text string) bool {
	lower := strings.ToLower(text)
	for _, tag := range tagNames {
		name := strings.ToLower(tag)
		if strings.Contains(lower, "<"+name) || strings.Contains(lower, "</"+name) {
			return true
		}
	}
	return false
}

// SanitizeStreamChunk removes provider-leaked reasoning XML blocks from one
// streamed content chunk while preserving state across chunks. The returned
// bool is true when the next chunk starts inside a reasoning block.
func SanitizeStreamChunk(text string, inReasoning bool) (string, bool) {
	if text == "" {
		return "", inReasoning
	}
	lower := strings.ToLower(text)
	var out strings.Builder
	pos := 0
	for pos < len(text) {
		if inReasoning {
			match, ok := nextTag(lower, pos, true)
			if !ok {
				return out.String(), true
			}
			end := tagEnd(text, match.start)
			if end < 0 {
				return out.String(), true
			}
			pos = end
			inReasoning = false
			continue
		}

		match, ok := nextTagAny(lower, pos)
		if !ok {
			out.WriteString(text[pos:])
			break
		}
		out.WriteString(text[pos:match.start])
		end := tagEnd(text, match.start)
		if end < 0 {
			if match.closing {
				break
			}
			return out.String(), true
		}
		pos = end
		inReasoning = !match.closing
	}
	cleaned := blankLines.ReplaceAllString(out.String(), "\n")
	return cleaned, inReasoning
}

type streamMatch struct {
	start   int
	closing bool
}

func nextTagAny(lower string, from int) (streamMatch, bool) {
	open, hasOpen := nextTag(lower, from, false)
	close, hasClose := nextTag(lower, from, true)
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

func nextTag(lower string, from int, closing bool) (streamMatch, bool) {
	best := streamMatch{start: -1, closing: closing}
	for _, tag := range tagNames {
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
			if tagNameBoundary(lower, idx+len(prefix)) {
				if best.start < 0 || idx < best.start {
					best.start = idx
				}
				break
			}
			searchFrom = idx + 1
		}
	}
	if best.start < 0 {
		return streamMatch{}, false
	}
	return best, true
}

func tagNameBoundary(lower string, idx int) bool {
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

func tagEnd(text string, start int) int {
	if start < 0 || start >= len(text) {
		return -1
	}
	idx := strings.IndexByte(text[start:], '>')
	if idx < 0 {
		return -1
	}
	return start + idx + 1
}
