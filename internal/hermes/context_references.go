package hermes

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/transcript/contextrefs"
)

const (
	ContextReferenceFile   = "file"
	ContextReferenceFolder = "folder"
	ContextReferenceGit    = "git"
	ContextReferenceURL    = "url"
	ContextReferenceDiff   = "diff"
	ContextReferenceStaged = "staged"
)

type ContextReference struct {
	Raw       string `json:"raw"`
	Kind      string `json:"kind"`
	Target    string `json:"target,omitempty"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	LineStart int    `json:"line_start,omitempty"`
	LineEnd   int    `json:"line_end,omitempty"`
}

type ContextReferenceHandleResult struct {
	Message         string               `json:"message"`
	OriginalMessage string               `json:"original_message"`
	References      []ContextReference   `json:"references"`
	Handles         []contextrefs.Handle `json:"handles"`
}

func ParseContextReferences(message string) []ContextReference {
	if message == "" {
		return nil
	}
	refs := make([]ContextReference, 0)
	for i := 0; i < len(message); i++ {
		if message[i] != '@' || referenceBlockedByPrefix(message, i) {
			continue
		}
		ref, ok := parseContextReferenceAt(message, i)
		if !ok {
			continue
		}
		refs = append(refs, ref)
		i = ref.End - 1
	}
	return refs
}

func AttachContextReferenceHandles(message string, store *contextrefs.Store) ContextReferenceHandleResult {
	refs := ParseContextReferences(message)
	handles := make([]contextrefs.Handle, 0, len(refs))
	if store == nil {
		store = contextrefs.NewStore()
	}
	for _, ref := range refs {
		handles = append(handles, store.Put(contextrefs.Record{
			Raw:       ref.Raw,
			Kind:      ref.Kind,
			Target:    ref.Target,
			Start:     ref.Start,
			End:       ref.End,
			LineStart: ref.LineStart,
			LineEnd:   ref.LineEnd,
		}))
	}
	return ContextReferenceHandleResult{
		Message:         RemoveContextReferenceTokens(message, refs),
		OriginalMessage: message,
		References:      refs,
		Handles:         handles,
	}
}

func RemoveContextReferenceTokens(message string, refs []ContextReference) string {
	if len(refs) == 0 {
		return strings.TrimSpace(message)
	}
	var b strings.Builder
	cursor := 0
	for _, ref := range refs {
		if ref.Start < cursor || ref.Start > len(message) || ref.End > len(message) {
			continue
		}
		b.WriteString(message[cursor:ref.Start])
		cursor = ref.End
	}
	b.WriteString(message[cursor:])
	text := regexp.MustCompile(`\s{2,}`).ReplaceAllString(b.String(), " ")
	text = regexp.MustCompile(`\s+([,.;:!?])`).ReplaceAllString(text, "$1")
	return strings.TrimSpace(text)
}

func parseContextReferenceAt(message string, start int) (ContextReference, bool) {
	afterAt := start + 1
	for _, simple := range []string{ContextReferenceDiff, ContextReferenceStaged} {
		if strings.HasPrefix(message[afterAt:], simple) {
			end := afterAt + len(simple)
			if end == len(message) || !isASCIIWord(message[end]) {
				return ContextReference{
					Raw:   message[start:end],
					Kind:  simple,
					Start: start,
					End:   end,
				}, true
			}
		}
	}

	for _, kind := range []string{
		ContextReferenceFile,
		ContextReferenceFolder,
		ContextReferenceGit,
		ContextReferenceURL,
	} {
		prefix := kind + ":"
		if !strings.HasPrefix(message[afterAt:], prefix) {
			continue
		}
		valueStart := afterAt + len(prefix)
		valueEnd, ok := scanReferenceValue(message, valueStart)
		if !ok {
			return ContextReference{}, false
		}
		rawValue := message[valueStart:valueEnd]
		targetValue := stripReferenceTrailingPunctuation(rawValue)
		target := stripReferenceWrappers(targetValue)
		lineStart, lineEnd := 0, 0
		if kind == ContextReferenceFile {
			target, lineStart, lineEnd = parseFileReferenceValue(targetValue)
		}
		return ContextReference{
			Raw:       message[start:valueEnd],
			Kind:      kind,
			Target:    target,
			Start:     start,
			End:       valueEnd,
			LineStart: lineStart,
			LineEnd:   lineEnd,
		}, true
	}
	return ContextReference{}, false
}

func scanReferenceValue(message string, start int) (int, bool) {
	if start >= len(message) || isReferenceSpace(message[start]) {
		return 0, false
	}
	switch quote := message[start]; quote {
	case '`', '"', '\'':
		endQuote := start + 1
		for endQuote < len(message) && message[endQuote] != quote && message[endQuote] != '\n' {
			endQuote++
		}
		if endQuote >= len(message) || message[endQuote] != quote {
			return 0, false
		}
		end := endQuote + 1
		if rangeEnd, ok := scanOptionalLineRange(message, end); ok {
			end = rangeEnd
		}
		return end, true
	default:
		end := start
		for end < len(message) && !isReferenceSpace(message[end]) {
			end++
		}
		return end, end > start
	}
}

func scanOptionalLineRange(message string, start int) (int, bool) {
	if start >= len(message) || message[start] != ':' || start+1 >= len(message) || !isDigit(message[start+1]) {
		return 0, false
	}
	end := start + 2
	for end < len(message) && isDigit(message[end]) {
		end++
	}
	if end < len(message) && message[end] == '-' {
		if end+1 >= len(message) || !isDigit(message[end+1]) {
			return 0, false
		}
		end += 2
		for end < len(message) && isDigit(message[end]) {
			end++
		}
	}
	return end, true
}

func parseFileReferenceValue(value string) (string, int, int) {
	if len(value) >= 2 {
		if quote := value[0]; quote == '`' || quote == '"' || quote == '\'' {
			endQuote := strings.IndexByte(value[1:], quote)
			if endQuote >= 0 {
				endQuote++
				target := value[1:endQuote]
				lineStart, lineEnd, ok := parseLineRangeSuffix(value[endQuote+1:])
				if ok || value[endQuote+1:] == "" {
					return target, lineStart, lineEnd
				}
			}
		}
	}
	if idx := strings.LastIndexByte(value, ':'); idx > 0 {
		if lineStart, lineEnd, ok := parseLineRangeSuffix(value[idx:]); ok {
			return stripReferenceWrappers(value[:idx]), lineStart, lineEnd
		}
	}
	return stripReferenceWrappers(value), 0, 0
}

func parseLineRangeSuffix(suffix string) (int, int, bool) {
	if len(suffix) < 2 || suffix[0] != ':' {
		return 0, 0, suffix == ""
	}
	body := suffix[1:]
	startText, endText, hasRange := strings.Cut(body, "-")
	start, err := strconv.Atoi(startText)
	if err != nil || start <= 0 {
		return 0, 0, false
	}
	if !hasRange {
		return start, start, true
	}
	end, err := strconv.Atoi(endText)
	if err != nil || end <= 0 {
		return 0, 0, false
	}
	return start, end, true
}

func stripReferenceTrailingPunctuation(value string) string {
	stripped := strings.TrimRight(value, ",.;!?")
	for len(stripped) > 0 {
		closer := stripped[len(stripped)-1]
		var opener byte
		switch closer {
		case ')':
			opener = '('
		case ']':
			opener = '['
		case '}':
			opener = '{'
		default:
			return stripped
		}
		if strings.Count(stripped, string(closer)) <= strings.Count(stripped, string(opener)) {
			return stripped
		}
		stripped = stripped[:len(stripped)-1]
	}
	return stripped
}

func stripReferenceWrappers(value string) string {
	if len(value) >= 2 && value[0] == value[len(value)-1] {
		switch value[0] {
		case '`', '"', '\'':
			return value[1 : len(value)-1]
		}
	}
	return value
}

func referenceBlockedByPrefix(message string, at int) bool {
	if at == 0 {
		return false
	}
	prev := message[at-1]
	return isASCIIWord(prev) || prev == '/'
}

func isReferenceSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func isASCIIWord(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
