package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance"

type MemoryGuidanceResult = guidance.MemoryGuidanceResult

func BuildMemoryGuidance(hasMemoryTool bool) MemoryGuidanceResult {
	return guidance.BuildMemoryGuidance(hasMemoryTool)
}
