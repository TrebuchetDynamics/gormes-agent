package llm

import (
	"bytes"
	"encoding/json"
	"strings"
)

var moonshotSchemaMapKeys = map[string]struct{}{
	"properties":        {},
	"patternProperties": {},
	"$defs":             {},
	"definitions":       {},
}

var moonshotSchemaListKeys = map[string]struct{}{
	"anyOf":       {},
	"oneOf":       {},
	"allOf":       {},
	"prefixItems": {},
}

var moonshotSchemaNodeKeys = map[string]struct{}{
	"items":                {},
	"contains":             {},
	"not":                  {},
	"additionalProperties": {},
	"propertyNames":        {},
}

// IsMoonshotModel reports whether a model slug targets Kimi/Moonshot, including
// aggregator-prefixed slugs such as openrouter/moonshotai/kimi-k2.
func IsMoonshotModel(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return false
	}
	tail := normalized
	if idx := strings.LastIndex(tail, "/"); idx >= 0 {
		tail = tail[idx+1:]
	}
	return strings.HasPrefix(tail, "kimi-") ||
		tail == "kimi" ||
		strings.Contains(normalized, "moonshot") ||
		strings.Contains(normalized, "/kimi") ||
		strings.HasPrefix(normalized, "kimi")
}

// SanitizeToolSchemaForModel applies provider-specific tool-parameter schema
// repairs when a model family needs stricter request shaping.
func SanitizeToolSchemaForModel(model string, raw json.RawMessage) json.RawMessage {
	if IsMoonshotModel(model) {
		return SanitizeMoonshotToolParameters(raw)
	}
	return sanitizeToolSchema(raw)
}

// SanitizeToolDescriptorsForModel returns a deep copy of descriptors using the
// provider-specific tool schema sanitizer selected by model.
func SanitizeToolDescriptorsForModel(model string, descriptors []ToolDescriptor) []ToolDescriptor {
	if len(descriptors) == 0 {
		return nil
	}
	out := make([]ToolDescriptor, 0, len(descriptors))
	for _, d := range descriptors {
		out = append(out, ToolDescriptor{
			Name:        d.Name,
			Description: d.Description,
			Schema:      SanitizeToolSchemaForModel(model, d.Schema),
		})
	}
	return out
}

// SanitizeMoonshotToolDescriptors returns provider-safe descriptors for
// Moonshot/Kimi's stricter flavored JSON Schema validator.
func SanitizeMoonshotToolDescriptors(descriptors []ToolDescriptor) []ToolDescriptor {
	if len(descriptors) == 0 {
		return nil
	}
	out := make([]ToolDescriptor, 0, len(descriptors))
	for _, d := range descriptors {
		out = append(out, ToolDescriptor{
			Name:        d.Name,
			Description: d.Description,
			Schema:      SanitizeMoonshotToolParameters(d.Schema),
		})
	}
	return out
}

// SanitizeMoonshotToolParameters normalizes tool parameters to the subset of
// JSON Schema accepted by Moonshot/Kimi tool calling.
func SanitizeMoonshotToolParameters(raw json.RawMessage) json.RawMessage {
	var node any
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&node); err != nil {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	top, ok := repairMoonshotSchema(node, true).(map[string]any)
	if !ok {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	if typ, _ := top["type"].(string); typ != "object" {
		top["type"] = "object"
	}
	if _, ok := top["properties"].(map[string]any); !ok {
		top["properties"] = map[string]any{}
	}
	out, err := json.Marshal(top)
	if err != nil {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	return out
}

func repairMoonshotSchema(node any, isSchema bool) any {
	switch v := node.(type) {
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, repairMoonshotSchema(item, true))
		}
		return out
	case map[string]any:
		repaired := make(map[string]any, len(v)+1)
		for key, value := range v {
			if _, ok := moonshotSchemaMapKeys[key]; ok {
				if children, ok := value.(map[string]any); ok {
					clean := make(map[string]any, len(children))
					for name, child := range children {
						clean[name] = repairMoonshotSchema(child, true)
					}
					repaired[key] = clean
					continue
				}
			}
			if _, ok := moonshotSchemaListKeys[key]; ok {
				if children, ok := value.([]any); ok {
					clean := make([]any, 0, len(children))
					for _, child := range children {
						clean = append(clean, repairMoonshotSchema(child, true))
					}
					repaired[key] = clean
					continue
				}
			}
			if _, ok := moonshotSchemaNodeKeys[key]; ok {
				if child, ok := value.(map[string]any); ok {
					repaired[key] = repairMoonshotSchema(child, true)
					continue
				}
			}
			repaired[key] = value
		}
		if !isSchema {
			return repaired
		}
		if _, ok := repaired["anyOf"].([]any); ok {
			delete(repaired, "type")
			return repaired
		}
		if _, ok := repaired["$ref"]; ok {
			return repaired
		}
		return fillMoonshotMissingType(repaired)
	default:
		return node
	}
}

func fillMoonshotMissingType(node map[string]any) map[string]any {
	if typ, _ := node["type"].(string); strings.TrimSpace(typ) != "" {
		return node
	}
	inferred := "string"
	if _, ok := node["properties"]; ok {
		inferred = "object"
	} else if _, ok := node["required"]; ok {
		inferred = "object"
	} else if _, ok := node["additionalProperties"]; ok {
		inferred = "object"
	} else if _, ok := node["items"]; ok {
		inferred = "array"
	} else if _, ok := node["prefixItems"]; ok {
		inferred = "array"
	} else if enum, ok := node["enum"].([]any); ok && len(enum) > 0 {
		inferred = moonshotEnumType(enum[0])
	}
	node["type"] = inferred
	return node
}

func moonshotEnumType(value any) string {
	switch v := value.(type) {
	case bool:
		return "boolean"
	case json.Number:
		raw := v.String()
		if strings.ContainsAny(raw, ".eE") {
			return "number"
		}
		return "integer"
	case float64, float32:
		return "number"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "integer"
	default:
		return "string"
	}
}
