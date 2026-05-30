package jsonfields

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// PreserveKnownZeroFunc reports whether a known field should still be carried
// in an Extra field map because its explicit zero value is semantically
// meaningful for round-tripping older progress files.
type PreserveKnownZeroFunc func(string, json.RawMessage) bool

// MarshalNoEscape marshals v with HTML escaping disabled and no trailing
// newline, matching the canonical progress JSON encoding helpers.
func MarshalNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// MarshalObjectWithExtra merges preserved unknown fields back into a JSON
// object body. Known fields already present in body win; extra keys are emitted
// in deterministic lexical order.
func MarshalObjectWithExtra(label string, body []byte, extra map[string]json.RawMessage) ([]byte, error) {
	if len(extra) == 0 {
		return body, nil
	}
	var known map[string]json.RawMessage
	if err := json.Unmarshal(body, &known); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(extra))
	for key, value := range extra {
		if _, exists := known[key]; exists {
			continue
		}
		value = bytes.TrimSpace(value)
		if len(value) == 0 {
			continue
		}
		if !json.Valid(value) {
			return nil, fmt.Errorf("progress %s extra field %q is not valid JSON", label, key)
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return body, nil
	}
	sort.Strings(keys)

	trimmed := bytes.TrimSpace(body)
	if !bytes.HasPrefix(trimmed, []byte("{")) || !bytes.HasSuffix(trimmed, []byte("}")) {
		return nil, fmt.Errorf("progress %s marshaled to non-object JSON", label)
	}
	var out bytes.Buffer
	out.Write(trimmed[:len(trimmed)-1])
	needsComma := len(trimmed) > 2
	for _, key := range keys {
		if needsComma {
			out.WriteByte(',')
		}
		needsComma = true
		quoted, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		out.Write(quoted)
		out.WriteByte(':')
		out.Write(bytes.TrimSpace(extra[key]))
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

// UnknownObjectFields returns unknown fields from data and any known fields
// whose explicit zero values must be preserved.
func UnknownObjectFields(data []byte, knownFields map[string]bool, preserveKnownZero PreserveKnownZeroFunc) (map[string]json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	extra := map[string]json.RawMessage{}
	for key, value := range raw {
		if !knownFields[key] || (preserveKnownZero != nil && preserveKnownZero(key, value)) {
			extra[key] = append(json.RawMessage(nil), value...)
		}
	}
	if len(extra) == 0 {
		return nil, nil
	}
	return extra, nil
}
