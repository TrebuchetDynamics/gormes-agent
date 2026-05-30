package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/browser"

const (
	browserSSRFGuardConfigInvalid = "ssrf_guard_config_invalid"
	browserSSRFPrivateURLBlocked  = "private_url_blocked"
)

// BrowserSSRFGuardBool is a normalized bool-like browser safety config value.
type BrowserSSRFGuardBool = browser.BrowserSSRFGuardBool

// BrowserSSRFGuardOptions are the pure inputs needed before a browser provider
// receives a navigation URL.
type BrowserSSRFGuardOptions = browser.BrowserSSRFGuardOptions

// BrowserSSRFGuardDecision is the pure pre-navigation cloud safety decision.
type BrowserSSRFGuardDecision = browser.BrowserSSRFGuardDecision

// CoerceBrowserSSRFGuardBool normalizes bool-like config values without using
// language truthiness for strings such as "false".
func CoerceBrowserSSRFGuardBool(raw any, fallback bool) BrowserSSRFGuardBool {
	return browser.CoerceBrowserSSRFGuardBool(raw, fallback)
}

// CheckBrowserSSRFGuard determines whether rawURL may proceed to its selected
// browser route without starting a browser or resolving DNS.
func CheckBrowserSSRFGuard(taskID, rawURL string, opts BrowserSSRFGuardOptions) BrowserSSRFGuardDecision {
	return browser.CheckBrowserSSRFGuard(taskID, rawURL, opts)
}
