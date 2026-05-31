package prompts

import prompteval "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval"

type EvalScenario = prompteval.EvalScenario
type EvalTrace = prompteval.EvalTrace
type EvalResult = prompteval.EvalResult
type VariantEvaluation = prompteval.VariantEvaluation
type PromptEvalRunner = prompteval.PromptEvalRunner
type PromptEvaluator = prompteval.PromptEvaluator

func NewPromptEvaluator(scenarios []EvalScenario) *PromptEvaluator {
	return prompteval.NewPromptEvaluator(scenarios)
}

func ScoreEvalTrace(variantID string, scenario EvalScenario, trace EvalTrace, err error) EvalResult {
	return prompteval.ScoreEvalTrace(variantID, scenario, trace, err)
}

func ComputeToolAccuracy(actual, expected []string) float64 {
	return prompteval.ComputeToolAccuracy(actual, expected)
}

func ComputeResponseQuality(response string, scenario EvalScenario) float64 {
	return prompteval.ComputeResponseQuality(response, scenario)
}

func BoolToFloat(b bool) float64 {
	return prompteval.BoolToFloat(b)
}

func AggregateScore(results []EvalResult) float64 {
	return prompteval.AggregateScore(results)
}

func CloneEvalScenarios(scenarios []EvalScenario) []EvalScenario {
	return prompteval.CloneEvalScenarios(scenarios)
}

func Round2(v float64) float64 {
	return prompteval.Round2(v)
}
