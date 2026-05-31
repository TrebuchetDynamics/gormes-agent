package prompts

import (
	"context"

	prompteval "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval"
)

const (
	PromptOptimizationConverged = prompteval.PromptOptimizationConverged
	PromptOptimizationBudget    = prompteval.PromptOptimizationBudget
	PromptOptimizationPerfect   = prompteval.PromptOptimizationPerfect
)

type PromptVariant = prompteval.PromptVariant
type PromptOptimizationIteration = prompteval.PromptOptimizationIteration
type PromptOptimizationResult = prompteval.PromptOptimizationResult
type PromptOptimizer = prompteval.PromptOptimizer

func NewPromptOptimizer(evaluator *PromptEvaluator, toolCallsFn func(context.Context, string) ([]string, error)) *PromptOptimizer {
	return prompteval.NewPromptOptimizer(evaluator, toolCallsFn)
}

func NewPromptOptimizerWithRunner(evaluator *PromptEvaluator, runner PromptEvalRunner) *PromptOptimizer {
	return prompteval.NewPromptOptimizerWithRunner(evaluator, runner)
}

func NewPromptOptimizerWithSettings(evaluator *PromptEvaluator, runner PromptEvalRunner, toolCallsFn func(context.Context, string) ([]string, error), variantsPerIteration, maxIterations int, improvementThreshold float64) *PromptOptimizer {
	return prompteval.NewPromptOptimizerWithSettings(evaluator, runner, toolCallsFn, variantsPerIteration, maxIterations, improvementThreshold)
}

func GeneratePromptVariants(base PromptVariant, iteration, limit int) []PromptVariant {
	return prompteval.GeneratePromptVariants(base, iteration, limit)
}

func StableVariantPrefix(id string) string {
	return prompteval.StableVariantPrefix(id)
}
