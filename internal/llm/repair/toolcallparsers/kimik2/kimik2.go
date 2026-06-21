// Package kimik2 parses Kimi K2 raw tool-call output.
//
// Kimi K2 uses a section begin/end pair that wraps a JSON array of tool calls:
//
//	<|tool_calls_section_begin|>
//	[{"name": "get_weather", "arguments": {"city": "Paris"}}]
//	<|tool_calls_section_end|>
//
// The JSON array uses "name" and "arguments" keys matching OpenAI convention.
package kimik2

import (
	"encoding/json"
	"strings"
)

const (
	tokBegin = "<|tool_calls_section_begin|>"
	tokEnd   = "<|tool_calls_section_end|>"
)

// ParsedToolCall holds one extracted tool call.
type ParsedToolCall struct {
	Name      string
	Arguments string
}

// ContainsToolCallBlock returns true when text has Kimi K2 section markers.
func ContainsToolCallBlock(text string) bool {
	return strings.Contains(text, tokBegin)
}

type rawCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ParseBlock extracts all tool calls from Kimi K2 raw output.
func ParseBlock(text string) (calls []ParsedToolCall, errs []error) {
	start := strings.Index(text, tokBegin)
	if start < 0 {
		return nil, nil
	}
	payload := text[start+len(tokBegin):]
	if end := strings.Index(payload, tokEnd); end >= 0 {
		payload = payload[:end]
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		errs = append(errs, degradedf("kimik2: section begin/end with empty payload"))
		return calls, errs
	}
	var rawCalls []rawCall
	if err := json.Unmarshal([]byte(payload), &rawCalls); err != nil {
		errs = append(errs, degradedf("kimik2: failed to parse tool calls array: %v", err))
		return calls, errs
	}
	for _, rc := range rawCalls {
		if rc.Name == "" {
			errs = append(errs, degradedf("kimik2: tool call missing name"))
			continue
		}
		args := "{}"
		if len(rc.Arguments) > 0 && string(rc.Arguments) != "null" {
			if !json.Valid(rc.Arguments) {
				errs = append(errs, degradedf("kimik2: invalid JSON arguments for %q", rc.Name))
			} else {
				args = string(rc.Arguments)
			}
		}
		calls = append(calls, ParsedToolCall{Name: rc.Name, Arguments: args})
	}
	return calls, errs
}
