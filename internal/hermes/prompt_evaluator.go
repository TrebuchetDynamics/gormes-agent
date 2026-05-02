package hermes

import (
	"context"
	"math"
)

type EvalScenario struct {
	Name            string
	Prompt          string
	ExpectedTools   []string
	ExpectedOutcome string
}

type EvalResult struct {
	Scenario       string
	TaskSuccess    bool
	ToolAccuracy   float64
	ResponseScore  float64
	AggregateScore float64
}

type PromptEvaluator struct {
	scenarios []EvalScenario
}

func NewPromptEvaluator(scenarios []EvalScenario) *PromptEvaluator {
	return &PromptEvaluator{scenarios: scenarios}
}

func (e *PromptEvaluator) Evaluate(ctx context.Context, variantID string, toolCalls func(context.Context, string) ([]string, error)) ([]EvalResult, error) {
	var results []EvalResult
	for _, s := range e.scenarios {
		tools, err := toolCalls(ctx, s.Prompt)
		if err != nil {
			results = append(results, EvalResult{Scenario: s.Name, TaskSuccess: false})
			continue
		}
		toolAcc := computeToolAccuracy(tools, s.ExpectedTools)
		respScore := 0.7
		agg := 0.4*float64(boolToFloat(toolAcc > 0.5)) + 0.3*toolAcc + 0.3*respScore

		result := EvalResult{
			Scenario:       s.Name,
			TaskSuccess:    toolAcc > 0.5,
			ToolAccuracy:   toolAcc,
			ResponseScore:  respScore,
			AggregateScore: math.Round(agg*100) / 100,
		}
		results = append(results, result)
	}
	return results, nil
}

func computeToolAccuracy(actual, expected []string) float64 {
	if len(expected) == 0 {
		return 1.0
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

func boolToFloat(b bool) float64 {
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
	return math.Round(sum/float64(len(results))*100) / 100
}
