// Package glm45 parses GLM-4.5 MoE raw tool-call output.
//
// GLM-4.5 MoE emits tool calls as XML with alternating arg_key/arg_value
// children inside an arguments element:
//
//	<tool_call>
//	<name>get_weather</name>
//	<arguments>
//	<arg_key>city</arg_key><arg_value>Paris</arg_value>
//	</arguments>
//	</tool_call>
//
// Multiple tool calls appear as consecutive <tool_call>...</tool_call> blocks.
// The resulting arguments are serialized back to a JSON object.
package glm45

import (
	"encoding/json"
	"strings"
)

// ParsedToolCall holds one extracted tool call.
type ParsedToolCall struct {
	Name      string
	Arguments string
}

// ContainsToolCallBlock returns true when text has a GLM-4.5 tool_call block
// with arg_key/arg_value children.
func ContainsToolCallBlock(text string) bool {
	return strings.Contains(text, "<tool_call>") && strings.Contains(text, "<arg_key>")
}

// ParseBlock extracts all tool calls from GLM-4.5 MoE raw output.
func ParseBlock(text string) (calls []ParsedToolCall, errs []error) {
	for {
		start := strings.Index(text, "<tool_call>")
		if start < 0 {
			break
		}
		end := strings.Index(text[start:], "</tool_call>")
		var block string
		if end < 0 {
			block = text[start+len("<tool_call>"):]
			text = ""
		} else {
			block = text[start+len("<tool_call>") : start+end]
			text = text[start+end+len("</tool_call>"):]
		}

		name := extractTag(block, "name")
		if name == "" {
			errs = append(errs, degradedf("glm45: tool_call block missing <name>"))
			continue
		}

		argsBlock := extractTag(block, "arguments")
		args, err := parseArgKeyValues(argsBlock)
		if err != nil {
			errs = append(errs, err)
			calls = append(calls, ParsedToolCall{Name: name, Arguments: "{}"})
			continue
		}
		calls = append(calls, ParsedToolCall{Name: name, Arguments: args})
	}
	return calls, errs
}

func extractTag(text, tag string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	start := strings.Index(text, open)
	if start < 0 {
		return ""
	}
	rest := text[start+len(open):]
	end := strings.Index(rest, close)
	if end < 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}

// parseArgKeyValues converts alternating <arg_key>/<arg_value> pairs to a
// JSON object. Values are always string-typed in GLM-4.5 raw output.
func parseArgKeyValues(block string) (string, error) {
	params := map[string]string{}
	var order []string
	rest := block
	for {
		ki := strings.Index(rest, "<arg_key>")
		if ki < 0 {
			break
		}
		rest = rest[ki+len("<arg_key>"):]
		ke := strings.Index(rest, "</arg_key>")
		if ke < 0 {
			return "{}", degradedf("glm45: unclosed <arg_key>")
		}
		key := strings.TrimSpace(rest[:ke])
		rest = rest[ke+len("</arg_key>"):]

		vi := strings.Index(rest, "<arg_value>")
		if vi < 0 {
			return "{}", degradedf("glm45: <arg_key> %q missing matching <arg_value>", key)
		}
		rest = rest[vi+len("<arg_value>"):]
		ve := strings.Index(rest, "</arg_value>")
		if ve < 0 {
			return "{}", degradedf("glm45: unclosed <arg_value> for key %q", key)
		}
		val := strings.TrimSpace(rest[:ve])
		rest = rest[ve+len("</arg_value>"):]

		if _, seen := params[key]; !seen {
			order = append(order, key)
		}
		params[key] = val
	}
	if len(params) == 0 {
		return "{}", nil
	}
	obj := make(map[string]string, len(params))
	for k, v := range params {
		obj[k] = v
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return "{}", degradedf("glm45: failed to marshal args: %v", err)
	}
	_ = order
	return string(b), nil
}
