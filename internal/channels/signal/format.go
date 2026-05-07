package signal

import (
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

type SignalTextStyle string

const (
	SignalStyleBold          SignalTextStyle = "BOLD"
	SignalStyleItalic        SignalTextStyle = "ITALIC"
	SignalStyleStrikethrough SignalTextStyle = "STRIKETHROUGH"
	SignalStyleMonospace     SignalTextStyle = "MONOSPACE"
)

// SignalBodyRange mirrors Signal's bodyRanges textStyle shape. Start and
// Length are UTF-16 code units, not bytes or runes.
type SignalBodyRange struct {
	Start  int             `json:"start"`
	Length int             `json:"length"`
	Style  SignalTextStyle `json:"style"`
}

// MarkdownToSignal converts the Hermes-supported Markdown subset to Signal's
// plain text plus bodyRanges. It intentionally avoids broad Markdown parsing so
// Signal-specific false-positive guards stay obvious and testable.
func MarkdownToSignal(text string) (string, []SignalBodyRange) {
	if text == "" {
		return "", nil
	}
	var out signalMarkdownBuilder
	inCodeBlock := false
	for _, raw := range splitSignalLines(text) {
		line, newline := splitSignalLineEnding(raw)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			out.appendStyled(line, SignalStyleMonospace)
			out.appendPlain(newline)
			continue
		}
		if heading, ok := signalMarkdownHeading(line); ok {
			out.appendStyled(heading, SignalStyleBold)
			out.appendPlain(newline)
			continue
		}
		parseSignalInlineMarkdown(line, &out)
		out.appendPlain(newline)
	}
	return out.String(), out.ranges
}

type signalMarkdownBuilder struct {
	b      strings.Builder
	pos    int
	ranges []SignalBodyRange
}

func (b *signalMarkdownBuilder) String() string {
	return b.b.String()
}

func (b *signalMarkdownBuilder) appendPlain(text string) {
	if text == "" {
		return
	}
	b.b.WriteString(text)
	b.pos += len(signalUTF16Units(text))
}

func (b *signalMarkdownBuilder) appendStyled(text string, style SignalTextStyle) {
	if text == "" {
		return
	}
	start := b.pos
	b.appendPlain(text)
	b.ranges = append(b.ranges, SignalBodyRange{
		Start:  start,
		Length: b.pos - start,
		Style:  style,
	})
}

func splitSignalLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.SplitAfter(text, "\n")
}

func splitSignalLineEnding(line string) (string, string) {
	if strings.HasSuffix(line, "\n") {
		return strings.TrimSuffix(line, "\n"), "\n"
	}
	return line, ""
}

func signalMarkdownHeading(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	for level := 6; level >= 1; level-- {
		prefix := strings.Repeat("#", level) + " "
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)), true
		}
	}
	return "", false
}

func parseSignalInlineMarkdown(line string, out *signalMarkdownBuilder) {
	for i := 0; i < len(line); {
		switch {
		case strings.HasPrefix(line[i:], "`"):
			if end := strings.Index(line[i+1:], "`"); end >= 0 {
				close := i + 1 + end
				out.appendStyled(line[i+1:close], SignalStyleMonospace)
				i = close + 1
				continue
			}
		case strings.HasPrefix(line[i:], "**"):
			if signalCanOpenStyle(line, i, "**") {
				if close := findSignalClosingMarker(line, i+2, "**"); close >= 0 {
					out.appendStyled(line[i+2:close], SignalStyleBold)
					i = close + 2
					continue
				}
			}
		case strings.HasPrefix(line[i:], "__"):
			if signalCanOpenStyle(line, i, "__") {
				if close := findSignalClosingMarker(line, i+2, "__"); close >= 0 {
					out.appendStyled(line[i+2:close], SignalStyleBold)
					i = close + 2
					continue
				}
			}
		case strings.HasPrefix(line[i:], "~~"):
			if signalCanOpenStyle(line, i, "~~") {
				if close := findSignalClosingMarker(line, i+2, "~~"); close >= 0 {
					out.appendStyled(line[i+2:close], SignalStyleStrikethrough)
					i = close + 2
					continue
				}
			}
		case line[i] == '*':
			if !strings.HasPrefix(line[i:], "**") && signalCanOpenStyle(line, i, "*") {
				if close := findSignalClosingMarker(line, i+1, "*"); close >= 0 {
					out.appendStyled(line[i+1:close], SignalStyleItalic)
					i = close + 1
					continue
				}
			}
		case line[i] == '_':
			if !strings.HasPrefix(line[i:], "__") && signalCanOpenStyle(line, i, "_") {
				if close := findSignalClosingMarker(line, i+1, "_"); close >= 0 {
					out.appendStyled(line[i+1:close], SignalStyleItalic)
					i = close + 1
					continue
				}
			}
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError && size == 0 {
			break
		}
		out.appendPlain(line[i : i+size])
		i += size
	}
}

func findSignalClosingMarker(line string, start int, marker string) int {
	for i := start; i < len(line); {
		idx := strings.Index(line[i:], marker)
		if idx < 0 {
			return -1
		}
		close := i + idx
		if marker == "*" && strings.HasPrefix(line[close:], "**") {
			i = close + 2
			continue
		}
		if marker == "_" && strings.HasPrefix(line[close:], "__") {
			i = close + 2
			continue
		}
		if close > start && signalCanCloseStyle(line, close, marker) {
			return close
		}
		i = close + len(marker)
	}
	return -1
}

func signalCanOpenStyle(line string, pos int, marker string) bool {
	next, ok := runeAfter(line, pos+len(marker))
	if !ok || unicode.IsSpace(next) {
		return false
	}
	if marker == "*" && pos == 0 && next == ' ' {
		return false
	}
	if marker == "*" && pos == 0 && unicode.IsSpace(next) {
		return false
	}
	if marker == "*" && pos == 0 && next == '-' {
		return false
	}
	if marker == "_" || marker == "__" {
		if prev, ok := runeBefore(line, pos); ok && signalWordRune(prev) {
			return false
		}
		if signalWordRune(next) {
			return true
		}
	}
	if marker == "*" && pos == 0 && next == ' ' {
		return false
	}
	return true
}

func signalCanCloseStyle(line string, pos int, marker string) bool {
	prev, ok := runeBefore(line, pos)
	if !ok || unicode.IsSpace(prev) {
		return false
	}
	if marker == "_" || marker == "__" {
		if next, ok := runeAfter(line, pos+len(marker)); ok && signalWordRune(next) {
			return false
		}
	}
	return true
}

func signalWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func runeBefore(s string, pos int) (rune, bool) {
	if pos <= 0 || pos > len(s) {
		return 0, false
	}
	r, _ := utf8.DecodeLastRuneInString(s[:pos])
	return r, r != utf8.RuneError
}

func runeAfter(s string, pos int) (rune, bool) {
	if pos < 0 || pos >= len(s) {
		return 0, false
	}
	r, _ := utf8.DecodeRuneInString(s[pos:])
	return r, r != utf8.RuneError
}

func signalUTF16Units(text string) []uint16 {
	return utf16.Encode([]rune(text))
}

func signalRunesFromUTF16(units []uint16) []rune {
	return utf16.Decode(units)
}
