package input

import (
	"regexp"
	"strings"
)

const TerminalResponseStrippedEvidence = "terminal_response_stripped"

var (
	dsrCPREscapePattern     = regexp.MustCompile(`\x1b\[\d+;\d+R`)
	dsrCPRVisiblePattern    = regexp.MustCompile(`\^\[\[\d+;\d+R`)
	sgrMouseEscapePattern   = regexp.MustCompile(`\x1b\[<\d+;\d+;\d+[Mm]`)
	sgrMouseVisiblePattern  = regexp.MustCompile(`\^\[\[<\d+;\d+;\d+[Mm]`)
	sgrMouseBareFormPattern = regexp.MustCompile(`<\d+;\d+;\d+[Mm]`)
)

type TerminalResponseSanitizerMeta struct {
	Stripped             bool
	MouseReportsStripped bool
	Evidence             string
}

// StripLeakedTerminalResponses removes leaked terminal control responses that
// can appear in operator input after resize, mouse-tracking, or multiplexer
// events. It preserves ordinary text and non-response ANSI sequences.
func StripLeakedTerminalResponses(text string) string {
	cleaned, _ := StripLeakedTerminalResponsesWithMeta(text)
	return cleaned
}

// StripLeakedTerminalResponsesWithMeta strips leaked DSR/CPR and SGR mouse
// report fragments while returning redacted evidence about whether a terminal
// response was removed. The metadata deliberately does not include input text
// or terminal coordinates.
func StripLeakedTerminalResponsesWithMeta(text string) (string, TerminalResponseSanitizerMeta) {
	if text == "" {
		return text, TerminalResponseSanitizerMeta{}
	}

	hasEscape := containsTerminalResponsePrefix(text, "\x1b[")
	hasVisible := containsTerminalResponsePrefix(text, "^[")
	hasBareMouse := containsBareMouseShape(text)
	if !hasEscape && !hasVisible && !hasBareMouse {
		return text, TerminalResponseSanitizerMeta{}
	}

	var meta TerminalResponseSanitizerMeta

	if hasEscape {
		text, meta.Stripped = stripPattern(text, dsrCPREscapePattern, meta.Stripped)
		var strippedMouse bool
		text, strippedMouse = stripPattern(text, sgrMouseEscapePattern, false)
		meta.MouseReportsStripped = meta.MouseReportsStripped || strippedMouse
		meta.Stripped = meta.Stripped || strippedMouse
	}

	if hasVisible {
		text, meta.Stripped = stripPattern(text, dsrCPRVisiblePattern, meta.Stripped)
		var strippedMouse bool
		text, strippedMouse = stripPattern(text, sgrMouseVisiblePattern, false)
		meta.MouseReportsStripped = meta.MouseReportsStripped || strippedMouse
		meta.Stripped = meta.Stripped || strippedMouse
	}

	if hasBareMouse {
		var strippedMouse bool
		text, strippedMouse = stripPattern(text, sgrMouseBareFormPattern, false)
		meta.MouseReportsStripped = meta.MouseReportsStripped || strippedMouse
		meta.Stripped = meta.Stripped || strippedMouse
	}

	if meta.Stripped {
		meta.Evidence = TerminalResponseStrippedEvidence
	}
	return text, meta
}

func stripPattern(text string, pattern *regexp.Regexp, alreadyStripped bool) (string, bool) {
	if !pattern.MatchString(text) {
		return text, alreadyStripped
	}
	return pattern.ReplaceAllString(text, ""), true
}

func containsTerminalResponsePrefix(text, prefix string) bool {
	return strings.Contains(text, prefix)
}

func containsBareMouseShape(text string) bool {
	return strings.Contains(text, "<") &&
		strings.Contains(text, ";") &&
		(strings.Contains(text, "M") || strings.Contains(text, "m"))
}
