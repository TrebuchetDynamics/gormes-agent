package browser

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/browser/ssrfguard"

const (
	browserSSRFGuardConfigInvalid = ssrfguard.EvidenceConfigInvalid
	browserSSRFPrivateURLBlocked  = ssrfguard.EvidencePrivateURLBlocked
)

// BrowserSSRFGuardBool is a normalized bool-like browser safety config value.
type BrowserSSRFGuardBool = ssrfguard.Bool

// BrowserSSRFGuardOptions are the pure inputs needed before a browser provider
// receives a navigation URL.
type BrowserSSRFGuardOptions = ssrfguard.Options

// BrowserSSRFGuardDecision is the pure pre-navigation cloud safety decision.
type BrowserSSRFGuardDecision = ssrfguard.Decision

// CoerceBrowserSSRFGuardBool normalizes bool-like config values without using
// language truthiness for strings such as "false".
func CoerceBrowserSSRFGuardBool(raw any, fallback bool) BrowserSSRFGuardBool {
	return ssrfguard.CoerceBool(raw, fallback)
}

// CheckBrowserSSRFGuard determines whether rawURL may proceed to its selected
// browser route without starting a browser or resolving DNS.
func CheckBrowserSSRFGuard(taskID, rawURL string, opts BrowserSSRFGuardOptions) BrowserSSRFGuardDecision {
	return ssrfguard.Check(taskID, rawURL, opts)
}
