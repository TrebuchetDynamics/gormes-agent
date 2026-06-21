// Package llama parses Llama 3.x raw tool-call output.
//
// Llama 3.x models emit tool calls as a raw JSON object, optionally prefixed
// with the <|python_tag|> token:
//
//	<|python_tag|>{"name": "get_weather", "parameters": {"city": "Paris"}}
//	{"name": "get_weather", "parameters": {"city": "Paris"}}
//
// The "parameters" key (not "arguments") is the Llama convention. Both keys
// are accepted for maximum compatibility. Only one tool call per model turn
// is typical for Llama 3.x function-calling.
package llama

import (
	"encoding/json"
	"strings"
)

const pythonTag = "<|python_tag|>"

// ParsedToolCall holds one extracted tool call.
type ParsedToolCall struct {
	Name      string
	Arguments string
}

// ContainsToolCallBlock returns true when text has a <|python_tag|> sentinel or
// looks like a bare JSON tool call (starts with '{' and has a "name" key).
func ContainsToolCallBlock(text string) bool {
	t := strings.TrimSpace(text)
	if strings.HasPrefix(t, pythonTag) {
		return true
	}
	// Cheap heuristic: starts with { and contains "name"
	return strings.HasPrefix(t, "{") && strings.Contains(t, `"name"`)
}

type rawCall struct {
	Name       string          `json:"name"`
	Parameters json.RawMessage `json:"parameters"`
	Arguments  json.RawMessage `json:"arguments"`
}

// ParseBlock extracts tool calls from Llama raw output.
func ParseBlock(text string) (calls []ParsedToolCall, errs []error) {
	t := strings.TrimSpace(text)
	if strings.HasPrefix(t, pythonTag) {
		t = strings.TrimSpace(t[len(pythonTag):])
	}
	// Find the JSON object boundary.
	start := strings.Index(t, "{")
	if start < 0 {
		errs = append(errs, degradedf("llama: no JSON object found in output"))
		return calls, errs
	}
	t = t[start:]
	var rc rawCall
	if err := json.Unmarshal([]byte(t), &rc); err != nil {
		errs = append(errs, degradedf("llama: failed to parse tool call object: %v", err))
		return calls, errs
	}
	if rc.Name == "" {
		errs = append(errs, degradedf("llama: tool call object missing 'name' field"))
		return calls, errs
	}
	args := rc.Arguments
	if len(args) == 0 || string(args) == "null" {
		args = rc.Parameters
	}
	normalized := "{}"
	if len(args) > 0 && string(args) != "null" {
		if !json.Valid(args) {
			errs = append(errs, degradedf("llama: invalid JSON in arguments for %q", rc.Name))
		} else {
			normalized = string(args)
		}
	}
	calls = append(calls, ParsedToolCall{Name: rc.Name, Arguments: normalized})
	return calls, errs
}
