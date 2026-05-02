package hermes

import (
	"context"
	"testing"
)

func TestPromptEvaluator_Evaluate(t *testing.T) {
	scenarios := []EvalScenario{
		{Name: "greeting", Prompt: "hello", ExpectedTools: []string{}, ExpectedOutcome: "greet"},
		{Name: "file_read", Prompt: "read main.go", ExpectedTools: []string{"read_file"}, ExpectedOutcome: "read"},
	}
	evaluator := NewPromptEvaluator(scenarios)

	results, err := evaluator.Evaluate(context.Background(), "v1", func(ctx context.Context, prompt string) ([]string, error) {
		if prompt == "hello" {
			return []string{}, nil
		}
		return []string{"read_file"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if !results[0].TaskSuccess {
		t.Fatal("greeting should succeed with no tools")
	}
	if !results[1].TaskSuccess {
		t.Fatal("file_read should succeed with correct tool")
	}
}

func TestPromptEvaluator_AggregateScore(t *testing.T) {
	results := []EvalResult{
		{AggregateScore: 0.8},
		{AggregateScore: 0.9},
		{AggregateScore: 0.7},
	}
	score := AggregateScore(results)
	if score != 0.8 {
		t.Fatalf("aggregate = %f, want 0.8", score)
	}
}

func TestComputeToolAccuracy(t *testing.T) {
	if computeToolAccuracy([]string{"read_file"}, []string{"read_file"}) != 1.0 {
		t.Fatal("exact match should be 1.0")
	}
	if computeToolAccuracy([]string{"write_file"}, []string{"read_file"}) != 0.0 {
		t.Fatal("no match should be 0.0")
	}
	if computeToolAccuracy([]string{"read_file"}, []string{"read_file", "edit_file"}) != 0.5 {
		t.Fatal("partial match should be 0.5")
	}
}
