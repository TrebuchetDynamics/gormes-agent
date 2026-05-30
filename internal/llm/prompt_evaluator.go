package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts"

type EvalScenario = prompts.EvalScenario
type EvalTrace = prompts.EvalTrace
type EvalResult = prompts.EvalResult
type VariantEvaluation = prompts.VariantEvaluation
type PromptEvalRunner = prompts.PromptEvalRunner
type PromptEvaluator = prompts.PromptEvaluator

func NewPromptEvaluator(scenarios []EvalScenario) *PromptEvaluator {
	return prompts.NewPromptEvaluator(scenarios)
}

func scoreEvalTrace(variantID string, scenario EvalScenario, trace EvalTrace, err error) EvalResult {
	return prompts.ScoreEvalTrace(variantID, scenario, trace, err)
}

func computeToolAccuracy(actual, expected []string) float64 {
	return prompts.ComputeToolAccuracy(actual, expected)
}

func computeResponseQuality(response string, scenario EvalScenario) float64 {
	return prompts.ComputeResponseQuality(response, scenario)
}

func boolToFloat(b bool) float64 {
	return prompts.BoolToFloat(b)
}

func AggregateScore(results []EvalResult) float64 {
	return prompts.AggregateScore(results)
}

func cloneEvalScenarios(scenarios []EvalScenario) []EvalScenario {
	return prompts.CloneEvalScenarios(scenarios)
}

func round2(v float64) float64 {
	return prompts.Round2(v)
}
