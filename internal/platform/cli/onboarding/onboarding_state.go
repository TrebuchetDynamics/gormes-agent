package onboarding

import onboardingseen "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/onboarding/seen"

const (
	// OpenClawResidueCleanupFlag is the stable onboarding.seen key for the
	// one-time OpenClaw residue cleanup banner.
	OpenClawResidueCleanupFlag = onboardingseen.OpenClawResidueCleanupFlag
	OpenClawResidueFlag        = onboardingseen.OpenClawResidueFlag
)

// OnboardingSeen reports whether config has onboarding.seen.<flag> set to
// true. With no flag argument, it checks the OpenClaw residue cleanup flag.
// Malformed or missing onboarding maps are treated as unseen.
func OnboardingSeen(config map[string]any, flags ...string) bool {
	return onboardingseen.OnboardingSeen(config, flags...)
}

// MarkOnboardingSeen sets onboarding.seen.<flag> to true in memory and returns
// the corrected config map. With no flag argument, it marks the OpenClaw
// residue cleanup flag.
func MarkOnboardingSeen(config map[string]any, flags ...string) map[string]any {
	return onboardingseen.MarkOnboardingSeen(config, flags...)
}
