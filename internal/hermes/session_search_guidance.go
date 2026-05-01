package hermes

type SessionSearchGuidanceResult struct {
	Guidance string
	Injected bool
	Evidence string
}

func BuildSessionSearchGuidance(enabled bool) SessionSearchGuidanceResult {
	if !enabled {
		return SessionSearchGuidanceResult{
			Guidance: "",
			Injected: false,
			Evidence: "session_search_guidance_suppressed: session search disabled",
		}
	}
	return SessionSearchGuidanceResult{
		Guidance: SessionSearchGuidance,
		Injected: true,
		Evidence: "session_search_guidance_injected",
	}
}
