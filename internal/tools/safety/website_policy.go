package safety

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WebsitePolicyError is raised when a website policy file is malformed.
type WebsitePolicyError struct {
	Message string
}

func (e *WebsitePolicyError) Error() string {
	return e.Message
}

// WebsitePolicyResult holds the outcome of a website policy check.
type WebsitePolicyResult struct {
	Allowed bool
	Host    string
	Rule    string
	Source  string
	Message string
}

// WebsitePolicyRule represents a single policy rule.
type WebsitePolicyRule struct {
	Pattern string
	Source  string
}

// WebsitePolicy holds the parsed website blocklist/allowlist policy.
type WebsitePolicy struct {
	Enabled bool
	Rules   []WebsitePolicyRule
}

// websitePolicyCache caches parsed policy with TTL.
type websitePolicyCache struct {
	policy    *WebsitePolicy
	path      string
	timestamp time.Time
}

const websitePolicyCacheTTL = 30 * time.Second

var (
	policyCache   = websitePolicyCache{}
	policyCacheMu = sync.RWMutex{}
)

// --------------------------------------------------------------------------
// Default config
// --------------------------------------------------------------------------

var defaultWebsitePolicy = WebsitePolicy{
	Enabled: false,
	Rules:   []WebsitePolicyRule{},
}

// --------------------------------------------------------------------------
// Config path resolution
// --------------------------------------------------------------------------

// DefaultWebsitePolicyConfigPath returns the default config path for website policy.
func DefaultWebsitePolicyConfigPath() string {
	// Check GORMES_HOME env var first.
	if gormesHome := os.Getenv("GORMES_HOME"); gormesHome != "" {
		return filepath.Join(gormesHome, "config.toml")
	}
	// Fallback to home directory.
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "gormes", "config.toml")
}

// --------------------------------------------------------------------------
// Policy loading
// --------------------------------------------------------------------------

// LoadWebsitePolicy loads the website blocklist policy from a config file.
// It returns a cached policy if still fresh.
func LoadWebsitePolicy(configPath string) (*WebsitePolicy, error) {
	policyCacheMu.RLock()
	if policyCache.policy != nil &&
		policyCache.path == configPath &&
		time.Since(policyCache.timestamp) < websitePolicyCacheTTL {
		policyCacheMu.RUnlock()
		return policyCache.policy, nil
	}
	policyCacheMu.RUnlock()

	policy, err := loadWebsitePolicyImpl(configPath)
	if err != nil {
		return nil, err
	}

	policyCacheMu.Lock()
	policyCache.policy = policy
	policyCache.path = configPath
	policyCache.timestamp = time.Now()
	policyCacheMu.Unlock()

	return policy, nil
}

// loadWebsitePolicyImpl is the internal implementation without caching.
func loadWebsitePolicyImpl(configPath string) (*WebsitePolicy, error) {
	if configPath == "" {
		return &defaultWebsitePolicy, nil
	}

	// Check if file exists.
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &defaultWebsitePolicy, nil
	}

	// Try to read and parse the config.
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, &WebsitePolicyError{Message: "failed to read config file: " + err.Error()}
	}

	return parseWebsitePolicyContent(data)
}

// parseWebsitePolicyContent parses website policy from raw config content.
func parseWebsitePolicyContent(data []byte) (*WebsitePolicy, error) {
	content := string(data)

	policy := &WebsitePolicy{
		Enabled: false,
		Rules:   []WebsitePolicyRule{},
	}

	// Simple TOML-like parsing for website_blocklist section.
	lines := strings.Split(content, "\n")

	inBlocklistSection := false
	var currentSection string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for section headers.
		if strings.HasPrefix(trimmed, "[") {
			inBlocklistSection = false
			currentSection = strings.ToLower(strings.Trim(trimmed, "[] "))
			if currentSection == "security.website_blocklist" ||
				currentSection == "security.website_blocklist.enabled" {
				inBlocklistSection = true
			}
			continue
		}

		// Check for top-level website_blocklist keys.
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "website_blocklist") {
			inBlocklistSection = true
			currentSection = "website_blocklist"
		}

		// Parse key = value pairs.
		if inBlocklistSection || currentSection == "website_blocklist" {
			if strings.Contains(trimmed, "=") {
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(strings.ToLower(parts[0]))
					value := strings.TrimSpace(parts[1])

					switch key {
					case "enabled":
						if value == "true" || value == "1" {
							policy.Enabled = true
						}
					case "domains":
						// Parse array of domains.
						domains := parseStringArray(value)
						for _, domain := range domains {
							normalized := NormalizeBlocklistRule(domain)
							if normalized != "" {
								policy.Rules = append(policy.Rules, WebsitePolicyRule{
									Pattern: normalized,
									Source:  "config",
								})
							}
						}
					case "shared_files":
						files := parseStringArray(value)
						for _, file := range files {
							file = strings.Trim(file, "\" ")
							if file != "" {
								// Load rules from shared file.
								rules, err := loadRulesFromSharedFile(file)
								if err == nil {
									policy.Rules = append(policy.Rules, rules...)
								}
							}
						}
					}
				}
			}
		}
	}

	return policy, nil
}

// parseStringArray parses a TOML array string like ["a.com", "b.com"].
func parseStringArray(value string) []string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return []string{}
	}
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")

	var result []string
	elements := strings.Split(value, ",")
	for _, el := range elements {
		el = strings.TrimSpace(el)
		el = strings.Trim(el, "\" ")
		if el != "" {
			result = append(result, el)
		}
	}
	return result
}

// loadRulesFromSharedFile loads rules from a shared blocklist file.
func loadRulesFromSharedFile(path string) ([]WebsitePolicyRule, error) {
	path = os.ExpandEnv(path)

	// Resolve relative paths against GORMES_HOME.
	if !filepath.IsAbs(path) {
		if gormesHome := os.Getenv("GORMES_HOME"); gormesHome != "" {
			path = filepath.Join(gormesHome, path)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &WebsitePolicyError{Message: "failed to read shared blocklist file: " + err.Error()}
	}

	var rules []WebsitePolicyRule
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		normalized := NormalizeBlocklistRule(trimmed)
		if normalized != "" {
			rules = append(rules, WebsitePolicyRule{
				Pattern: normalized,
				Source:  path,
			})
		}
	}

	return rules, nil
}

// --------------------------------------------------------------------------
// InvalidateCache forces the next check to re-read config.
// --------------------------------------------------------------------------

// InvalidateWebsitePolicyCache clears the cached policy.
func InvalidateWebsitePolicyCache() {
	policyCacheMu.Lock()
	defer policyCacheMu.Unlock()
	policyCache.policy = nil
	policyCache.path = ""
	policyCache.timestamp = time.Time{}
}

// --------------------------------------------------------------------------
// Check website access
// --------------------------------------------------------------------------

// CheckWebsiteAccess checks whether a URL is allowed by the website policy.
// Returns nil if allowed, or a WebsitePolicyResult with block metadata if blocked.
func CheckWebsiteAccess(rawURL string, configPath string) *WebsitePolicyResult {
	host := extractHostFromURL(rawURL)
	if host == "" {
		return nil
	}

	policy, err := LoadWebsitePolicy(configPath)
	if err != nil {
		// Fail open on config errors — don't let a config typo break all web tools.
		return nil
	}

	if !policy.Enabled {
		return nil
	}

	for _, rule := range policy.Rules {
		if matchHostAgainstRule(host, rule.Pattern) {
			return &WebsitePolicyResult{
				Allowed: false,
				Host:    host,
				Rule:    rule.Pattern,
				Source:  rule.Source,
				Message: "Blocked by website policy: '" + host + "' matched rule '" + rule.Pattern + "' from " + rule.Source,
			}
		}
	}

	return nil
}

// CheckWebsiteAccessResult is the return type for CheckWebsiteAccess.
// nil return means allowed; non-nil means blocked.
type CheckWebsiteAccessResult = *WebsitePolicyResult

// --------------------------------------------------------------------------
// Policy mutation helpers
// --------------------------------------------------------------------------

// NewWebsitePolicy creates a WebsitePolicy from raw configuration.
func NewWebsitePolicy(enabled bool, domains []string, sharedFiles []string, source string) *WebsitePolicy {
	policy := &WebsitePolicy{
		Enabled: enabled,
		Rules:   []WebsitePolicyRule{},
	}

	for _, domain := range domains {
		normalized := NormalizeBlocklistRule(domain)
		if normalized != "" {
			policy.Rules = append(policy.Rules, WebsitePolicyRule{
				Pattern: normalized,
				Source:  source,
			})
		}
	}

	for _, file := range sharedFiles {
		rules, err := loadRulesFromSharedFile(file)
		if err == nil {
			policy.Rules = append(policy.Rules, rules...)
		}
	}

	return policy
}

// IsWebsiteBlocked checks if a URL is blocked by the given policy.
func (p *WebsitePolicy) IsWebsiteBlocked(rawURL string) *WebsitePolicyResult {
	host := extractHostFromURL(rawURL)
	if host == "" {
		return nil
	}

	if !p.Enabled {
		return nil
	}

	for _, rule := range p.Rules {
		if matchHostAgainstRule(host, rule.Pattern) {
			return &WebsitePolicyResult{
				Allowed: false,
				Host:    host,
				Rule:    rule.Pattern,
				Source:  rule.Source,
				Message: "Blocked by website policy: '" + host + "' matched rule '" + rule.Pattern + "' from " + rule.Source,
			}
		}
	}

	return nil
}

// AddRule adds a rule to the policy.
func (p *WebsitePolicy) AddRule(pattern, source string) {
	p.Rules = append(p.Rules, WebsitePolicyRule{
		Pattern: pattern,
		Source:  source,
	})
}

// --------------------------------------------------------------------------
// Web search / browser integration helpers
// --------------------------------------------------------------------------

// WebSearchURLSafetyResult combines URL safety and website policy results.
type WebSearchURLSafetyResult struct {
	URL          string
	Allowed      bool
	SSRFReason   string // non-empty if blocked by SSRF
	BlockReason  string // non-empty if blocked by website policy
	PolicyRule   string
	PolicySource string
}

// CheckWebSearchURL performs a combined URL safety and website policy check.
// This is the main integration point for web_search and browser tools.
func CheckWebSearchURL(rawURL string, safetyPolicy URLSafetyPolicy, policyConfigPath string) *WebSearchURLSafetyResult {
	result := &WebSearchURLSafetyResult{
		URL:     rawURL,
		Allowed: true,
	}

	// Step 1: Check URL safety (SSRF, blocklist, allowlist).
	safetyChecker := NewURLSafetyChecker(safetyPolicy)
	urlResult := safetyChecker.CheckURL(rawURL)

	if !urlResult.Safe {
		result.Allowed = false
		result.SSRFReason = urlResult.Reason
		return result
	}

	// Step 2: Check website policy.
	policyResult := CheckWebsiteAccess(rawURL, policyConfigPath)

	if policyResult != nil {
		result.Allowed = false
		result.BlockReason = policyResult.Message
		result.PolicyRule = policyResult.Rule
		result.PolicySource = policyResult.Source
		return result
	}

	return result
}

// CheckWebSearchURLStatic is a convenience function using default policies.
func CheckWebSearchURLStatic(rawURL string) *WebSearchURLSafetyResult {
	return CheckWebSearchURL(rawURL, DefaultURLSafetyPolicy(), DefaultWebsitePolicyConfigPath())
}
