// Package longcat parses Longcat Flash raw tool-call output.
//
// Longcat Flash uses the same XML/JSON-body format as Hermes with
// <tool_call>{"name": "fn", "arguments": {...}}</tool_call> blocks.
// This package delegates to the hermesxml parser.
package longcat

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair/toolcallparsers/hermesxml"
)

// ParsedToolCall is the same as hermesxml.ParsedToolCall.
type ParsedToolCall = hermesxml.ParsedToolCall

// ContainsToolCallBlock returns true when text has a <tool_call> block.
func ContainsToolCallBlock(text string) bool {
	return hermesxml.ContainsToolCallBlock(text)
}

// ParseBlock extracts tool calls from Longcat Flash raw output by delegating
// to the Hermes XML parser which handles the same format.
func ParseBlock(text string) ([]ParsedToolCall, []error) {
	return hermesxml.ParseBlock(text)
}
