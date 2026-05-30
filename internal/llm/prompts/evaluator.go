package prompts

import (
	"context"
	"math"
	"sort"
	"strings"
)

type EvalScenario struct {
	Name                   string
	Prompt                 string
	ExpectedTools          []string
	ExpectedOutcome        string
	RequiredResponseTerms  []string
	ForbiddenResponseTerms []string
}

type EvalTrace struct {
	Tools    []string
	Response string
}

type EvalResult struct {
	VariantID       string
	Scenario        string
	TaskSuccess     bool
	ToolAccuracy    float64
	ResponseQuality float64
	ResponseScore   float64
	AggregateScore  float64
	Error           string
}

type VariantEvaluation struct {
	VariantID      string
	Prompt         string
	Results        []EvalResult
	AggregateScore float64
}

type PromptEvalRunner func(context.Context, PromptVariant, EvalScenario) (EvalTrace, error)

type PromptEvaluator struct {
	scenarios []EvalScenario
}

func NewPromptEvaluator(scenarios []EvalScenario) *PromptEvaluator {
	return &PromptEvaluator{scenarios: CloneEvalScenarios(scenarios)}
}

func (e *PromptEvaluator) Evaluate(ctx context.Context, variantID string, toolCalls func(context.Context, string) ([]string, error)) ([]EvalResult, error) {
	evaluation, err := e.EvaluateVariant(ctx, PromptVariant{ID: variantID}, func(ctx context.Context, variant PromptVariant, scenario EvalScenario) (EvalTrace, error) {
		tools, err := toolCalls(ctx, scenario.Prompt)
		if err != nil {
			return EvalTrace{}, err
		}
		return EvalTrace{Tools: tools, Response: scenario.ExpectedOutcome}, nil
	})
	if err != nil {
		return nil, err
	}
	return evaluation.Results, nil
}

func (e *PromptEvaluator) EvaluateVariant(ctx context.Context, variant PromptVariant, runner PromptEvalRunner) (VariantEvaluation, error) {
	if runner == nil {
		return VariantEvaluation{VariantID: variant.ID, Prompt: variant.Prompt}, nil
	}

	results := make([]EvalResult, 0, len(e.scenarios))
	for _, scenario := range e.scenarios {
		trace, err := runner(ctx, variant, scenario)
		result := ScoreEvalTrace(variant.ID, scenario, trace, err)
		results = append(results, result)
	}

	return VariantEvaluation{
		VariantID:      variant.ID,
		Prompt:         variant.Prompt,
		Results:        results,
		AggregateScore: AggregateScore(results),
	}, nil
}

func (e *PromptEvaluator) CompareVariants(ctx context.Context, variants []PromptVariant, runner PromptEvalRunner) ([]VariantEvaluation, error) {
	evaluations := make([]VariantEvaluation, 0, len(variants))
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

func ScoreEvalTrace(variantID string, scenario EvalScenario, trace EvalTrace, err error) EvalResult {
	if err != nil {
		return EvalResult{
			VariantID:       variantID,
			Scenario:        scenario.Name,
			TaskSuccess:     false,
			ResponseQuality: 1,
			ResponseScore:   1,
			Error:           err.Error(),
		}
	}

	toolAcc := ComputeToolAccuracy(trace.Tools, scenario.ExpectedTools)
	respQuality := ComputeResponseQuality(trace.Response, scenario)
	success := toolAcc == 1 && respQuality >= 4
	agg := 0.4*BoolToFloat(success) + 0.3*toolAcc + 0.3*(respQuality/5)

	return EvalResult{
		VariantID:       variantID,
		Scenario:        scenario.Name,
		TaskSuccess:     success,
		ToolAccuracy:    Round2(toolAcc),
		ResponseQuality: respQuality,
		ResponseScore:   respQuality,
		AggregateScore:  Round2(agg),
	}
}

func ComputeToolAccuracy(actual, expected []string) float64 {
	if len(expected) == 0 {
		if len(actual) == 0 {
			return 1.0
		}
		return 0.0
	}
	expectedSet := make(map[string]bool)
	for _, t := range expected {
		expectedSet[t] = true
	}
	matches := 0
	for _, t := range actual {
		if expectedSet[t] {
			matches++
		}
	}
	return float64(matches) / float64(len(expected))
}

func ComputeResponseQuality(response string, scenario EvalScenario) float64 {
	required := make([]string, 0, 1+len(scenario.RequiredResponseTerms))
	if strings.TrimSpace(scenario.ExpectedOutcome) != "" {
		required = append(required, scenario.ExpectedOutcome)
	}
	required = append(required, scenario.RequiredResponseTerms...)
	if len(required) == 0 && len(scenario.ForbiddenResponseTerms) == 0 {
		return 5
	}

	response = strings.ToLower(response)
	matches := 0
	for _, term := range required {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" || strings.Contains(response, term) {
			matches++
		}
	}
	var score float64
	if len(required) == 0 {
		score = 5
	} else {
		score = 1 + 4*(float64(matches)/float64(len(required)))
	}
	for _, term := range scenario.ForbiddenResponseTerms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term != "" && strings.Contains(response, term) {
			score--
		}
	}
	if score < 1 {
		return 1
	}
	if score > 5 {
		return 5
	}
	return Round2(score)
}

func BoolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func AggregateScore(results []EvalResult) float64 {
	if len(results) == 0 {
		return 0
	}
	var sum float64
	for _, r := range results {
		sum += r.AggregateScore
	}
	return Round2(sum / float64(len(results)))
}

func CloneEvalScenarios(scenarios []EvalScenario) []EvalScenario {
	cloned := make([]EvalScenario, len(scenarios))
	for i, scenario := range scenarios {
		cloned[i] = scenario
		cloned[i].ExpectedTools = append([]string(nil), scenario.ExpectedTools...)
		cloned[i].RequiredResponseTerms = append([]string(nil), scenario.RequiredResponseTerms...)
		cloned[i].ForbiddenResponseTerms = append([]string(nil), scenario.ForbiddenResponseTerms...)
	}
	return cloned
}

func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}
