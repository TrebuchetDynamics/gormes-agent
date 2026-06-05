package llm

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

func TestSanitizeToolSchemaForModelRepairsMoonshotFlavoredSchema(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"description": "search text"},
			"filter": {
				"type": "string",
				"anyOf": [
					{"type": "string"},
					{"description": "typeless option"}
				]
			},
			"tags": {
				"type": "array",
				"items": {"description": "tag"}
			},
			"mode": {"enum": [true, false]},
			"payload": {"$ref": "#/$defs/Payload"}
		},
		"$defs": {
			"Payload": {"properties": {"id": {"description": "identifier"}}}
		},
		"required": ["query"]
	}`)
	original := string(raw)

	sanitized := SanitizeToolSchemaForModel("openrouter/moonshotai/kimi-k2-thinking", raw)

	if string(raw) != original {
		t.Fatalf("SanitizeToolSchemaForModel mutated input:\n got %s\nwant %s", raw, original)
	}
	schema := decodeMoonshotSchemaForTest(t, sanitized)
	props := schemaMapForTest(t, schema, "properties")

	query := schemaMapForTest(t, props, "query")
	if got := query["type"]; got != "string" {
		t.Fatalf("query.type = %v, want string", got)
	}

	filter := schemaMapForTest(t, props, "filter")
	if _, ok := filter["type"]; ok {
		t.Fatalf("filter kept parent type with anyOf: %+v", filter)
	}
	filterAnyOf, ok := filter["anyOf"].([]any)
	if !ok || len(filterAnyOf) != 2 {
		t.Fatalf("filter.anyOf = %#v, want two children", filter["anyOf"])
	}
	secondAnyOf, ok := filterAnyOf[1].(map[string]any)
	if !ok {
		t.Fatalf("filter.anyOf[1] = %T, want object", filterAnyOf[1])
	}
	if got := secondAnyOf["type"]; got != "string" {
		t.Fatalf("filter.anyOf[1].type = %v, want string", got)
	}

	tags := schemaMapForTest(t, props, "tags")
	tagItems := schemaMapForTest(t, tags, "items")
	if got := tagItems["type"]; got != "string" {
		t.Fatalf("tags.items.type = %v, want string", got)
	}

	mode := schemaMapForTest(t, props, "mode")
	if got := mode["type"]; got != "boolean" {
		t.Fatalf("mode.type = %v, want boolean inferred from enum", got)
	}

	payload := schemaMapForTest(t, props, "payload")
	if _, ok := payload["type"]; ok {
		t.Fatalf("$ref payload received synthetic type: %+v", payload)
	}

	defs := schemaMapForTest(t, schema, "$defs")
	payloadDef := schemaMapForTest(t, defs, "Payload")
	if got := payloadDef["type"]; got != "object" {
		t.Fatalf("$defs.Payload.type = %v, want object inferred from properties", got)
	}
}

func TestSanitizeToolSchemaForModelUsesGenericSchemaForNonMoonshotModels(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"description": "search text"}
		}
	}`)

	sanitized := SanitizeToolSchemaForModel("openai/gpt-5.4", raw)
	schema := decodeMoonshotSchemaForTest(t, sanitized)
	query := schemaMapForTest(t, schemaMapForTest(t, schema, "properties"), "query")
	if _, ok := query["type"]; ok {
		t.Fatalf("non-Moonshot model received Moonshot missing-type repair: %+v", query)
	}
}

func TestSanitizeMoonshotToolParametersReturnsEmptyObjectForInvalidInputs(t *testing.T) {
	inputs := []json.RawMessage{
		nil,
		json.RawMessage(``),
		json.RawMessage(`"garbage"`),
		json.RawMessage(`[]`),
		json.RawMessage(`{not-json`),
	}
	for _, input := range inputs {
		t.Run(string(input), func(t *testing.T) {
			got := decodeMoonshotSchemaForTest(t, SanitizeMoonshotToolParameters(input))
			if got["type"] != "object" {
				t.Fatalf("type = %v, want object for %q", got["type"], input)
			}
			props := schemaMapForTest(t, got, "properties")
			if len(props) != 0 {
				t.Fatalf("properties = %+v, want empty object", props)
			}
		})
	}
}

func TestSanitizeMoonshotToolParametersCoercesNonObjectTopLevel(t *testing.T) {
	got := decodeMoonshotSchemaForTest(t, SanitizeMoonshotToolParameters(json.RawMessage(`{"type":"string"}`)))
	if got["type"] != "object" {
		t.Fatalf("type = %v, want object", got["type"])
	}
	props := schemaMapForTest(t, got, "properties")
	if len(props) != 0 {
		t.Fatalf("properties = %+v, want empty object", props)
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

	got := decodeMoonshotSchemaForTest(t, SanitizeMoonshotToolParameters(raw))
	props := schemaMapForTest(t, got, "properties")

	dict := schemaMapForTest(t, props, "dict")
	patterns := schemaMapForTest(t, dict, "patternProperties")
	pattern := schemaMapForTest(t, patterns, "^x_")
	if pattern["type"] != "string" {
		t.Fatalf("patternProperties child type = %v, want string", pattern["type"])
	}
	additional := schemaMapForTest(t, dict, "additionalProperties")
	if additional["type"] != "string" {
		t.Fatalf("additionalProperties type = %v, want string", additional["type"])
	}
	propertyNames := schemaMapForTest(t, dict, "propertyNames")
	if propertyNames["type"] != "string" {
		t.Fatalf("propertyNames type = %v, want string", propertyNames["type"])
	}

	tuple := schemaMapForTest(t, props, "tuple")
	prefixItems, ok := tuple["prefixItems"].([]any)
	if !ok || len(prefixItems) != 1 {
		t.Fatalf("prefixItems = %#v, want one child", tuple["prefixItems"])
	}
	first, ok := prefixItems[0].(map[string]any)
	if !ok || first["type"] != "string" {
		t.Fatalf("prefixItems[0] = %#v, want typed child", prefixItems[0])
	}
	contains := schemaMapForTest(t, tuple, "contains")
	if contains["type"] != "string" {
		t.Fatalf("contains type = %v, want string", contains["type"])
	}

	negated := schemaMapForTest(t, props, "negated")
	not := schemaMapForTest(t, negated, "not")
	if not["type"] != "string" {
		t.Fatalf("not type = %v, want string", not["type"])
	}

	definitions := schemaMapForTest(t, got, "definitions")
	legacy := schemaMapForTest(t, definitions, "Legacy")
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

func TestOpenAICompatibleChatRequestUsesMoonshotSanitizerForAggregatorModels(t *testing.T) {
	client := &httpClient{baseURL: "https://openrouter.ai/api/v1"}
	body, descriptors, err := client.buildOpenAICompatibleChatRequestBody(ChatRequest{
		Model: "openrouter/moonshotai/kimi-k2-thinking",
		Messages: []Message{
			{Role: "user", Content: "search"},
		},
		Tools: []ToolDescriptor{{
			Name:        "search",
			Description: "Search",
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"q": {"description": "query"},
					"format": {
						"type": "string",
						"anyOf": [{"type": "string"}, {"type": "null"}]
					}
				}
			}`),
		}},
	})
	if err != nil {
		t.Fatalf("buildOpenAICompatibleChatRequestBody() error = %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptors len = %d, want 1", len(descriptors))
	}

	wire := decodeOpenAICompatibleToolsForTest(t, body)
	params := decodeMoonshotSchemaForTest(t, wire[0].Function.Parameters)
	props := schemaMapForTest(t, params, "properties")
	q := schemaMapForTest(t, props, "q")
	if got := q["type"]; got != "string" {
		t.Fatalf("wire q.type = %v, want string", got)
	}
	format := schemaMapForTest(t, props, "format")
	if _, ok := format["type"]; ok {
		t.Fatalf("wire format kept parent type with anyOf: %+v", format)
	}

	descriptorSchema := decodeMoonshotSchemaForTest(t, descriptors[0].Schema)
	descriptorQ := schemaMapForTest(t, schemaMapForTest(t, descriptorSchema, "properties"), "q")
	if got := descriptorQ["type"]; got != "string" {
		t.Fatalf("descriptor q.type = %v, want string", got)
	}
}

func TestOpenAICompatibleChatRequestLeavesNonMoonshotSchemasGeneric(t *testing.T) {
	client := &httpClient{baseURL: "https://openrouter.ai/api/v1"}
	body, descriptors, err := client.buildOpenAICompatibleChatRequestBody(ChatRequest{
		Model: "anthropic/claude-sonnet-4.6",
		Messages: []Message{
			{Role: "user", Content: "search"},
		},
		Tools: []ToolDescriptor{{
			Name:        "search",
			Description: "Search",
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"q": {"description": "query"}
				}
			}`),
		}},
	})
	if err != nil {
		t.Fatalf("buildOpenAICompatibleChatRequestBody() error = %v", err)
	}

	wire := decodeOpenAICompatibleToolsForTest(t, body)
	wireParams := decodeMoonshotSchemaForTest(t, wire[0].Function.Parameters)
	wireQ := schemaMapForTest(t, schemaMapForTest(t, wireParams, "properties"), "q")
	if _, ok := wireQ["type"]; ok {
		t.Fatalf("wire non-Moonshot q received Moonshot missing-type repair: %+v", wireQ)
	}

	descriptorSchema := decodeMoonshotSchemaForTest(t, descriptors[0].Schema)
	descriptorQ := schemaMapForTest(t, schemaMapForTest(t, descriptorSchema, "properties"), "q")
	if _, ok := descriptorQ["type"]; ok {
		t.Fatalf("descriptor non-Moonshot q received Moonshot missing-type repair: %+v", descriptorQ)
	}
}

func decodeMoonshotSchemaForTest(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode schema: %v\n%s", err, raw)
	}
	return out
}

func schemaMapForTest(t *testing.T, node map[string]any, key string) map[string]any {
	t.Helper()
	child, ok := node[key].(map[string]any)
	if !ok {
		t.Fatalf("%s = %T(%#v), want object", key, node[key], node[key])
	}
	return child
}

func decodeOpenAICompatibleToolsForTest(t *testing.T, body []byte) []orToolDescriptor {
	t.Helper()
	var payload struct {
		Tools []orToolDescriptor `json:"tools"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode request body: %v\n%s", err, body)
	}
	return payload.Tools
}
