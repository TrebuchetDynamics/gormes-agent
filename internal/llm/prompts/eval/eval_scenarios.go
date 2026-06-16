package eval

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/scenarios"

func DefaultPromptEvalScenarios() []EvalScenario {
	return scenarios.Default()
}
