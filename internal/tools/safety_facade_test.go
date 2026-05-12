package tools

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety"
)

func TestSafetyFacadeContractsStayAliased(t *testing.T) {
	policy := URLSafetyPolicy{
		Enabled: true,
		Blocklist: []URLSafetyBlocklistEntry{{
			Pattern:  "example.invalid",
			Category: URLSafetyCategoryPhishing,
			Source:   "test",
		}},
	}
	var safetyPolicy safety.URLSafetyPolicy = policy

	checker := NewURLSafetyChecker(safetyPolicy)
	var safetyChecker *safety.URLSafetyChecker = checker
	result := safetyChecker.CheckURL("https://example.invalid/path")
	var facadeResult URLSafetyResult = result
	if facadeResult.Safe {
		t.Fatal("expected facade safety checker to preserve blocklist behavior")
	}

	websitePolicy := NewWebsitePolicy(true, []string{"blocked.example"}, nil, "test")
	var safetyWebsitePolicy *safety.WebsitePolicy = websitePolicy
	if safetyWebsitePolicy.IsWebsiteBlocked("https://blocked.example/page") == nil {
		t.Fatal("expected facade website policy to preserve blocked-domain behavior")
	}
}
