package websearchpolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/urlsafety"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/websitepolicy"
)

func TestCheckURLStopsAtURLSafetyFailure(t *testing.T) {
	policy := urlsafety.URLSafetyPolicy{Enabled: true}

	result := CheckURL("not a url", policy, "")
	if result.Allowed {
		t.Fatalf("CheckURL invalid host allowed = true, want false")
	}
	if result.SSRFReason == "" {
		t.Fatalf("CheckURL invalid host SSRFReason empty, result = %+v", result)
	}
	if result.BlockReason != "" {
		t.Fatalf("CheckURL URL-safety failure BlockReason = %q, want empty", result.BlockReason)
	}
}

func TestCheckURLAppliesWebsitePolicyAfterURLSafety(t *testing.T) {
	websitepolicy.InvalidateWebsitePolicyCache()
	configPath := filepath.Join(t.TempDir(), "policy.toml")
	if err := os.WriteFile(configPath, []byte("[security.website_blocklist]\nenabled = true\ndomains = [\"blocked.example\"]\n"), 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	result := CheckURL("https://blocked.example/page", urlsafety.URLSafetyPolicy{Enabled: false}, configPath)
	if result.Allowed {
		t.Fatalf("CheckURL blocked site allowed = true, want false")
	}
	if !strings.Contains(result.BlockReason, "blocked.example") {
		t.Fatalf("CheckURL BlockReason = %q, want blocked.example", result.BlockReason)
	}
	if result.PolicyRule != "blocked.example" || result.PolicySource == "" {
		t.Fatalf("CheckURL policy evidence = rule %q source %q, want populated", result.PolicyRule, result.PolicySource)
	}
}
