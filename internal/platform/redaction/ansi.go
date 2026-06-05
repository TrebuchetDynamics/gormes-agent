package redaction

import "regexp"

var (
	ansiEscapePattern   = regexp.MustCompile(`\x1b(?:\[[\x30-\x3f]*[\x20-\x2f]*[\x40-\x7e]|\][\s\S]*?(?:\x07|\x1b\\)|[PX^_][\s\S]*?(?:\x1b\\)|[\x20-\x2f]+[\x30-\x7e]|[\x30-\x7e])|\x{9b}[\x30-\x3f]*[\x20-\x2f]*[\x40-\x7e]|\x{9d}[\s\S]*?(?:\x07|\x{9c})|[\x{80}-\x{9f}]`)
	ansiFastPathPattern = regexp.MustCompile(`[\x1b\x{80}-\x{9f}]`)
)

// StripANSI removes ECMA-48 terminal escape/control sequences from text before
// tool output reaches model context or artifacts. Ported from Hermes'
// tools/ansi_strip.py behavior, adapted to Go regexp syntax.
func StripANSI(text string) string {
	if text == "" || !ansiFastPathPattern.MatchString(text) {
		return text
	}
	return ansiEscapePattern.ReplaceAllString(text, "")
}
