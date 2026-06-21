// Package deepseekv31 parses DeepSeek V3.1 raw tool-call output.
//
// DeepSeek V3.1 uses fullwidth Unicode special tokens to delimit tool calls:
//
//	<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>name<｜tool▁sep｜>{"arg":"val"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>
//
// Multiple tool calls appear as consecutive <｜tool▁call▁begin｜>...<｜tool▁call▁end｜> blocks.
// The outer <｜tool▁calls▁begin｜>/<｜tool▁calls▁end｜> tokens are optional sentinels.
package deepseekv31

import (
	"encoding/json"
	"strings"
)

const (
	tokBegin    = "<｜tool▁call▁begin｜>"
	tokSep      = "<｜tool▁sep｜>"
	tokEnd      = "<｜tool▁call▁end｜>"
	tokAllBegin = "<｜tool▁calls▁begin｜>"
)

// ParsedToolCall holds one extracted tool call.
type ParsedToolCall struct {
	Name      string
	Arguments string
}

// ContainsToolCallBlock returns true when text has at least one DeepSeek V3.1
// tool-call token pair.
func ContainsToolCallBlock(text string) bool {
	return strings.Contains(text, tokBegin) && strings.Contains(text, tokSep)
}

// ParseBlock extracts all tool calls from text and returns any parse errors per
// malformed block. The remaining prefix before the first tool-call sentinel is
// discarded by callers as content text.
func ParseBlock(text string) (calls []ParsedToolCall, errs []error) {
	for {
		start := strings.Index(text, tokBegin)
		if start < 0 {
			break
		}
		after := text[start+len(tokBegin):]
		sepIdx := strings.Index(after, tokSep)
		if sepIdx < 0 {
			// Incomplete block — degrade.
			errs = append(errs, degradedf("tool call block missing %q separator", tokSep))
			break
		}
		name := strings.TrimSpace(after[:sepIdx])
		rest := after[sepIdx+len(tokSep):]
		endIdx := strings.Index(rest, tokEnd)
		var rawArgs string
		if endIdx < 0 {
			rawArgs = strings.TrimSpace(rest)
			text = ""
		} else {
			rawArgs = strings.TrimSpace(rest[:endIdx])
			text = rest[endIdx+len(tokEnd):]
		}
		// Validate JSON arguments; accept empty / null as empty object.
		normalized := rawArgs
		if normalized == "" || normalized == "null" {
			normalized = "{}"
		}
		if !json.Valid([]byte(normalized)) {
			errs = append(errs, degradedf("deepseek_v3_1: invalid JSON arguments for %q: %q", name, rawArgs))
			if name != "" {
				calls = append(calls, ParsedToolCall{Name: name, Arguments: "{}"})
			}
		} else {
			if name != "" {
				calls = append(calls, ParsedToolCall{Name: name, Arguments: normalized})
			}
		}
	}
	return calls, errs
}
