package hermes

import (
	"context"
	"testing"
)

func TestPromptOptimizer_Mutate(t *testing.T) {
	evaluator := NewPromptEvaluator([]EvalScenario{
		{Name: "test", Prompt: "hello", ExpectedTools: []string{}},
	})
	optimizer := NewPromptOptimizer(evaluator, func(ctx context.Context, prompt string) ([]string, error) {
		return []string{}, nil
	})

	variant := optimizer.mutate("hello world this is a test prompt")
	if variant.ID == "" {
		t.Fatal("mutation should produce an ID")
	}
	if variant.Prompt == "hello world this is a test prompt" {
		t.Fatal("mutation should change the prompt")
	}
}

func TestPromptOptimizer_Optimize(t *testing.T) {
	scenarios := []EvalScenario{
		{Name: "greet", Prompt: "hello", ExpectedTools: []string{}},
		{Name: "read", Prompt: "read file", ExpectedTools: []string{"read"}},
	}
	evaluator := NewPromptEvaluator(scenarios)
	optimizer := NewPromptOptimizer(evaluator, func(ctx context.Context, prompt string) ([]string, error) {
		return []string{}, nil
	})
	optimizer.maxIterations = 3
	optimizer.improvementThreshold = 0.01

	best, err := optimizer.Optimize(context.Background(), "be helpful and concise")
	if err != nil {
		t.Fatal(err)
	}
	if best.Score <= 0 {
		t.Fatal("optimizer should produce a scored variant")
	}
}
