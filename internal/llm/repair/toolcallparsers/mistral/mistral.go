// Package mistral parses Mistral raw tool-call output.
//
// Mistral pre-v11 models emit tool calls as a JSON array after a [TOOL_CALLS]
// sentinel token:
//
//	[TOOL_CALLS] [{"name": "get_weather", "arguments": {"city": "Paris"}}]
//
// Mistral v11+ models use native function-calling via the API (structured
// tool_calls field) which does not require raw text parsing. This package
// handles only the pre-v11 text-stream sentinel format.
package mistral

import (
	"encoding/json"
	"strings"
)

const tokenSentinel = "[TOOL_CALLS]"

// ParsedToolCall holds one extracted tool call.
type ParsedToolCall struct {
	Name      string
	Arguments string
}

// ContainsToolCallBlock returns true when text has the Mistral pre-v11
// [TOOL_CALLS] sentinel.
func ContainsToolCallBlock(text string) bool {
	return strings.Contains(text, tokenSentinel)
}

type rawCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ParseBlock extracts all tool calls from Mistral pre-v11 raw output.
// Returns any parse errors for malformed payloads.
func ParseBlock(text string) (calls []ParsedToolCall, errs []error) {
	idx := strings.Index(text, tokenSentinel)
	if idx < 0 {
		return nil, nil
	}
	payload := strings.TrimSpace(text[idx+len(tokenSentinel):])
	if payload == "" {
		errs = append(errs, degradedf("mistral: [TOOL_CALLS] sentinel with empty payload"))
		return calls, errs
	}
	var rawCalls []rawCall
	if err := json.Unmarshal([]byte(payload), &rawCalls); err != nil {
		errs = append(errs, degradedf("mistral: failed to parse tool calls array: %v", err))
		return calls, errs
	}
	for _, rc := range rawCalls {
		if rc.Name == "" {
			errs = append(errs, degradedf("mistral: tool call missing name"))
			continue
		}
		args := "{}"
		if len(rc.Arguments) > 0 && string(rc.Arguments) != "null" {
			if !json.Valid(rc.Arguments) {
				errs = append(errs, degradedf("mistral: invalid JSON arguments for %q", rc.Name))
			} else {
				args = string(rc.Arguments)
			}
		}
		calls = append(calls, ParsedToolCall{Name: rc.Name, Arguments: args})
	}
	return calls, errs
}
