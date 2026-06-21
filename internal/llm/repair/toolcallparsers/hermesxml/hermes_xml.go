// Package hermesxml parses raw tool-call blocks in the Hermes/NousChatML format:
//
//	<tool_call>
//	{"name": "<tool_name>", "arguments": {...}}
//	</tool_call>
//
// This matches the `tool_call_xml_json_body` input style from the upstream
// Hermes tools/hermes_parser.py and is used by NousResearch Hermes-3 family
// models when the model emits tool calls as raw text rather than via the
// structured tool-call API.
package hermesxml

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	openTag  = "<tool_call>"
	closeTag = "</tool_call>"
)

// ParsedToolCall is one extracted tool call from a Hermes XML block.
type ParsedToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ParseBlock extracts all <tool_call>...</tool_call> blocks from text and
// parses the JSON body of each. Returns one ParsedToolCall per well-formed
// block. Malformed JSON or blocks with no recognisable name are returned in
// errs; the caller decides whether to surface them.
func ParseBlock(text string) (calls []ParsedToolCall, errs []error) {
	remaining := text
	for {
		start := strings.Index(remaining, openTag)
		if start < 0 {
			break
		}
		after := remaining[start+len(openTag):]
		end := strings.Index(after, closeTag)
		if end < 0 {
			// Unclosed tag — stop scanning.
			errs = append(errs, fmt.Errorf("unclosed %s block", openTag))
			break
		}
		body := strings.TrimSpace(after[:end])
		remaining = after[end+len(closeTag):]

		call, err := parseBody(body)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		calls = append(calls, call)
	}
	return calls, errs
}

// parseBody unmarshals the JSON inside one <tool_call>…</tool_call> block.
// Hermes models sometimes emit single-quoted JSON; we normalise it first.
func parseBody(body string) (ParsedToolCall, error) {
	// Some Hermes variants emit Python-dict-style single quotes.
	normalised := normaliseSingleQuotes(body)

	var raw struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(normalised), &raw); err != nil {
		// Second chance: try the original body before normalisation.
		if err2 := json.Unmarshal([]byte(body), &raw); err2 != nil {
			return ParsedToolCall{}, fmt.Errorf("hermes_xml: JSON parse error: %w (body: %.80s)", err, body)
		}
	}
	if raw.Name == "" {
		return ParsedToolCall{}, fmt.Errorf("hermes_xml: tool call has empty name (body: %.80s)", body)
	}
	// Default to empty object when arguments are absent.
	if len(raw.Arguments) == 0 || string(raw.Arguments) == "null" {
		raw.Arguments = json.RawMessage("{}")
	}
	return ParsedToolCall{Name: raw.Name, Arguments: raw.Arguments}, nil
}

// normaliseSingleQuotes performs a best-effort conversion of Python-style
// single-quoted keys/values to double-quoted JSON. It handles the common
// cases emitted by Hermes models and is intentionally conservative: it only
// replaces simple unescaped single-quoted tokens, leaving complex strings
// (containing escaped chars or nested quotes) untouched.
func normaliseSingleQuotes(s string) string {
	if !strings.ContainsRune(s, '\'') {
		return s
	}
	var sb strings.Builder
	inDouble := false
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		switch {
		case ch == '"' && (i == 0 || runes[i-1] != '\\'):
			inDouble = !inDouble
			sb.WriteRune(ch)
		case ch == '\'' && !inDouble:
			sb.WriteRune('"')
		default:
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}

// ContainsToolCallBlock reports whether text contains at least one
// (possibly malformed) <tool_call> opening tag.
func ContainsToolCallBlock(text string) bool {
	return strings.Contains(text, openTag)
}
