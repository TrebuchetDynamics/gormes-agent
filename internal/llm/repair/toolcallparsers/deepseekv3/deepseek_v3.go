// Package deepseekv3 parses DeepSeek V3 raw tool-call output.
//
// DeepSeek V3 uses the same fullwidth Unicode token structure as V3.1 but
// wraps the JSON arguments in a markdown fenced code block:
//
//	<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>name<｜tool▁sep｜>```json
//	{"arg":"val"}
//	```<｜tool▁call▁end｜>
//
// The code-fence markers (```json / ```) are stripped before JSON parsing.
package deepseekv3

import (
	"encoding/json"
	"strings"
)

const (
	tokBegin = "<｜tool▁call▁begin｜>"
	tokSep   = "<｜tool▁sep｜>"
	tokEnd   = "<｜tool▁call▁end｜>"
)

// ParsedToolCall holds one extracted tool call.
type ParsedToolCall struct {
	Name      string
	Arguments string
}

// ContainsToolCallBlock returns true when text contains DeepSeek V3 markers.
func ContainsToolCallBlock(text string) bool {
	return strings.Contains(text, tokBegin) && strings.Contains(text, tokSep)
}

// ParseBlock extracts tool calls from DeepSeek V3 raw output.
func ParseBlock(text string) (calls []ParsedToolCall, errs []error) {
	for {
		start := strings.Index(text, tokBegin)
		if start < 0 {
			break
		}
		after := text[start+len(tokBegin):]
		sepIdx := strings.Index(after, tokSep)
		if sepIdx < 0 {
			errs = append(errs, degradedf("deepseek_v3: tool call block missing %q separator", tokSep))
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
		rawArgs = stripCodeFence(rawArgs)
		normalized := rawArgs
		if normalized == "" || normalized == "null" {
			normalized = "{}"
		}
		if !json.Valid([]byte(normalized)) {
			errs = append(errs, degradedf("deepseek_v3: invalid JSON arguments for %q: %q", name, rawArgs))
			if name != "" {
				calls = append(calls, ParsedToolCall{Name: name, Arguments: "{}"})
			}
		} else if name != "" {
			calls = append(calls, ParsedToolCall{Name: name, Arguments: normalized})
		}
	}
	return calls, errs
}

// stripCodeFence removes optional ```json / ``` markdown fencing.
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	for _, prefix := range []string{"```json", "```"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimPrefix(s, prefix)
			s = strings.TrimSuffix(strings.TrimSpace(s), "```")
			return strings.TrimSpace(s)
		}
	}
	return s
}
