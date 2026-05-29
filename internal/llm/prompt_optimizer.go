package llm

import (
	"context"
	"fmt"
	"strings"
)

const (
	PromptOptimizationConverged = "converged"
	PromptOptimizationBudget    = "budget_exhausted"
	PromptOptimizationPerfect   = "perfect_score"
)

type PromptVariant struct {
	ID     string
	Prompt string
	Score  float64
}

type PromptOptimizationIteration struct {
	Index       int
	Base        PromptVariant
	Candidates  []PromptVariant
	Evaluations []VariantEvaluation
	Selected    PromptVariant
	Improved    bool
}

type PromptOptimizationResult struct {
	Base              PromptVariant
	Best              PromptVariant
	Iterations        []PromptOptimizationIteration
	ImprovementRatio  float64
	TerminationReason string
}

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

func (o *PromptOptimizer) Optimize(ctx context.Context, basePrompt string) (*PromptVariant, error) {
	result, err := o.OptimizeDetailed(ctx, basePrompt)
	if err != nil {
		return nil, err
	}
	best := result.Best
	return &best, nil
}

func (o *PromptOptimizer) OptimizeDetailed(ctx context.Context, basePrompt string) (PromptOptimizationResult, error) {
	if o == nil || o.evaluator == nil {
		return PromptOptimizationResult{}, fmt.Errorf("prompt optimizer requires evaluator")
	}
	runner := o.runner
	if runner == nil {
		runner = func(ctx context.Context, variant PromptVariant, scenario EvalScenario) (EvalTrace, error) {
			return EvalTrace{Response: scenario.ExpectedOutcome}, nil
		}
	}

	base := PromptVariant{ID: "base", Prompt: strings.TrimSpace(basePrompt)}
	baseEvaluation, err := o.evaluator.EvaluateVariant(ctx, base, runner)
	if err != nil {
		return PromptOptimizationResult{}, err
	}
	base.Score = baseEvaluation.AggregateScore
	best := base

	result := PromptOptimizationResult{
		Base:              base,
		Best:              best,
		TerminationReason: PromptOptimizationBudget,
	}
	maxIterations := o.maxIterations
	if maxIterations <= 0 {
		maxIterations = 1
	}
	for i := 0; i < maxIterations; i++ {
		iterationBase := best
		candidates := o.GenerateVariants(iterationBase, i)
		evaluations, err := o.evaluator.CompareVariants(ctx, candidates, runner)
		if err != nil {
			return PromptOptimizationResult{}, err
		}
		selected := PromptVariant{}
		if len(evaluations) > 0 {
			selectedEval := evaluations[0]
			selected = PromptVariant{ID: selectedEval.VariantID, Prompt: selectedEval.Prompt, Score: selectedEval.AggregateScore}
		}
		improved := selected.Score > best.Score+o.improvementThreshold
		if improved {
			best = selected
			result.Best = best
		}
		result.Iterations = append(result.Iterations, PromptOptimizationIteration{
			Index:       i,
			Base:        iterationBase,
			Candidates:  candidates,
			Evaluations: evaluations,
			Selected:    selected,
			Improved:    improved,
		})
		if !improved {
			result.TerminationReason = PromptOptimizationConverged
			break
		}
		if best.Score >= 1 {
			result.TerminationReason = PromptOptimizationPerfect
			break
		}
	}
	if base.Score > 0 {
		result.ImprovementRatio = round2((result.Best.Score - base.Score) / base.Score)
	}
	return result, nil
}

func (o *PromptOptimizer) mutate(prompt string) PromptVariant {
	variants := o.GenerateVariants(PromptVariant{ID: "base", Prompt: prompt}, 0)
	if len(variants) == 0 {
		return PromptVariant{ID: "mut_empty", Prompt: strings.TrimSpace(prompt)}
	}
	return variants[0]
}

func (o *PromptOptimizer) GenerateVariants(base PromptVariant, iteration int) []PromptVariant {
	prompt := strings.TrimSpace(base.Prompt)
	if prompt == "" {
		prompt = "Be accurate."
	}
	strategies := []struct {
		idSuffix string
		line     string
	}{
		{
			idSuffix: "tool-selection",
			line:     "When a task maps to a tool, call the exact required tool before answering.",
		},
		{
			idSuffix: "response-quality",
			line:     "Report the concrete outcome in the response using task-specific terms.",
		},
		{
			idSuffix: "decomposition",
			line:     "Break multi-step tasks into search, edit, verify, and document steps.",
		},
		{
			idSuffix: "command-safety",
			line:     "Classify commands before execution and block destructive operations.",
		},
		{
			idSuffix: "research-quality",
			line:     "Evaluate external-project research with source-backed maturity, license, fit, limitations, and a test-backed workflow recommendation.",
		},
	}

	limit := o.variantsPerIteration
	if limit <= 0 || limit > len(strategies) {
		limit = len(strategies)
	}
	variants := make([]PromptVariant, 0, limit)
	for i := 0; i < limit; i++ {
		strategy := strategies[i]
		if strings.Contains(prompt, strategy.line) {
			continue
		}
		variants = append(variants, PromptVariant{
			ID:     fmt.Sprintf("%s_iter%d_%s", stableVariantPrefix(base.ID), iteration, strategy.idSuffix),
			Prompt: prompt + "\n" + strategy.line,
		})
	}
	if len(variants) == 0 {
		variants = append(variants, PromptVariant{
			ID:     fmt.Sprintf("%s_iter%d_stable", stableVariantPrefix(base.ID), iteration),
			Prompt: prompt + "\nKeep the same behavior when evaluation scores do not improve.",
		})
	}
	return variants
}

func stableVariantPrefix(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "variant"
	}
	id = strings.NewReplacer(" ", "_", "\n", "_", "\t", "_").Replace(id)
	return strings.Trim(id, "_")
}
