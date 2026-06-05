package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts"
)

const (
	PromptOptimizationConverged = prompts.PromptOptimizationConverged
	PromptOptimizationBudget    = prompts.PromptOptimizationBudget
	PromptOptimizationPerfect   = prompts.PromptOptimizationPerfect
)

type PromptVariant = prompts.PromptVariant
type PromptOptimizationIteration = prompts.PromptOptimizationIteration
type PromptOptimizationResult = prompts.PromptOptimizationResult

type PromptOptimizer struct {
	evaluator            *PromptEvaluator
	variantsPerIteration int
	maxIterations        int
	improvementThreshold float64
	runner               PromptEvalRunner
	toolCallsFn          func(context.Context, string) ([]string, error)
}

func NewPromptOptimizer(evaluator *PromptEvaluator, toolCallsFn func(context.Context, string) ([]string, error)) *PromptOptimizer {
	optimizer := NewPromptOptimizerWithRunner(evaluator, nil)
	optimizer.toolCallsFn = toolCallsFn
	optimizer.runner = func(ctx context.Context, variant PromptVariant, scenario EvalScenario) (EvalTrace, error) {
		if toolCallsFn == nil {
			return EvalTrace{Response: scenario.ExpectedOutcome}, nil
		}
		tools, err := toolCallsFn(ctx, variant.Prompt+"\n\nTask: "+scenario.Prompt)
		if err != nil {
			return EvalTrace{}, err
		}
		return EvalTrace{Tools: tools, Response: scenario.ExpectedOutcome}, nil
	}
	return optimizer
}

func NewPromptOptimizerWithRunner(evaluator *PromptEvaluator, runner PromptEvalRunner) *PromptOptimizer {
	return &PromptOptimizer{
		evaluator:            evaluator,
		variantsPerIteration: 4,
		maxIterations:        50,
		improvementThreshold: 0.05,
		runner:               runner,
	}
}

func (o *PromptOptimizer) delegate() *prompts.PromptOptimizer {
	if o == nil {
		return nil
	}
	return prompts.NewPromptOptimizerWithSettings(o.evaluator, o.runner, o.toolCallsFn, o.variantsPerIteration, o.maxIterations, o.improvementThreshold)
}

func (o *PromptOptimizer) Optimize(ctx context.Context, basePrompt string) (*PromptVariant, error) {
	result, err := o.OptimizeDetailed(ctx, basePrompt)
	if err != nil {
		return nil, err
	}
	best := result.Best
	return &best, nil
}

func (o *PromptOptimizer) OptimizeDetailed(ctx context.Context, basePrompt string) (PromptOptimizationResult, error) {
	delegate := o.delegate()
	if delegate == nil {
		return PromptOptimizationResult{}, fmt.Errorf("prompt optimizer requires evaluator")
	}
	return delegate.OptimizeDetailed(ctx, basePrompt)
}

func (o *PromptOptimizer) mutate(prompt string) PromptVariant {
	variants := o.GenerateVariants(PromptVariant{ID: "base", Prompt: prompt}, 0)
	if len(variants) == 0 {
		return PromptVariant{ID: "mut_empty", Prompt: strings.TrimSpace(prompt)}
	}
	return variants[0]
}

func (o *PromptOptimizer) GenerateVariants(base PromptVariant, iteration int) []PromptVariant {
	limit := 0
	if o != nil {
		limit = o.variantsPerIteration
	}
	return prompts.GeneratePromptVariants(base, iteration, limit)
}

func stableVariantPrefix(id string) string {
	return prompts.StableVariantPrefix(id)
}
