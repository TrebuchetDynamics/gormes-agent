package eval

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/mutation"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/optimization"
)

const (
	PromptOptimizationConverged = optimization.Converged
	PromptOptimizationBudget    = optimization.Budget
	PromptOptimizationPerfect   = optimization.PerfectScore
)

type PromptVariant = model.Variant

type PromptOptimizationIteration = optimization.Iteration

type PromptOptimizationResult = optimization.Result

type PromptOptimizer = optimization.Optimizer

func NewPromptOptimizer(evaluator *PromptEvaluator, toolCallsFn func(context.Context, string) ([]string, error)) *PromptOptimizer {
	return optimization.New(evaluator, toolCallsFn)
}

func NewPromptOptimizerWithRunner(evaluator *PromptEvaluator, runner PromptEvalRunner) *PromptOptimizer {
	return optimization.NewWithRunner(evaluator, runner)
}

func NewPromptOptimizerWithSettings(evaluator *PromptEvaluator, runner PromptEvalRunner, toolCallsFn func(context.Context, string) ([]string, error), variantsPerIteration, maxIterations int, improvementThreshold float64) *PromptOptimizer {
	return optimization.NewWithSettings(evaluator, runner, toolCallsFn, variantsPerIteration, maxIterations, improvementThreshold)
}

func GeneratePromptVariants(base PromptVariant, iteration, limit int) []PromptVariant {
	return mutation.GenerateVariants(base, iteration, limit)
}

func StableVariantPrefix(id string) string {
	return mutation.StablePrefix(id)
}
