package moonshot

import (
	"encoding/json"
	"testing"
)

func TestMoonshotModelDetectionMatchesUpstreamAggregatorPrefixes(t *testing.T) {
	positive := []string{
		"kimi-k2.6",
		"kimi-k2-thinking",
		"moonshotai/Kimi-K2.6",
		"moonshotai/kimi-k2.6",
		"nous/moonshotai/kimi-k2.6",
		"openrouter/moonshotai/kimi-k2-thinking",
		"MOONSHOTAI/KIMI-K2.6",
	}
	for _, model := range positive {
		t.Run("positive/"+model, func(t *testing.T) {
			if !IsMoonshotModel(model) {
				t.Fatalf("IsMoonshotModel(%q) = false, want true", model)
			}
		})
	}

	negative := []string{
		"",
		"anthropic/claude-sonnet-4.6",
		"openai/gpt-5.4",
		"google/gemini-3-flash-preview",
		"deepseek-chat",
	}
	for _, model := range negative {
		t.Run("negative/"+model, func(t *testing.T) {
			if IsMoonshotModel(model) {
				t.Fatalf("IsMoonshotModel(%q) = true, want false", model)
			}
		})
	}
}

func TestSanitizeMoonshotToolParametersTraversesNestedSchemaContainers(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"dict": {
				"type": "object",
				"patternProperties": {
					"^x_": {"description": "pattern value"}
				},
				"additionalProperties": {"description": "extra value"},
				"propertyNames": {"description": "property name"}
			},
			"tuple": {
				"type": "array",
				"prefixItems": [
					{"description": "first item"}
				],
				"contains": {"description": "contained item"}
			},
			"negated": {
				"not": {"description": "disallowed shape"}
			}
		},
		"definitions": {
			"Legacy": {"properties": {"id": {"description": "legacy id"}}}
		}
	}`)

	got := decodeMoonshotSchemaForRepairTest(t, SanitizeMoonshotToolParameters(raw))
	props := schemaMapForRepairTest(t, got, "properties")

	dict := schemaMapForRepairTest(t, props, "dict")
	patterns := schemaMapForRepairTest(t, dict, "patternProperties")
	pattern := schemaMapForRepairTest(t, patterns, "^x_")
	if pattern["type"] != "string" {
		t.Fatalf("patternProperties child type = %v, want string", pattern["type"])
	}
	additional := schemaMapForRepairTest(t, dict, "additionalProperties")
	if additional["type"] != "string" {
		t.Fatalf("additionalProperties type = %v, want string", additional["type"])
	}
	propertyNames := schemaMapForRepairTest(t, dict, "propertyNames")
	if propertyNames["type"] != "string" {
		t.Fatalf("propertyNames type = %v, want string", propertyNames["type"])
	}

	tuple := schemaMapForRepairTest(t, props, "tuple")
	prefixItems, ok := tuple["prefixItems"].([]any)
	if !ok || len(prefixItems) != 1 {
		t.Fatalf("prefixItems = %#v, want one child", tuple["prefixItems"])
	}
	first, ok := prefixItems[0].(map[string]any)
	if !ok || first["type"] != "string" {
		t.Fatalf("prefixItems[0] = %#v, want typed child", prefixItems[0])
	}
	contains := schemaMapForRepairTest(t, tuple, "contains")
	if contains["type"] != "string" {
		t.Fatalf("contains type = %v, want string", contains["type"])
	}

	negated := schemaMapForRepairTest(t, props, "negated")
	not := schemaMapForRepairTest(t, negated, "not")
	if not["type"] != "string" {
		t.Fatalf("not type = %v, want string", not["type"])
	}

	definitions := schemaMapForRepairTest(t, got, "definitions")
	legacy := schemaMapForRepairTest(t, definitions, "Legacy")
	if legacy["type"] != "object" {
		t.Fatalf("definitions.Legacy.type = %v, want object", legacy["type"])
	}
}

func TestSanitizeToolDescriptorsForModelEmptyInput(t *testing.T) {
	if got := SanitizeToolDescriptorsForModel("kimi-k2", nil); got != nil {
		t.Fatalf("nil descriptors = %+v, want nil", got)
	}
	if got := SanitizeToolDescriptorsForModel("kimi-k2", []ToolDescriptor{}); got != nil {
		t.Fatalf("empty descriptors = %+v, want nil", got)
	}
}

func decodeMoonshotSchemaForRepairTest(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode schema: %v\n%s", err, raw)
	}
	return out
}

func schemaMapForRepairTest(t *testing.T, node map[string]any, key string) map[string]any {
	t.Helper()
	child, ok := node[key].(map[string]any)
	if !ok {
		t.Fatalf("%s = %T(%#v), want object", key, node[key], node[key])
	}
	return child
}
