package guidance

// SessionSearchGuidanceResult is the result of BuildSessionSearchGuidance.
// Deprecated: use GuidanceSwitchResult. This type alias is preserved for
// backward compatibility.
type SessionSearchGuidanceResult = GuidanceSwitchResult

func BuildSessionSearchGuidance(enabled bool) SessionSearchGuidanceResult {
	return buildGuidanceSwitch(SessionSearchGuidance, enabled,
		"session_search_guidance_suppressed: session search disabled",
		"session_search_guidance_injected",
	)
}
