package safety

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/websearchpolicy"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/websitepolicy"
)

type WebsitePolicyError = websitepolicy.WebsitePolicyError
type WebsitePolicyResult = websitepolicy.WebsitePolicyResult
type WebsitePolicyRule = websitepolicy.WebsitePolicyRule
type WebsitePolicy = websitepolicy.WebsitePolicy
type CheckWebsiteAccessResult = websitepolicy.CheckWebsiteAccessResult

func DefaultWebsitePolicyConfigPath() string {
	return websitepolicy.DefaultWebsitePolicyConfigPath()
}

func LoadWebsitePolicy(configPath string) (*WebsitePolicy, error) {
	return websitepolicy.LoadWebsitePolicy(configPath)
}

func InvalidateWebsitePolicyCache() {
	websitepolicy.InvalidateWebsitePolicyCache()
}

func CheckWebsiteAccess(rawURL string, configPath string) *WebsitePolicyResult {
	return websitepolicy.CheckWebsiteAccess(rawURL, configPath)
}

func NewWebsitePolicy(enabled bool, domains []string, sharedFiles []string, source string) *WebsitePolicy {
	return websitepolicy.NewWebsitePolicy(enabled, domains, sharedFiles, source)
}

// --------------------------------------------------------------------------
// Web search / browser integration helpers
// --------------------------------------------------------------------------

// WebSearchURLSafetyResult combines URL safety and website policy results.
type WebSearchURLSafetyResult = websearchpolicy.URLSafetyResult

// CheckWebSearchURL performs a combined URL safety and website policy check.
// This is the main integration point for web_search and browser tools.
func CheckWebSearchURL(rawURL string, safetyPolicy URLSafetyPolicy, policyConfigPath string) *WebSearchURLSafetyResult {
	return websearchpolicy.CheckURL(rawURL, safetyPolicy, policyConfigPath)
}

// CheckWebSearchURLStatic is a convenience function using default policies.
func CheckWebSearchURLStatic(rawURL string) *WebSearchURLSafetyResult {
	return websearchpolicy.CheckURLStatic(rawURL)
}
