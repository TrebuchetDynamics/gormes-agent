package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var echoToolDescriptor = ToolDescriptor{
	Name:        "echo",
	Description: "echo text",
	Schema:      json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"],"additionalProperties":false}`),
}

var arrayObjectCoercionToolDescriptor = ToolDescriptor{
	Name:        "coerce_args",
	Description: "exercise schema-guided provider argument coercion",
	Schema: json.RawMessage(`{
		"type":"object",
		"properties":{
			"urls":{"type":"array","items":{"type":"string"}},
			"config":{"type":"object","properties":{"max":{"type":"integer"}},"additionalProperties":false},
			"stages":{"type":"array","items":{"type":"object"},"nullable":true}
		},
		"additionalProperties":false
	}`),
}

var integerArrayCoercionToolDescriptor = ToolDescriptor{
	Name:        "coerce_ids",
	Description: "exercise malformed array-string validation",
	Schema:      json.RawMessage(`{"type":"object","properties":{"ids":{"type":"array","items":{"type":"integer"}}},"required":["ids"],"additionalProperties":false}`),
}

func TestRepairToolCalls_ToolCallArgumentsStringifiedArrayCoercedAgainstSchema(t *testing.T) {
	repaired := repairSingleToolCall(t, arrayObjectCoercionToolDescriptor, ToolCall{
		ID:        "call_urls",
		Name:      "coerce_args",
		Arguments: json.RawMessage(`{"urls":"[\"https://a.test\",\"https://b.test\"]"}`),
	})

	var got struct {
		URLs []string `json:"urls"`
	}
	if err := json.Unmarshal(repaired.Arguments, &got); err != nil {
		t.Fatalf("repaired arguments are invalid JSON: %v: %s", err, repaired.Arguments)
	}
	want := []string{"https://a.test", "https://b.test"}
	if len(got.URLs) != len(want) {
		t.Fatalf("urls len = %d, want %d: %s", len(got.URLs), len(want), repaired.Arguments)
	}
	for i := range want {
		if got.URLs[i] != want[i] {
			t.Fatalf("urls[%d] = %q, want %q (all=%v)", i, got.URLs[i], want[i], got.URLs)
		}
	}
}

func TestRepairToolCalls_ToolCallArgumentsStringifiedObjectCoercedAgainstSchema(t *testing.T) {
	repaired := repairSingleToolCall(t, arrayObjectCoercionToolDescriptor, ToolCall{
		ID:        "call_config",
		Name:      "coerce_args",
		Arguments: json.RawMessage(`{"config":"{\"max\":50}"}`),
	})

	var got struct {
		Config struct {
			Max int `json:"max"`
		} `json:"config"`
	}
	if err := json.Unmarshal(repaired.Arguments, &got); err != nil {
		t.Fatalf("repaired arguments are invalid JSON: %v: %s", err, repaired.Arguments)
	}
	if got.Config.Max != 50 {
		t.Fatalf("config.max = %d, want 50: %s", got.Config.Max, repaired.Arguments)
	}
}

func TestRepairToolCalls_ToolCallArgumentsNullableStringNullCoercedAgainstSchema(t *testing.T) {
	repaired := repairSingleToolCall(t, arrayObjectCoercionToolDescriptor, ToolCall{
		ID:        "call_null",
		Name:      "coerce_args",
		Arguments: json.RawMessage(`{"stages":"null"}`),
	})

	var got map[string]any
	if err := json.Unmarshal(repaired.Arguments, &got); err != nil {
		t.Fatalf("repaired arguments are invalid JSON: %v: %s", err, repaired.Arguments)
	}
	if value, ok := got["stages"]; !ok || value != nil {
		t.Fatalf("stages = %#v (present=%v), want explicit null: %s", value, ok, repaired.Arguments)
	}
}

func TestRepairToolCalls_ToolCallArgumentsMalformedArrayStringWrapsWhenItemSchemaAllowsString(t *testing.T) {
	repaired := repairSingleToolCall(t, arrayObjectCoercionToolDescriptor, ToolCall{
		ID:        "call_malformed_array",
		Name:      "coerce_args",
		Arguments: json.RawMessage(`{"urls":"[not-json"}`),
	})

	var got struct {
		URLs []string `json:"urls"`
	}
	if err := json.Unmarshal(repaired.Arguments, &got); err != nil {
		t.Fatalf("repaired arguments are invalid JSON: %v: %s", err, repaired.Arguments)
	}
	if len(got.URLs) != 1 || got.URLs[0] != "[not-json" {
		t.Fatalf("urls = %#v, want single malformed string fallback: %s", got.URLs, repaired.Arguments)
	}
}

func TestRepairToolCalls_ToolCallArgumentsMalformedArrayStringStillValidatesWrappedItem(t *testing.T) {
	_, err := RepairToolCalls([]ToolCall{{
		ID:        "call_ids",
		Name:      "coerce_ids",
		Arguments: json.RawMessage(`{"ids":"[not-json"}`),
	}}, []ToolDescriptor{integerArrayCoercionToolDescriptor})
	if err == nil {
		t.Fatal("RepairToolCalls() error = nil, want item-schema validation failure")
	}
	if !strings.Contains(err.Error(), `argument "ids[0]" must be an integer`) {
		t.Fatalf("error = %q, want wrapped item validation evidence", err.Error())
	}
}

func TestStream_ToolCallArgumentsRepairDeterministicAgainstAdvertisedSchema(t *testing.T) {
	final, err := runToolCallRepairStream(t, []ToolDescriptor{echoToolDescriptor}, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_echo","type":"function","function":{"name":"echo","arguments":"{\"text\":\"hi\","}}]}}]}

data: {"choices":[{"finish_reason":"tool_calls"}]}

data: [DONE]

`)
	if err != nil {
		t.Fatalf("stream returned error: %v", err)
	}
	if len(final.ToolCalls) != 1 {
		t.Fatalf("tool calls len = %d, want 1", len(final.ToolCalls))
	}
	call := final.ToolCalls[0]
	if call.Name != "echo" || call.ID != "call_echo" {
		t.Fatalf("tool call = %+v, want call_echo/echo", call)
	}
	var got map[string]string
	if err := json.Unmarshal(call.Arguments, &got); err != nil {
		t.Fatalf("repaired arguments are invalid JSON: %v: %s", err, call.Arguments)
	}
	if got["text"] != "hi" {
		t.Fatalf("repaired arguments = %s, want text=hi", call.Arguments)
	}
}

func repairSingleToolCall(t *testing.T, descriptor ToolDescriptor, call ToolCall) ToolCall {
	t.Helper()
	repaired, err := RepairToolCalls([]ToolCall{call}, []ToolDescriptor{descriptor})
	if err != nil {
		t.Fatalf("RepairToolCalls() error = %v", err)
	}
	if len(repaired) != 1 {
		t.Fatalf("repaired len = %d, want 1", len(repaired))
	}
	return repaired[0]
}

func TestStream_ToolCallArgumentsRejectImpossibleRepairBeforeExecution(t *testing.T) {
	_, err := runToolCallRepairStream(t, []ToolDescriptor{echoToolDescriptor}, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_echo","type":"function","function":{"name":"echo","arguments":"{\"text\":"}}]}}]}

data: {"choices":[{"finish_reason":"tool_calls"}]}

data: [DONE]

`)
	if err == nil {
		t.Fatal("stream error = nil, want tool-call repair error")
	}
	var repairErr *ToolCallRepairError
	if !errors.As(err, &repairErr) {
		t.Fatalf("stream error = %T %v, want ToolCallRepairError", err, err)
	}
	if repairErr.ToolName != "echo" || repairErr.ToolCallID != "call_echo" {
		t.Fatalf("repair error = %+v, want call_echo/echo", repairErr)
	}
}

func TestStream_ToolCallArgumentsRejectUnavailableTool(t *testing.T) {
	_, err := runToolCallRepairStream(t, []ToolDescriptor{echoToolDescriptor}, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_missing","type":"function","function":{"name":"missing","arguments":"{}"}}]}}]}

data: {"choices":[{"finish_reason":"tool_calls"}]}

data: [DONE]

`)
	if err == nil {
		t.Fatal("stream error = nil, want unavailable-tool repair error")
	}
	var repairErr *ToolCallRepairError
	if !errors.As(err, &repairErr) {
		t.Fatalf("stream error = %T %v, want ToolCallRepairError", err, err)
	}
	if !strings.Contains(repairErr.Error(), "not advertised") {
		t.Fatalf("repair error = %q, want not advertised", repairErr.Error())
	}
}

func TestStream_ToolCallArgumentsRejectMissingRequiredAfterRepair(t *testing.T) {
	_, err := runToolCallRepairStream(t, []ToolDescriptor{echoToolDescriptor}, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_echo","type":"function","function":{"name":"echo","arguments":"None"}}]}}]}

data: {"choices":[{"finish_reason":"tool_calls"}]}

data: [DONE]

`)
	if err == nil {
		t.Fatal("stream error = nil, want missing-required repair error")
	}
	if !strings.Contains(err.Error(), `missing required argument "text"`) {
		t.Fatalf("stream error = %q, want missing required text", err.Error())
	}
}

func TestSanitizeToolDescriptorsUsesProviderSafeSchemasWithoutMutatingInput(t *testing.T) {
	descriptors := []ToolDescriptor{{
		Name:        "read_file",
		Description: "read a file",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": ["string", "null"]},
				"metadata": "object"
			},
			"required": ["path", "missing"]
		}`),
	}}
	original := append(json.RawMessage(nil), descriptors[0].Schema...)

	sanitized := SanitizeToolDescriptors(descriptors)

	if string(descriptors[0].Schema) != string(original) {
		t.Fatalf("SanitizeToolDescriptors mutated input schema:\n got %s\nwant %s", descriptors[0].Schema, original)
	}
	var schema struct {
		Type       string `json:"type"`
		Required   []string
		Properties map[string]struct {
			Type       string         `json:"type"`
			Nullable   bool           `json:"nullable,omitempty"`
			Properties map[string]any `json:"properties,omitempty"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(sanitized[0].Schema, &schema); err != nil {
		t.Fatalf("sanitized schema invalid JSON: %v: %s", err, sanitized[0].Schema)
	}
	if schema.Type != "object" {
		t.Fatalf("top-level type = %q, want object", schema.Type)
	}
	if len(schema.Required) != 1 || schema.Required[0] != "path" {
		t.Fatalf("required = %v, want [path]", schema.Required)
	}
	if got := schema.Properties["path"]; got.Type != "string" || !got.Nullable {
		t.Fatalf("path schema = %+v, want nullable string", got)
	}
	if got := schema.Properties["metadata"]; got.Type != "object" || got.Properties == nil {
		t.Fatalf("metadata schema = %+v, want object with properties", got)
	}
}

func runToolCallRepairStream(t *testing.T, tools []ToolDescriptor, fixture string) (Event, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, fixture)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model:    "x",
		Messages: []Message{{Role: "user", Content: "echo hi"}},
		Tools:    tools,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	for {
		e, err := s.Recv(context.Background())
		if err == io.EOF {
			return Event{}, io.EOF
		}
		if err != nil {
			return Event{}, err
		}
		if e.Kind == EventDone {
			return e, nil
		}
	}
}
