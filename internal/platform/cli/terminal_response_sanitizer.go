package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/input"

const TerminalResponseStrippedEvidence = input.TerminalResponseStrippedEvidence

type TerminalResponseSanitizerMeta = input.TerminalResponseSanitizerMeta

// StripLeakedTerminalResponses removes leaked terminal control responses that
// can appear in operator input after resize, mouse-tracking, or multiplexer
// events. It preserves ordinary text and non-response ANSI sequences.
func StripLeakedTerminalResponses(text string) string {
	return input.StripLeakedTerminalResponses(text)
}

// StripLeakedTerminalResponsesWithMeta strips leaked DSR/CPR and SGR mouse
// report fragments while returning redacted evidence about whether a terminal
// response was removed. The metadata deliberately does not include input text
// or terminal coordinates.
func StripLeakedTerminalResponsesWithMeta(text string) (string, TerminalResponseSanitizerMeta) {
	return input.StripLeakedTerminalResponsesWithMeta(text)
}
