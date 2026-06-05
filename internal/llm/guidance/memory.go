package guidance

// MemoryGuidanceResult is the result of BuildMemoryGuidance.
// Deprecated: use GuidanceSwitchResult. This type alias is preserved for
// backward compatibility.
type MemoryGuidanceResult = GuidanceSwitchResult

func BuildMemoryGuidance(hasMemoryTool bool) MemoryGuidanceResult {
	return buildGuidanceSwitch(MemoryGuidance, hasMemoryTool,
		"memory_guidance_suppressed: no memory tool available",
		"memory_guidance_injected",
	)
}
