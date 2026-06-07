package optimization

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/evaluation"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/mutation"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/scoring"
)

const (
	Converged    = "converged"
	Budget       = "budget_exhausted"
	PerfectScore = "perfect_score"
)

type Iteration struct {
	Index       int
	Base        model.Variant
	Candidates  []model.Variant
	Evaluations []model.VariantEvaluation
	Selected    model.Variant
	Improved    bool
}

type Result struct {
	Base              model.Variant
	Best              model.Variant
	Iterations        []Iteration
	ImprovementRatio  float64
	TerminationReason string
}

type Optimizer struct {
	evaluator            *evaluation.Evaluator
	variantsPerIteration int
	maxIterations        int
	improvementThreshold float64
	runner               model.Runner
	toolCallsFn          func(context.Context, string) ([]string, error)
}

func New(evaluator *evaluation.Evaluator, toolCallsFn func(context.Context, string) ([]string, error)) *Optimizer {
	optimizer := NewWithRunner(evaluator, nil)
	optimizer.toolCallsFn = toolCallsFn
	optimizer.runner = func(ctx context.Context, variant model.Variant, scenario model.Scenario) (model.Trace, error) {
		if toolCallsFn == nil {
			return model.Trace{Response: scenario.ExpectedOutcome}, nil
		}
		tools, err := toolCallsFn(ctx, variant.Prompt+"\n\nTask: "+scenario.Prompt)
		if err != nil {
			return model.Trace{}, err
		}
		return model.Trace{Tools: tools, Response: scenario.ExpectedOutcome}, nil
	}
	return optimizer
}

func NewWithRunner(evaluator *evaluation.Evaluator, runner model.Runner) *Optimizer {
	return NewWithSettings(evaluator, runner, nil, 4, 50, 0.05)
}

func NewWithSettings(evaluator *evaluation.Evaluator, runner model.Runner, toolCallsFn func(context.Context, string) ([]string, error), variantsPerIteration, maxIterations int, improvementThreshold float64) *Optimizer {
	return &Optimizer{
		evaluator:            evaluator,
		variantsPerIteration: variantsPerIteration,
		maxIterations:        maxIterations,
		improvementThreshold: improvementThreshold,
		runner:               runner,
		toolCallsFn:          toolCallsFn,
	}
}

func (o *Optimizer) Optimize(ctx context.Context, basePrompt string) (*model.Variant, error) {
	result, err := o.OptimizeDetailed(ctx, basePrompt)
	if err != nil {
		return nil, err
	}
	best := result.Best
	return &best, nil
}

func (o *Optimizer) OptimizeDetailed(ctx context.Context, basePrompt string) (Result, error) {
	if o == nil || o.evaluator == nil {
		return Result{}, fmt.Errorf("prompt optimizer requires evaluator")
	}
	runner := o.runner
	if runner == nil {
		runner = func(ctx context.Context, variant model.Variant, scenario model.Scenario) (model.Trace, error) {
			return model.Trace{Response: scenario.ExpectedOutcome}, nil
		}
	}

	base := model.Variant{ID: "base", Prompt: strings.TrimSpace(basePrompt)}
	baseEvaluation, err := o.evaluator.EvaluateVariant(ctx, base, runner)
	if err != nil {
		return Result{}, err
	}
	base.Score = baseEvaluation.AggregateScore
	best := base

	result := Result{
		Base:              base,
		Best:              best,
		TerminationReason: Budget,
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
			return Result{}, err
		}
		selected := model.Variant{}
		if len(evaluations) > 0 {
			selectedEval := evaluations[0]
			selected = model.Variant{ID: selectedEval.VariantID, Prompt: selectedEval.Prompt, Score: selectedEval.AggregateScore}
		}
		improved := selected.Score > best.Score+o.improvementThreshold
		if improved {
			best = selected
			result.Best = best
		}
		result.Iterations = append(result.Iterations, Iteration{
			Index:       i,
			Base:        iterationBase,
			Candidates:  candidates,
			Evaluations: evaluations,
			Selected:    selected,
			Improved:    improved,
		})
		if !improved {
			result.TerminationReason = Converged
			break
		}
		if best.Score >= 1 {
			result.TerminationReason = PerfectScore
			break
		}
	}
	if base.Score > 0 {
		result.ImprovementRatio = scoring.Round2((result.Best.Score - base.Score) / base.Score)
	}
	return result, nil
}

func (o *Optimizer) Mutate(prompt string) model.Variant {
	variants := o.GenerateVariants(model.Variant{ID: "base", Prompt: prompt}, 0)
	if len(variants) == 0 {
		return model.Variant{ID: "mut_empty", Prompt: strings.TrimSpace(prompt)}
	}
	return variants[0]
}

func (o *Optimizer) GenerateVariants(base model.Variant, iteration int) []model.Variant {
	limit := 0
	if o != nil {
		limit = o.variantsPerIteration
	}
	return mutation.GenerateVariants(base, iteration, limit)
}
