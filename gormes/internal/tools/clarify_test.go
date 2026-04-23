package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestClarifyTool_MultipleChoiceResponse(t *testing.T) {
	var seen ClarifyRequest
	tool := &ClarifyTool{
		Prompter: func(_ context.Context, req ClarifyRequest) (ClarifyReply, error) {
			seen = req
			return ClarifyReply{Answer: "Ship the core tool first."}, nil
		},
	}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{
		"question": "Which rollout should we take?",
		"choices": [
			"Ship the core tool first.",
			"Wait for the full TUI prompt flow."
		]
	}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if seen.Question != "Which rollout should we take?" {
		t.Fatalf("question = %q, want original question", seen.Question)
	}
	if len(seen.Choices) != 2 {
		t.Fatalf("choices len = %d, want 2", len(seen.Choices))
	}

	var payload struct {
		Question       string `json:"question"`
		Answer         string `json:"answer"`
		SelectedChoice string `json:"selected_choice"`
		UsedOther      bool   `json:"used_other"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload.Answer != "Ship the core tool first." {
		t.Fatalf("answer = %q, want selected response", payload.Answer)
	}
	if payload.SelectedChoice != "Ship the core tool first." {
		t.Fatalf("selected_choice = %q, want chosen option", payload.SelectedChoice)
	}
	if payload.UsedOther {
		t.Fatal("used_other = true, want false")
	}
}

func TestClarifyTool_RejectsTooManyChoices(t *testing.T) {
	tool := &ClarifyTool{
		Prompter: func(_ context.Context, req ClarifyRequest) (ClarifyReply, error) {
			t.Fatalf("Prompter should not be called for invalid input: %+v", req)
			return ClarifyReply{}, nil
		},
	}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{
		"question": "Pick one",
		"choices": ["one", "two", "three", "four", "five"]
	}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want validation failure")
	}
	if !strings.Contains(err.Error(), "at most 4 choices") {
		t.Fatalf("error = %q, want at most 4 choices", err)
	}
}

func TestClarifyTool_RejectsMissingPrompter(t *testing.T) {
	tool := &ClarifyTool{}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"question":"Need input?"}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want missing prompter failure")
	}
	if !strings.Contains(err.Error(), "no clarify prompter configured") {
		t.Fatalf("error = %q, want missing prompter message", err)
	}
}
