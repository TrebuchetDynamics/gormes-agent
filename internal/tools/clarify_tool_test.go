package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestClarifyTool_SchemaIsOpenAICompatible(t *testing.T) {
	tool := NewClarifyTool(nil)
	var schema map[string]any
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("Schema() invalid JSON: %v\n%s", err, tool.Schema())
	}
	if schema["type"] != "object" {
		t.Fatalf("schema type = %v, want object", schema["type"])
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok || props["question"] == nil || props["choices"] == nil {
		t.Fatalf("schema properties = %#v, want question and choices", schema["properties"])
	}
}

func TestClarifyTool_ValidatesAndNormalizesArguments(t *testing.T) {
	tool := NewClarifyTool(nil)

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"question":"  Pick deploy region?  ","choices":[" us-east ","","eu-west",42,"ap-south","extra"]}`))
	if !errors.Is(err, ErrClarifyUnavailable) {
		t.Fatalf("Execute() error = %v, want ErrClarifyUnavailable", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("clarify result is not JSON: %v\n%s", err, raw)
	}
	if got["status"] != "clarify_unavailable" {
		t.Fatalf("status = %v, want clarify_unavailable", got["status"])
	}
	if got["question"] != "Pick deploy region?" {
		t.Fatalf("question = %v, want trimmed question", got["question"])
	}
	choices, ok := got["choices_offered"].([]any)
	if !ok {
		t.Fatalf("choices_offered = %#v, want array", got["choices_offered"])
	}
	want := []string{"us-east", "eu-west", "42", "ap-south"}
	if len(choices) != len(want) {
		t.Fatalf("choices len = %d, want %d (%v)", len(choices), len(want), choices)
	}
	for i := range want {
		if choices[i] != want[i] {
			t.Fatalf("choices[%d] = %v, want %q (all=%v)", i, choices[i], want[i], choices)
		}
	}
	if got["truncated"] != true {
		t.Fatalf("truncated = %v, want true", got["truncated"])
	}
}

func TestClarifyTool_InvalidArgumentsReturnTypedEvidence(t *testing.T) {
	tool := NewClarifyTool(nil)

	for _, tc := range []struct {
		name string
		args string
		want string
	}{
		{name: "missing question", args: `{}`, want: "question_required"},
		{name: "blank question", args: `{"question":"   "}`, want: "question_required"},
		{name: "choices not list", args: `{"question":"Continue?","choices":"yes"}`, want: "choices_must_be_list"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := tool.Execute(context.Background(), json.RawMessage(tc.args))
			if err == nil {
				t.Fatalf("Execute() error = nil, want invalid args")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if !strings.Contains(string(raw), `"status":"clarify_invalid_args"`) || !strings.Contains(string(raw), tc.want) {
				t.Fatalf("raw = %s, want clarify_invalid_args with %q", raw, tc.want)
			}
		})
	}
}

func TestClarifyTool_CallbackReturnsUserResponse(t *testing.T) {
	var seen ClarifyRequest
	tool := NewClarifyTool(ClarifyCallbackFunc(func(ctx context.Context, req ClarifyRequest) (ClarifyResponse, error) {
		seen = req
		return ClarifyResponse{UserResponse: "eu-west"}, nil
	}))

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"question":"Region?","choices":["us-east","eu-west"]}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if seen.Question != "Region?" || len(seen.Choices) != 2 || seen.Choices[1] != "eu-west" {
		t.Fatalf("callback request = %#v, want normalized question/choices", seen)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("clarify result is not JSON: %v\n%s", err, raw)
	}
	if got["status"] != "clarify_answered" {
		t.Fatalf("status = %v, want clarify_answered", got["status"])
	}
	if got["user_response"] != "eu-west" {
		t.Fatalf("user_response = %v, want eu-west", got["user_response"])
	}
}

func TestClarifyTool_CallbackFailureIsRouteMissingEvidence(t *testing.T) {
	tool := NewClarifyTool(ClarifyCallbackFunc(func(context.Context, ClarifyRequest) (ClarifyResponse, error) {
		return ClarifyResponse{}, ErrClarifyRouteMissing
	}))

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"question":"Proceed?"}`))
	if err == nil {
		t.Fatalf("Execute() error = nil, want route missing")
	}
	if !strings.Contains(err.Error(), "clarify_route_missing") {
		t.Fatalf("error = %v, want clarify_route_missing", err)
	}
	if !strings.Contains(string(raw), `"status":"clarify_route_missing"`) {
		t.Fatalf("raw = %s, want clarify_route_missing", raw)
	}
}
