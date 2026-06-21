// Package qwen3coder parses Qwen3-Coder raw tool-call output.
//
// Qwen3-Coder emits tool calls in an XML function/parameter style:
//
//	<function=get_weather>
//	<parameter=city>Paris</parameter>
//	</function>
//
// Multiple tool calls appear as consecutive <function=...>...</function> blocks.
// Parameter values are always string-typed; the result is serialized as JSON.
package qwen3coder

import (
	"encoding/json"
	"strings"
)

// ParsedToolCall holds one extracted tool call.
type ParsedToolCall struct {
	Name      string
	Arguments string
}

// ContainsToolCallBlock returns true when text has a Qwen3-Coder function block.
func ContainsToolCallBlock(text string) bool {
	return strings.Contains(text, "<function=") && strings.Contains(text, "<parameter=")
}

// ParseBlock extracts all tool calls from Qwen3-Coder raw output.
func ParseBlock(text string) (calls []ParsedToolCall, errs []error) {
	for {
		start := strings.Index(text, "<function=")
		if start < 0 {
			break
		}
		// Extract function name from <function=name>
		nameStart := start + len("<function=")
		nameEnd := strings.Index(text[nameStart:], ">")
		if nameEnd < 0 {
			errs = append(errs, degradedf("qwen3coder: unclosed <function= tag"))
			break
		}
		name := strings.TrimSpace(text[nameStart : nameStart+nameEnd])
		blockStart := nameStart + nameEnd + 1
		closeTag := "</function>"
		closeIdx := strings.Index(text[blockStart:], closeTag)
		var block string
		if closeIdx < 0 {
			block = text[blockStart:]
			text = ""
		} else {
			block = text[blockStart : blockStart+closeIdx]
			text = text[blockStart+closeIdx+len(closeTag):]
		}
		args, err := parseParameters(block)
		if err != nil {
			errs = append(errs, err)
			if name != "" {
				calls = append(calls, ParsedToolCall{Name: name, Arguments: "{}"})
			}
			continue
		}
		if name != "" {
			calls = append(calls, ParsedToolCall{Name: name, Arguments: args})
		}
	}
	return calls, errs
}

// parseParameters extracts <parameter=key>value</parameter> pairs and
// serializes them to a JSON object.
func parseParameters(block string) (string, error) {
	params := map[string]string{}
	rest := block
	for {
		pi := strings.Index(rest, "<parameter=")
		if pi < 0 {
			break
		}
		keyStart := pi + len("<parameter=")
		keyEnd := strings.Index(rest[keyStart:], ">")
		if keyEnd < 0 {
			return "{}", degradedf("qwen3coder: unclosed <parameter= tag")
		}
		key := strings.TrimSpace(rest[keyStart : keyStart+keyEnd])
		valStart := keyStart + keyEnd + 1
		closeParam := "</parameter>"
		valEnd := strings.Index(rest[valStart:], closeParam)
		var val string
		if valEnd < 0 {
			val = strings.TrimSpace(rest[valStart:])
			rest = ""
		} else {
			val = strings.TrimSpace(rest[valStart : valStart+valEnd])
			rest = rest[valStart+valEnd+len(closeParam):]
		}
		params[key] = val
	}
	if len(params) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(params)
	if err != nil {
		return "{}", degradedf("qwen3coder: failed to marshal parameters: %v", err)
	}
	return string(b), nil
}
