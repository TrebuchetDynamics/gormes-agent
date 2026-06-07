package scoring

import (
	"math"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/model"
)

func Trace(variantID string, scenario model.Scenario, trace model.Trace, err error) model.Result {
	if err != nil {
		return model.Result{
			VariantID:       variantID,
			Scenario:        scenario.Name,
			TaskSuccess:     false,
			ResponseQuality: 1,
			ResponseScore:   1,
			Error:           err.Error(),
		}
	}

	toolAcc := ToolAccuracy(trace.Tools, scenario.ExpectedTools)
	respQuality := ResponseQuality(trace.Response, scenario)
	success := toolAcc == 1 && respQuality >= 4
	agg := 0.4*BoolToFloat(success) + 0.3*toolAcc + 0.3*(respQuality/5)

	return model.Result{
		VariantID:       variantID,
		Scenario:        scenario.Name,
		TaskSuccess:     success,
		ToolAccuracy:    Round2(toolAcc),
		ResponseQuality: respQuality,
		ResponseScore:   respQuality,
		AggregateScore:  Round2(agg),
	}
}

func ToolAccuracy(actual, expected []string) float64 {
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

func ResponseQuality(response string, scenario model.Scenario) float64 {
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

func Aggregate(results []model.Result) float64 {
	if len(results) == 0 {
		return 0
	}
	var sum float64
	for _, r := range results {
		sum += r.AggregateScore
	}
	return Round2(sum / float64(len(results)))
}

func CloneScenarios(scenarios []model.Scenario) []model.Scenario {
	cloned := make([]model.Scenario, len(scenarios))
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
