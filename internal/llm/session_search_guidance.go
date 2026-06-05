package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance"

type SessionSearchGuidanceResult = guidance.SessionSearchGuidanceResult

func BuildSessionSearchGuidance(enabled bool) SessionSearchGuidanceResult {
	return guidance.BuildSessionSearchGuidance(enabled)
}
