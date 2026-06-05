package jsonvalue

import (
	"encoding/json"
	"strings"
)

// CloneRaw returns an independent copy of raw JSON bytes so normalized MCP
// contracts never alias caller-owned decoder buffers or package-level defaults.
func CloneRaw(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

// NullishRaw reports whether raw JSON is absent or explicitly null after
// trimming protocol whitespace.
func NullishRaw(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed == "" || trimmed == "null"
}

// CloneMap returns an independent copy of a JSON-like object graph. It handles
// both generic JSON decoder shapes and typed map/slice literals used by tests
// or in-process tool declarations.
func CloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = Clone(v)
	}
	return out
}

// Clone returns an independent copy of a JSON-like value graph.
func Clone(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return CloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = Clone(item)
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(typed))
		for i, item := range typed {
			out[i] = CloneMap(item)
		}
		return out
	case json.RawMessage:
		return CloneRaw(typed)
	case []byte:
		return append([]byte(nil), typed...)
	default:
		return typed
	}
}
