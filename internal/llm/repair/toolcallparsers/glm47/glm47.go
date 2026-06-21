// Package glm47 parses GLM-4.7 raw tool-call output.
//
// GLM-4.7 uses the same arg_key/arg_value XML format as GLM-4.5 MoE but is
// more tolerant of extra whitespace and newlines between XML elements. This
// package delegates to the glm45 parser with the same block extraction logic.
package glm47

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair/toolcallparsers/glm45"
)

// ParsedToolCall is the same as glm45.ParsedToolCall.
type ParsedToolCall = glm45.ParsedToolCall

// ContainsToolCallBlock returns true when text has a GLM-4.7 tool_call block.
// GLM-4.7 uses the same sentinel as GLM-4.5.
func ContainsToolCallBlock(text string) bool {
	return glm45.ContainsToolCallBlock(text)
}

// ParseBlock extracts tool calls from GLM-4.7 raw output. GLM-4.7 uses the
// same arg_key/arg_value format as GLM-4.5 with newline tolerance; the
// underlying parser already handles whitespace-padded values.
func ParseBlock(text string) ([]ParsedToolCall, []error) {
	return glm45.ParseBlock(text)
}
