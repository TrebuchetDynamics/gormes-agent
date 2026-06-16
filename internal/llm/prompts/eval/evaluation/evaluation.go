package evaluation

import (
	"context"
	"sort"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/scoring"
)

type Evaluator struct {
	scenarios []model.Scenario
}

func New(scenarios []model.Scenario) *Evaluator {
	return &Evaluator{scenarios: scoring.CloneScenarios(scenarios)}
}

func (e *Evaluator) Evaluate(ctx context.Context, variantID string, toolCalls func(context.Context, string) ([]string, error)) ([]model.Result, error) {
	evaluation, err := e.EvaluateVariant(ctx, model.Variant{ID: variantID}, func(ctx context.Context, variant model.Variant, scenario model.Scenario) (model.Trace, error) {
		tools, err := toolCalls(ctx, scenario.Prompt)
		if err != nil {
			return model.Trace{}, err
		}
		return model.Trace{Tools: tools, Response: scenario.ExpectedOutcome}, nil
	})
	if err != nil {
		return nil, err
	}
	return evaluation.Results, nil
}

func (e *Evaluator) EvaluateVariant(ctx context.Context, variant model.Variant, runner model.Runner) (model.VariantEvaluation, error) {
	if runner == nil {
		return model.VariantEvaluation{VariantID: variant.ID, Prompt: variant.Prompt}, nil
	}

	results := make([]model.Result, 0, len(e.scenarios))
	for _, scenario := range e.scenarios {
		trace, err := runner(ctx, variant, scenario)
		result := scoring.Trace(variant.ID, scenario, trace, err)
		results = append(results, result)
	}

	return model.VariantEvaluation{
		VariantID:      variant.ID,
		Prompt:         variant.Prompt,
		Results:        results,
		AggregateScore: scoring.Aggregate(results),
	}, nil
}

func (e *Evaluator) CompareVariants(ctx context.Context, variants []model.Variant, runner model.Runner) ([]model.VariantEvaluation, error) {
	evaluations := make([]model.VariantEvaluation, 0, len(variants))
	for _, variant := range variants {
		evaluation, err := e.EvaluateVariant(ctx, variant, runner)
		if err != nil {
			return nil, err
		}
		evaluations = append(evaluations, evaluation)
	}
	sort.SliceStable(evaluations, func(i, j int) bool {
		if evaluations[i].AggregateScore == evaluations[j].AggregateScore {
			return evaluations[i].VariantID < evaluations[j].VariantID
		}
		return evaluations[i].AggregateScore > evaluations[j].AggregateScore
	})
	return evaluations, nil
}
