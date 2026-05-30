package guidance

// GuidanceSwitchResult is a generic result type for conditional guidance
// injection. It carries the guidance text if injected, along with evidence
// strings for diagnostics. Both MemoryGuidanceResult and
// SessionSearchGuidanceResult are type aliases of this struct.
type GuidanceSwitchResult struct {
	Guidance string
	Injected bool
	Evidence string
}

// buildGuidanceSwitch constructs a GuidanceSwitchResult. When enabled is
// true, the provided guidance text is returned with injected=true and the
// injectedEvidence string. When enabled is false, an empty result with the
// suppressedEvidence string is returned. Guidance is empty when not injected
// so callers can omit empty blocks.
func buildGuidanceSwitch(guidance string, enabled bool, suppressedEvidence, injectedEvidence string) GuidanceSwitchResult {
	if !enabled {
		return GuidanceSwitchResult{
			Evidence: suppressedEvidence,
		}
	}
	return GuidanceSwitchResult{
		Guidance: guidance,
		Injected: true,
		Evidence: injectedEvidence,
	}
}
