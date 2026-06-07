package eval

import (
	"context"
	"strings"
	"testing"
)

func TestPromptOptimizer_Mutate(t *testing.T) {
	evaluator := NewPromptEvaluator([]EvalScenario{
		{Name: "test", Prompt: "hello", ExpectedTools: []string{}},
	})
	optimizer := NewPromptOptimizer(evaluator, func(ctx context.Context, prompt string) ([]string, error) {
		return []string{}, nil
	})

	variant := optimizer.Mutate("hello world this is a test prompt")
	if variant.ID == "" {
		t.Fatal("mutation should produce an ID")
	}
	if variant.Prompt == "hello world this is a test prompt" {
		t.Fatal("mutation should change the prompt")
	}
	if !strings.Contains(variant.Prompt, "exact required tool") {
		t.Fatalf("mutation prompt = %q, want tool-selection heuristic", variant.Prompt)
	}
}

func TestPromptOptimizer_GenerateVariants(t *testing.T) {
	evaluator := NewPromptEvaluator([]EvalScenario{{Name: "test", Prompt: "hello"}})
	optimizer := NewPromptOptimizerWithSettings(evaluator, nil, func(ctx context.Context, prompt string) ([]string, error) {
		return []string{}, nil
	}, 4, 50, 0.05)

	variants := optimizer.GenerateVariants(PromptVariant{ID: "base", Prompt: "be helpful"}, 0)
	if len(variants) != 4 {
		t.Fatalf("variants = %d, want 4", len(variants))
	}
	seen := map[string]bool{}
	for _, variant := range variants {
		if variant.ID == "" {
			t.Fatalf("variant missing id: %+v", variant)
		}
		if variant.Prompt == "be helpful" {
			t.Fatalf("variant did not change prompt: %+v", variant)
		}
		if seen[variant.Prompt] {
			t.Fatalf("duplicate variant prompt: %q", variant.Prompt)
		}
		seen[variant.Prompt] = true
	}
	if !strings.Contains(variants[0].Prompt, "exact required tool") {
		t.Fatalf("variant[0] = %q, want tool-selection mutation", variants[0].Prompt)
	}
	if !strings.Contains(variants[1].Prompt, "concrete outcome") {
		t.Fatalf("variant[1] = %q, want response-quality mutation", variants[1].Prompt)
	}
	if !strings.Contains(variants[2].Prompt, "Break multi-step") {
		t.Fatalf("variant[2] = %q, want decomposition mutation", variants[2].Prompt)
	}
}

func TestPromptOptimizer_GenerateVariantsIncludesResearchQualityWhenCapacityAllows(t *testing.T) {
	evaluator := NewPromptEvaluator([]EvalScenario{{Name: "test", Prompt: "hello"}})
	optimizer := NewPromptOptimizerWithSettings(evaluator, nil, func(ctx context.Context, prompt string) ([]string, error) {
		return []string{}, nil
	}, 5, 50, 0.05)

	variants := optimizer.GenerateVariants(PromptVariant{ID: "base", Prompt: "be helpful"}, 0)
	found := false
	for _, variant := range variants {
		if strings.Contains(variant.Prompt, "Evaluate external-project research") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("variants missing research-quality mutation: %#v", variants)
	}
}

func TestPromptOptimizer_Optimize(t *testing.T) {
	scenarios := []EvalScenario{
		{Name: "greet", Prompt: "hello", ExpectedTools: []string{}},
		{Name: "read", Prompt: "read file", ExpectedTools: []string{"read"}},
	}
	evaluator := NewPromptEvaluator(scenarios)
	optimizer := NewPromptOptimizerWithSettings(evaluator, nil, func(ctx context.Context, prompt string) ([]string, error) {
		return []string{}, nil
	}, 4, 3, 0.01)

	best, err := optimizer.Optimize(context.Background(), "be helpful and concise")
	if err != nil {
		t.Fatal(err)
	}
	if best.Score <= 0 {
		t.Fatal("optimizer should produce a scored variant")
	}
}

func TestPromptOptimizer_OptimizeDetailedImprovesByTenPercent(t *testing.T) {
	scenarios := []EvalScenario{
		{Name: "read", Prompt: "read file", ExpectedTools: []string{"read_file"}, ExpectedOutcome: "read complete"},
	}
	evaluator := NewPromptEvaluator(scenarios)
	runner := func(ctx context.Context, variant PromptVariant, scenario EvalScenario) (EvalTrace, error) {
		trace := EvalTrace{Response: "done"}
		if strings.Contains(variant.Prompt, "exact required tool") {
			trace.Tools = append([]string(nil), scenario.ExpectedTools...)
		}
		if strings.Contains(variant.Prompt, "concrete outcome") {
			trace.Response = scenario.ExpectedOutcome
		}
		return trace, nil
	}
	optimizer := NewPromptOptimizerWithSettings(evaluator, runner, nil, 2, 4, 0.01)

	result, err := optimizer.OptimizeDetailed(context.Background(), "be helpful")
	if err != nil {
		t.Fatal(err)
	}
	if result.Base.Score <= 0 {
		t.Fatalf("base score = %v, want scored baseline", result.Base.Score)
	}
	if result.Best.Score < result.Base.Score*1.10 {
		t.Fatalf("best score = %v, base = %v, want at least 10%% improvement", result.Best.Score, result.Base.Score)
	}
	if !strings.Contains(result.Best.Prompt, "exact required tool") || !strings.Contains(result.Best.Prompt, "concrete outcome") {
		t.Fatalf("best prompt = %q, want cumulative tool and response heuristics", result.Best.Prompt)
	}
	if len(result.Iterations) < 2 {
		t.Fatalf("iterations = %d, want iterative improvement through next-base mutation", len(result.Iterations))
	}
	if result.TerminationReason == "" {
		t.Fatalf("missing termination reason: %+v", result)
	}
}

func TestPromptOptimizer_StopsWhenImprovementBelowThreshold(t *testing.T) {
	evaluator := NewPromptEvaluator([]EvalScenario{
		{Name: "noop", Prompt: "hello", ExpectedTools: []string{}, ExpectedOutcome: "done"},
	})
	runner := func(ctx context.Context, variant PromptVariant, scenario EvalScenario) (EvalTrace, error) {
		return EvalTrace{Response: scenario.ExpectedOutcome}, nil
	}
	optimizer := NewPromptOptimizerWithSettings(evaluator, runner, nil, 3, 5, 0.5)

	result, err := optimizer.OptimizeDetailed(context.Background(), "already good")
	if err != nil {
		t.Fatal(err)
	}
	if result.TerminationReason != PromptOptimizationConverged {
		t.Fatalf("termination = %q, want %q", result.TerminationReason, PromptOptimizationConverged)
	}
	if len(result.Iterations) != 1 {
		t.Fatalf("iterations = %d, want one convergence check", len(result.Iterations))
	}
}
