package ansitext

import "strings"

const ansiESC = byte(0x1b)

// StripForTUI removes terminal control sequences before text enters
// cursor/source-of-truth calculations.
func StripForTUI(s string) string {
	return sanitize(s, false)
}

// SanitizeForRender keeps SGR color sequences but strips cursor movement,
// OSC strings, malformed CSI, and C0 controls that can corrupt renderer state.
func SanitizeForRender(s string) string {
	return sanitize(s, true)
}

func HasANSI(s string) bool {
	return strings.Contains(s, "\x1b")
}

func sanitize(s string, keepSGR bool) string {
	var out strings.Builder
	out.Grow(len(s))
	for i := 0; i < len(s); {
		b := s[i]
		if b == ansiESC {
			next, keepStart, keepEnd := consume(s, i)
			if keepSGR && keepStart >= 0 {
				out.WriteString(s[keepStart:keepEnd])
			}
			i = next
			continue
		}
		if isStrippedControl(b) {
			i++
			continue
		}
		out.WriteByte(b)
		i++
	}
	return out.String()
}

func consume(s string, start int) (next int, keepStart int, keepEnd int) {
	keepStart = -1
	if start+1 >= len(s) {
		return len(s), keepStart, keepEnd
	}
	switch s[start+1] {
	case '[':
		return consumeCSI(s, start, keepStart, keepEnd)
	case ']', 'P', 'X', '^', '_':
		return consumeStringControl(s, start), keepStart, keepEnd
	default:
		return consumeNonCSI(s, start), keepStart, keepEnd
	}
}

func consumeCSI(s string, start int, keepStart int, keepEnd int) (int, int, int) {
	for i := start + 2; i < len(s); i++ {
		b := s[i]
		if b == ansiESC || b == '\n' {
			return i, keepStart, keepEnd
		}
		if b >= 0x40 && b <= 0x7e {
			end := i + 1
			if b == 'm' {
				return end, start, end
			}
			return end, keepStart, keepEnd
		}
	}
	return len(s), keepStart, keepEnd
}

func consumeStringControl(s string, start int) int {
	for i := start + 2; i < len(s); i++ {
		if s[i] == '\a' {
			return i + 1
		}
		if s[i] == ansiESC && i+1 < len(s) && s[i+1] == '\\' {
			return i + 2
		}
	}
	return len(s)
}

func consumeNonCSI(s string, start int) int {
	i := start + 1
	for i < len(s) && s[i] >= 0x20 && s[i] <= 0x2f {
		i++
	}
	if i < len(s) && s[i] >= 0x30 && s[i] <= 0x7e {
		return i + 1
	}
	if i < len(s) {
		return i + 1
	}
	return i
}

func isStrippedControl(b byte) bool {
	return b <= 0x08 ||
		b == 0x0b ||
		b == 0x0c ||
		b == 0x0d ||
		(b >= 0x0e && b <= 0x1a) ||
		(b >= 0x1c && b <= 0x1f) ||
		b == 0x7f
}
