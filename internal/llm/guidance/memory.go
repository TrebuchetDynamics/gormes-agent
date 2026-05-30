package guidance

type MemoryGuidanceResult struct {
	Guidance string
	Injected bool
	Evidence string
}

func BuildMemoryGuidance(hasMemoryTool bool) MemoryGuidanceResult {
	if !hasMemoryTool {
		return MemoryGuidanceResult{
			Guidance: "",
			Injected: false,
			Evidence: "memory_guidance_suppressed: no memory tool available",
		}
	}
	return MemoryGuidanceResult{
		Guidance: MemoryGuidance,
		Injected: true,
		Evidence: "memory_guidance_injected",
	}
}
