package websearchpolicy

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/urlsafety"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/websitepolicy"
)

// URLSafetyResult combines URL safety and website policy results for web search/browser tools.
type URLSafetyResult struct {
	URL          string
	Allowed      bool
	SSRFReason   string // non-empty if blocked by SSRF
	BlockReason  string // non-empty if blocked by website policy
	PolicyRule   string
	PolicySource string
}

// CheckURL performs a combined URL safety and website policy check.
func CheckURL(rawURL string, safetyPolicy urlsafety.URLSafetyPolicy, policyConfigPath string) *URLSafetyResult {
	result := &URLSafetyResult{
		URL:     rawURL,
		Allowed: true,
	}

	// Step 1: Check URL safety (SSRF, blocklist, allowlist).
	safetyChecker := urlsafety.NewURLSafetyChecker(safetyPolicy)
	urlResult := safetyChecker.CheckURL(rawURL)

	if !urlResult.Safe {
		result.Allowed = false
		result.SSRFReason = urlResult.Reason
		return result
	}

	// Step 2: Check website policy.
	policyResult := websitepolicy.CheckWebsiteAccess(rawURL, policyConfigPath)

	if policyResult != nil {
		result.Allowed = false
		result.BlockReason = policyResult.Message
		result.PolicyRule = policyResult.Rule
		result.PolicySource = policyResult.Source
		return result
	}

	return result
}

// CheckURLStatic is a convenience function using default policies.
func CheckURLStatic(rawURL string) *URLSafetyResult {
	return CheckURL(rawURL, urlsafety.DefaultURLSafetyPolicy(), websitepolicy.DefaultWebsitePolicyConfigPath())
}
