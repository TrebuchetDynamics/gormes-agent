// Package qwen parses raw tool-call blocks in the Qwen 2.5 / Qwen3 family
// format. Qwen uses the same tool_call_xml_json_body input style as the
// Hermes NousChatML format:
//
//	<tool_call>
//	{"name": "<tool_name>", "arguments": {...}}
//	</tool_call>
//
// This package delegates to the hermesxml parser and re-exports the
// ParsedToolCall type so callers can use a Qwen-specific import path while
// sharing the same parser logic. This matches the upstream manifest entry
// in internal/llm/repair/parsers/tool_call_parser_manifest.go where both
// hermes and qwen share ExpectedInputStyle "tool_call_xml_json_body".
package qwen

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair/toolcallparsers/hermesxml"
)

// ParsedToolCall is one extracted tool call. Alias of hermesxml.ParsedToolCall
// so callers importing this package get a stable type without an indirect dep.
type ParsedToolCall = hermesxml.ParsedToolCall

// ParseBlock extracts all <tool_call>…</tool_call> blocks from text and
// parses the JSON body of each, following the Qwen 2.5/Qwen3 format which
// is identical to the Hermes NousChatML tool_call_xml_json_body style.
func ParseBlock(text string) (calls []ParsedToolCall, errs []error) {
	return hermesxml.ParseBlock(text)
}

// ContainsToolCallBlock reports whether text contains at least one
// (possibly malformed) <tool_call> opening tag.
func ContainsToolCallBlock(text string) bool {
	return hermesxml.ContainsToolCallBlock(text)
}
