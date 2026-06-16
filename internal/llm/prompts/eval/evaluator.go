package eval

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/evaluation"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/scoring"
)

type EvalScenario = model.Scenario

type EvalTrace = model.Trace

type EvalResult = model.Result

type VariantEvaluation = model.VariantEvaluation

type PromptEvalRunner = model.Runner

type PromptEvaluator = evaluation.Evaluator

func NewPromptEvaluator(scenarios []EvalScenario) *PromptEvaluator {
	return evaluation.New(scenarios)
}

func ScoreEvalTrace(variantID string, scenario EvalScenario, trace EvalTrace, err error) EvalResult {
	return scoring.Trace(variantID, scenario, trace, err)
}

func ComputeToolAccuracy(actual, expected []string) float64 {
	return scoring.ToolAccuracy(actual, expected)
}

func ComputeResponseQuality(response string, scenario EvalScenario) float64 {
	return scoring.ResponseQuality(response, scenario)
}

func BoolToFloat(b bool) float64 {
	return scoring.BoolToFloat(b)
}

func AggregateScore(results []EvalResult) float64 {
	return scoring.Aggregate(results)
}

func CloneEvalScenarios(scenarios []EvalScenario) []EvalScenario {
	return scoring.CloneScenarios(scenarios)
}

func Round2(v float64) float64 {
	return scoring.Round2(v)
}
