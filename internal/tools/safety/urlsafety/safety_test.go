package urlsafety

import (
	"net"
	"testing"
)

// --------------------------------------------------------------------------
// IsPrivateBrowserHost pattern matching
// --------------------------------------------------------------------------

func TestMatchHostAgainstRule_ExactMatch(t *testing.T) {
	tests := []struct {
		host    string
		pattern string
		want    bool
	}{
		{"example.com", "example.com", true},
		{"www.example.com", "example.com", true},
		{"sub.example.com", "example.com", true},
		{"notexample.com", "example.com", false},
		{"evil.example.com", "example.com", true},
		{"EXAMPLE.COM", "example.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.host+"_"+tt.pattern, func(t *testing.T) {
			got := matchHostAgainstRule(tt.host, tt.pattern)
			if got != tt.want {
				t.Fatalf("matchHostAgainstRule(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchHostAgainstRule_WildcardSuffix(t *testing.T) {
	tests := []struct {
		host    string
		pattern string
		want    bool
	}{
		{"evil.example.com", "*.example.com", true},
		{"sub.evil.example.com", "*.example.com", true},
		{"deep.sub.evil.example.com", "*.example.com", true},
		{"example.com", "*.example.com", false},
		{"notexample.com", "*.example.com", false},
		{"EVIL.EXAMPLE.COM", "*.example.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.host+"_"+tt.pattern, func(t *testing.T) {
			got := matchHostAgainstRule(tt.host, tt.pattern)
			if got != tt.want {
				t.Fatalf("matchHostAgainstRule(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchHostAgainstRule_SuffixMatch(t *testing.T) {
	tests := []struct {
		host    string
		pattern string
		want    bool
	}{
		{"sub.example.com", "example.com", true},
		{"deep.sub.example.com", "example.com", true},
		{"notexample.com", "example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := matchHostAgainstRule(tt.host, tt.pattern)
			if got != tt.want {
				t.Fatalf("matchHostAgainstRule(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

// --------------------------------------------------------------------------
// Host normalization
// --------------------------------------------------------------------------

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Example.COM", "example.com"},
		{"www.Example.COM", "example.com"},
		{"Example.COM.", "example.com"},
		{"  Example.COM  ", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeHost(tt.input)
			if got != tt.expected {
				t.Fatalf("normalizeHost(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// --------------------------------------------------------------------------
// URL extraction (includes port stripping)
// --------------------------------------------------------------------------

func TestExtractHostFromURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/path", "example.com"},
		{"http://www.example.com/", "example.com"},
		{"example.com:8080", "example.com"},
		{"sub.example.com", "sub.example.com"},
		{"192.168.1.1:3000", "192.168.1.1"},
		{"[::1]:8080", "::1"},
		{"[2001:db8::1]:8080", "2001:db8::1"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractHostFromURL(tt.input)
			if got != tt.expected {
				t.Fatalf("extractHostFromURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// --------------------------------------------------------------------------
// Blocklist rule normalization
// --------------------------------------------------------------------------

func TestNormalizeBlocklistRule(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "example.com"},
		{"https://example.com/path", "example.com"},
		{"http://www.example.com/", "example.com"},
		{"example.com/", "example.com"},
		{"EXAMPLE.COM", "example.com"},
		{"# this is a comment", ""},
		{"", ""},
		{"   ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeBlocklistRule(tt.input)
			if got != tt.expected {
				t.Fatalf("NormalizeBlocklistRule(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// --------------------------------------------------------------------------
// Bool parsing
// --------------------------------------------------------------------------

func TestParseBoolLike(t *testing.T) {
	tests := []struct {
		name     string
		raw      any
		fallback bool
		want     bool
	}{
		{"true_bool", true, false, true},
		{"false_bool", false, true, false},
		{"nil_fallback_true", nil, true, true},
		{"nil_fallback_false", nil, false, false},
		{"string_true", "true", false, true},
		{"string_yes", "yes", false, true},
		{"string_1", "1", false, true},
		{"string_on", "on", false, true},
		{"string_false", "false", true, false},
		{"string_no", "no", true, false},
		{"string_0", "0", true, false},
		{"string_off", "off", true, false},
		{"string_unknown_fallback", "unknown", true, true},
		{"int_1", 1, false, true},
		{"int_0", 0, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseBoolLike(tt.raw, tt.fallback)
			if got != tt.want {
				t.Fatalf("ParseBoolLike(%v, %v) = %v, want %v", tt.raw, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestParseBoolLikeWithEvidence(t *testing.T) {
	tests := []struct {
		name         string
		raw          any
		fallback     bool
		wantValue    bool
		wantEvidence string
	}{
		{"valid_true", "true", false, true, ""},
		{"valid_false", "false", true, false, ""},
		{"invalid_string", "invalid", true, true, "invalid_bool_string"},
		{"quoted_true", `"true"`, false, true, ""},
		{"single_quoted_false", "'false'", true, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotEvidence := ParseBoolLikeWithEvidence(tt.raw, tt.fallback)
			if gotValue != tt.wantValue {
				t.Fatalf("ParseBoolLikeWithEvidence(%v, %v).Value = %v, want %v", tt.raw, tt.fallback, gotValue, tt.wantValue)
			}
			if gotEvidence != tt.wantEvidence {
				t.Fatalf("ParseBoolLikeWithEvidence(%v, %v).Evidence = %q, want %q", tt.raw, tt.fallback, gotEvidence, tt.wantEvidence)
			}
		})
	}
}

// --------------------------------------------------------------------------
// URLSafetyChecker basic tests
// --------------------------------------------------------------------------

func TestURLSafetyChecker_DefaultPolicyAllowsPublicURL(t *testing.T) {
	checker := NewURLSafetyChecker(DefaultURLSafetyPolicy())
	result := checker.CheckURL("https://example.com")
	if !result.Safe {
		t.Fatalf("CheckURL(example.com) = Safe=false, want Safe=true, result=%+v", result)
	}
}

func TestURLSafetyChecker_BlocksBlockedHostname(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	policy.Blocklist = append(policy.Blocklist, URLSafetyBlocklistEntry{
		Pattern:  "metadata.google.internal",
		Category: URLSafetyCategorySSRF,
		Source:   "test",
	})
	checker := NewURLSafetyChecker(policy)
	result := checker.CheckURL("https://metadata.google.internal/computeMetadata/v1/")
	if result.Safe {
		t.Fatalf("CheckURL(metadata.google.internal) = Safe=true, want Safe=false")
	}
	if result.Category != URLSafetyCategorySSRF {
		t.Fatalf("Category = %v, want URLSafetyCategorySSRF", result.Category)
	}
}

func TestURLSafetyChecker_AllowlistTakesPrecedence(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	// Block all of .example.com but allow trusted.example.com.
	policy.Blocklist = append(policy.Blocklist, URLSafetyBlocklistEntry{
		Pattern:  "*.example.com",
		Category: URLSafetyCategoryPhishing,
		Source:   "test",
	})
	policy.Allowlist = append(policy.Allowlist, URLSafetyAllowlistEntry{
		Pattern: "trusted.example.com",
		Source:  "test",
	})
	checker := NewURLSafetyChecker(policy)
	result := checker.CheckURL("https://trusted.example.com/safe")
	if !result.Safe {
		t.Fatalf("CheckURL(trusted.example.com) = Safe=false, want Safe=true (allowlist takes precedence)")
	}
}

func TestURLSafetyChecker_PrivateURLBlocked(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	policy.AllowPrivateURLs = false
	checker := NewURLSafetyChecker(policy)

	tests := []struct {
		name string
		url  string
	}{
		{"localhost", "http://localhost:3000/"},
		{"rfc1918_10", "http://10.0.0.1/"},
		{"rfc1918_192", "http://192.168.1.1/"},
		{"rfc1918_172", "http://172.16.0.1/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.CheckURL(tt.url)
			if result.Safe {
				t.Fatalf("CheckURL(%q) = Safe=true, want Safe=false", tt.url)
			}
			if result.Category != URLSafetyCategoryPrivate && result.Category != URLSafetyCategorySSRF {
				t.Fatalf("Category = %v, want URLSafetyCategoryPrivate or URLSafetyCategorySSRF", result.Category)
			}
		})
	}
}

func TestURLSafetyChecker_AllowPrivateURLsToggle(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	policy.AllowPrivateURLs = true
	checker := NewURLSafetyChecker(policy)

	result := checker.CheckURL("http://192.168.1.1/private")
	if !result.Safe {
		t.Fatalf("CheckURL(private) with AllowPrivateURLs=true = Safe=false, want Safe=true")
	}
}

func TestURLSafetyChecker_CloudMetadataAlwaysBlocked(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	policy.AllowPrivateURLs = true
	checker := NewURLSafetyChecker(policy)

	tests := []struct {
		url string
	}{
		{"http://169.254.169.254/latest/meta-data/"},
		{"https://metadata.google.internal/computeMetadata/v1/"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := checker.CheckURL(tt.url)
			if result.Safe {
				t.Fatalf("CheckURL(%q) with AllowPrivateURLs=true = Safe=true, want Safe=false (cloud metadata always blocked)", tt.url)
			}
			if result.Category != URLSafetyCategorySSRF {
				t.Fatalf("Category = %v, want URLSafetyCategorySSRF", result.Category)
			}
		})
	}
}

// --------------------------------------------------------------------------
// LoadBlocklistFromLines
// --------------------------------------------------------------------------

func TestLoadBlocklistFromLines(t *testing.T) {
	lines := []string{
		"evil.com",
		"# comment line",
		"malware.net",
		"   ",
		"adult-site.org",
	}
	entries := LoadBlocklistFromLines(lines, URLSafetyCategoryMalware, "test_source")
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	if entries[0].Pattern != "evil.com" || entries[0].Category != URLSafetyCategoryMalware || entries[0].Source != "test_source" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Pattern != "malware.net" || entries[1].Category != URLSafetyCategoryMalware || entries[1].Source != "test_source" {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}
}

// --------------------------------------------------------------------------
// Cache behavior
// --------------------------------------------------------------------------

func TestURLSafetyChecker_CacheInvalidation(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	checker := NewURLSafetyChecker(policy)

	// First check — populates cache.
	result1 := checker.CheckURL("https://example.com")
	if !result1.Safe {
		t.Fatalf("first check failed: %+v", result1)
	}

	// Add blocklist entry — should invalidate cache.
	checker.AddBlocklistEntry("example.com", URLSafetyCategoryPhishing, "test")

	// Now example.com should be blocked.
	result2 := checker.CheckURL("https://example.com")
	if result2.Safe {
		t.Fatalf("CheckURL(example.com) after blocklist add = Safe=true, want Safe=false")
	}
}

func TestURLSafetyChecker_CacheReturnsSameResult(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	checker := NewURLSafetyChecker(policy)

	result1 := checker.CheckURL("https://example.com")
	result2 := checker.CheckURL("https://example.com")

	if result1.Safe != result2.Safe {
		t.Fatalf("inconsistent results from cache: %v vs %v", result1.Safe, result2.Safe)
	}
}

// --------------------------------------------------------------------------
// Int parsing
// --------------------------------------------------------------------------

func TestParseIntLike(t *testing.T) {
	tests := []struct {
		name     string
		raw      any
		fallback int
		want     int
	}{
		{"int", 42, 0, 42},
		{"int64", int64(100), 0, 100},
		{"string", "123", 0, 123},
		{"invalid_string", "abc", 99, 99},
		{"nil", nil, 99, 99},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseIntLike(tt.raw, tt.fallback)
			if got != tt.want {
				t.Fatalf("ParseIntLike(%v, %v) = %v, want %v", tt.raw, tt.fallback, got, tt.want)
			}
		})
	}
}

// --------------------------------------------------------------------------
// SSRF protection — IP classification
// --------------------------------------------------------------------------

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"loopback_v4", "127.0.0.1", true},
		{"loopback_v6", "::1", true},
		{"private_10", "10.0.0.1", true},
		{"private_192", "192.168.1.1", true},
		{"private_172", "172.16.0.1", true},
		{"link_local", "169.254.1.1", true},
		{"cgnat", "100.64.0.1", true},
		{"public", "8.8.8.8", false},
		{"multicast", "224.0.0.1", true},
		{"unspecified", "0.0.0.0", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("could not parse IP: %s", tt.ip)
			}
			got := isBlockedIP(ip)
			if got != tt.want {
				t.Fatalf("isBlockedIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

// --------------------------------------------------------------------------
// CheckURLStatic convenience function
// --------------------------------------------------------------------------

func TestCheckURLStatic(t *testing.T) {
	result := CheckURLStatic("https://example.com")
	if !result.Safe {
		t.Fatalf("CheckURLStatic(example.com) = Safe=false, want Safe=true")
	}
}

// --------------------------------------------------------------------------
// GlobalAllowPrivateURLs
// --------------------------------------------------------------------------

func TestGlobalAllowPrivateURLs_EnvVarTrue(t *testing.T) {
	ResetAllowPrivateCache()
	t.Setenv("GORMES_ALLOW_PRIVATE_URLS", "true")
	got := GlobalAllowPrivateURLs()
	if !got {
		t.Fatalf("GlobalAllowPrivateURLs() with GORMES_ALLOW_PRIVATE_URLS=true = false, want true")
	}
	ResetAllowPrivateCache() // Clean up.
}

func TestGlobalAllowPrivateURLs_EnvVarFalse(t *testing.T) {
	ResetAllowPrivateCache()
	t.Setenv("GORMES_ALLOW_PRIVATE_URLS", "false")
	got := GlobalAllowPrivateURLs()
	if got {
		t.Fatalf("GlobalAllowPrivateURLs() with GORMES_ALLOW_PRIVATE_URLS=false = true, want false")
	}
	ResetAllowPrivateCache() // Clean up.
}

// --------------------------------------------------------------------------
// Blocklist entries with different categories
// --------------------------------------------------------------------------

func TestURLSafetyChecker_CategoryPhishing(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	policy.Blocklist = append(policy.Blocklist, URLSafetyBlocklistEntry{
		Pattern:  "phishing-site.com",
		Category: URLSafetyCategoryPhishing,
		Source:   "test",
	})
	checker := NewURLSafetyChecker(policy)
	result := checker.CheckURL("https://phishing-site.com/")
	if result.Safe {
		t.Fatalf("CheckURL(phishing-site.com) = Safe=true, want Safe=false")
	}
	if result.Category != URLSafetyCategoryPhishing {
		t.Fatalf("Category = %v, want URLSafetyCategoryPhishing", result.Category)
	}
}

func TestURLSafetyChecker_CategoryMalware(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	policy.Blocklist = append(policy.Blocklist, URLSafetyBlocklistEntry{
		Pattern:  "malware-site.com",
		Category: URLSafetyCategoryMalware,
		Source:   "test",
	})
	checker := NewURLSafetyChecker(policy)
	result := checker.CheckURL("https://malware-site.com/")
	if result.Safe {
		t.Fatalf("CheckURL(malware-site.com) = Safe=true, want Safe=false")
	}
	if result.Category != URLSafetyCategoryMalware {
		t.Fatalf("Category = %v, want URLSafetyCategoryMalware", result.Category)
	}
}

func TestURLSafetyChecker_CategoryAdult(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	policy.Blocklist = append(policy.Blocklist, URLSafetyBlocklistEntry{
		Pattern:  "adult-site.com",
		Category: URLSafetyCategoryAdult,
		Source:   "test",
	})
	checker := NewURLSafetyChecker(policy)
	result := checker.CheckURL("https://adult-site.com/")
	if result.Safe {
		t.Fatalf("CheckURL(adult-site.com) = Safe=true, want Safe=false")
	}
	if result.Category != URLSafetyCategoryAdult {
		t.Fatalf("Category = %v, want URLSafetyCategoryAdult", result.Category)
	}
}

// --------------------------------------------------------------------------
// Pattern matching edge cases
// --------------------------------------------------------------------------

func TestMatchHostAgainstRule_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		pattern string
		want    bool
	}{
		{"empty_host", "", "example.com", false},
		{"empty_pattern", "example.com", "", false},
		{"both_empty", "", "", false},
		{"www_stripped", "www.example.com", "example.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchHostAgainstRule(tt.host, tt.pattern)
			if got != tt.want {
				t.Fatalf("matchHostAgainstRule(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

// --------------------------------------------------------------------------
// Helper: check that category string values are correct
// --------------------------------------------------------------------------

func TestURLSafetyCategoryValues(t *testing.T) {
	if URLSafetyCategoryPhishing != "phishing" {
		t.Fatalf("URLSafetyCategoryPhishing = %q, want \"phishing\"", URLSafetyCategoryPhishing)
	}
	if URLSafetyCategoryMalware != "malware" {
		t.Fatalf("URLSafetyCategoryMalware = %q, want \"malware\"", URLSafetyCategoryMalware)
	}
	if URLSafetyCategoryAdult != "adult" {
		t.Fatalf("URLSafetyCategoryAdult = %q, want \"adult\"", URLSafetyCategoryAdult)
	}
	if URLSafetyCategorySSRF != "ssrf" {
		t.Fatalf("URLSafetyCategorySSRF = %q, want \"ssrf\"", URLSafetyCategorySSRF)
	}
	if URLSafetyCategoryPrivate != "private" {
		t.Fatalf("URLSafetyCategoryPrivate = %q, want \"private\"", URLSafetyCategoryPrivate)
	}
}

// --------------------------------------------------------------------------
// Policy enables/disables checks
// --------------------------------------------------------------------------

func TestURLSafetyChecker_DisabledPolicyAllowsAll(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	policy.Enabled = false
	checker := NewURLSafetyChecker(policy)

	result := checker.CheckURL("https://clearly-blocked-example.com/")
	if !result.Safe {
		t.Fatalf("CheckURL with Enabled=false = Safe=false, want Safe=true")
	}
}

// --------------------------------------------------------------------------
// Subdomain matching
// --------------------------------------------------------------------------

func TestMatchHostAgainstRule_SubdomainMatching(t *testing.T) {
	// *.example.com should match sub.example.com but not example.com or notexample.com.
	tests := []struct {
		host    string
		pattern string
		want    bool
	}{
		{"sub.example.com", "*.example.com", true},
		{"deep.sub.example.com", "*.example.com", true},
		{"example.com", "*.example.com", false},
		{"notexample.com", "*.example.com", false},
		{"sub.notexample.com", "*.example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.host+"_"+tt.pattern, func(t *testing.T) {
			got := matchHostAgainstRule(tt.host, tt.pattern)
			if got != tt.want {
				t.Fatalf("matchHostAgainstRule(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

// --------------------------------------------------------------------------
// Wildcard pattern in blocklist
// --------------------------------------------------------------------------

func TestURLSafetyChecker_WildcardBlocklist(t *testing.T) {
	policy := DefaultURLSafetyPolicy()
	policy.Blocklist = append(policy.Blocklist, URLSafetyBlocklistEntry{
		Pattern:  "*.evil.com",
		Category: URLSafetyCategoryMalware,
		Source:   "test",
	})
	checker := NewURLSafetyChecker(policy)

	// subdomain should be blocked
	result1 := checker.CheckURL("https://sub.evil.com/path")
	if result1.Safe {
		t.Fatalf("CheckURL(sub.evil.com) = Safe=true, want Safe=false")
	}

	// exact match should NOT be blocked (wildcard is for subdomains only)
	result2 := checker.CheckURL("https://evil.com/path")
	if !result2.Safe {
		t.Fatalf("CheckURL(evil.com) = Safe=false, want Safe=true (wildcard *.evil.com does not match evil.com itself)")
	}
}
