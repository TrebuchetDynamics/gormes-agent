package urlsafety

import (
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/configparse"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/urlpolicy"
)

// URLSafetyCategory classifies the type of blocked content.
type URLSafetyCategory string

const (
	URLSafetyCategoryPhishing URLSafetyCategory = "phishing"
	URLSafetyCategoryMalware  URLSafetyCategory = "malware"
	URLSafetyCategoryAdult    URLSafetyCategory = "adult"
	URLSafetyCategorySSRF     URLSafetyCategory = "ssrf"
	URLSafetyCategoryPrivate  URLSafetyCategory = "private"
)

// URLSafetyResult holds the outcome of a URL safety check.
type URLSafetyResult struct {
	Safe     bool
	Category URLSafetyCategory // empty when Safe is true
	Reason   string            // human-readable reason
	Host     string            // normalized host that was checked
}

// URLSafetyBlocklistEntry represents a single blocklist rule.
type URLSafetyBlocklistEntry struct {
	Pattern  string
	Category URLSafetyCategory // classification type
	Source   string            // "config", "shared_file", or "builtin"
}

// URLSafetyAllowlistEntry represents a single allowlist rule.
type URLSafetyAllowlistEntry struct {
	Pattern string // domain pattern
	Source  string // "config" or "builtin"
}

// URLSafetyPolicy holds the complete safety policy configuration.
type URLSafetyPolicy struct {
	Enabled          bool
	Blocklist        []URLSafetyBlocklistEntry
	Allowlist        []URLSafetyAllowlistEntry
	AllowPrivateURLs bool // global toggle for SSRF protection
	SharedFilesPaths []string
}

// URLSafetyChecker performs URL safety checks.
type URLSafetyChecker struct {
	policy      URLSafetyPolicy
	mu          sync.RWMutex
	cache       map[string]URLSafetyResult
	cacheTTL    time.Duration
	lastRefresh time.Time
}

// --------------------------------------------------------------------------
// Blocked hostnames — always blocked even with AllowPrivateURLs enabled.
// --------------------------------------------------------------------------

var blockedHostnames = map[string]bool{
	"metadata.google.internal": true,
	"metadata.goog":            true,
}

// --------------------------------------------------------------------------
// Blocked IPs — always blocked even with AllowPrivateURLs enabled.
// --------------------------------------------------------------------------

var blockedIPs = map[string]bool{
	"169.254.169.254": true, // AWS/GCP/Azure/DO/Oracle metadata
	"169.254.170.2":   true, // AWS ECS task metadata
	"169.254.169.253": true, // Azure IMDS wire server
	"fd00:ec2::254":   true, // AWS metadata (IPv6)
	"100.100.100.200": true, // Alibaba Cloud metadata
}

// Blocked CIDR networks.
var blockedNetworks = []string{
	"169.254.0.0/16", // Link-local range
}

// Trusted private IP hosts.
var trustedPrivateIPHosts = map[string]bool{
	"multimedia.nt.qq.com.cn": true,
}

// --------------------------------------------------------------------------
// Builtin blocklist.
// --------------------------------------------------------------------------

var builtinBlocklist = []URLSafetyBlocklistEntry{
	{Pattern: "phishing-example.com", Category: URLSafetyCategoryPhishing, Source: "builtin"},
	{Pattern: "malware-example.net", Category: URLSafetyCategoryMalware, Source: "builtin"},
	{Pattern: "adult-example.org", Category: URLSafetyCategoryAdult, Source: "builtin"},
}

// Builtin allowlist.
var builtinAllowlist = []URLSafetyAllowlistEntry{
	{Pattern: "trusted-partner.com", Source: "builtin"},
}

// Global toggle cache.
var (
	allowPrivateResolved bool
	cachedAllowPrivate   bool
	allowPrivateMu       sync.Mutex
)

// --------------------------------------------------------------------------
// Policy loading
// --------------------------------------------------------------------------

// NewURLSafetyChecker creates a new URLSafetyChecker with the given policy.
func NewURLSafetyChecker(policy URLSafetyPolicy) *URLSafetyChecker {
	return &URLSafetyChecker{
		policy:   policy,
		cache:    make(map[string]URLSafetyResult),
		cacheTTL: 30 * time.Second,
	}
}

// DefaultURLSafetyPolicy returns a conservative default policy.
func DefaultURLSafetyPolicy() URLSafetyPolicy {
	return URLSafetyPolicy{
		Enabled:          true,
		Blocklist:        append([]URLSafetyBlocklistEntry{}, builtinBlocklist...),
		Allowlist:        append([]URLSafetyAllowlistEntry{}, builtinAllowlist...),
		AllowPrivateURLs: false,
	}
}

// --------------------------------------------------------------------------
// Global toggle helpers
// --------------------------------------------------------------------------

// GlobalAllowPrivateURLs checks the global toggle priority.
func GlobalAllowPrivateURLs() bool {
	allowPrivateMu.Lock()
	defer allowPrivateMu.Unlock()

	if allowPrivateResolved {
		return cachedAllowPrivate
	}

	allowPrivateResolved = true
	cachedAllowPrivate = false

	envVal := os.Getenv("GORMES_ALLOW_PRIVATE_URLS")
	switch strings.ToLower(strings.TrimSpace(envVal)) {
	case "true", "1", "yes":
		cachedAllowPrivate = true
		return cachedAllowPrivate
	case "false", "0", "no":
		return cachedAllowPrivate
	}

	return cachedAllowPrivate
}

// ResetAllowPrivateCache resets the global toggle cache (for tests only).
func ResetAllowPrivateCache() {
	allowPrivateMu.Lock()
	defer allowPrivateMu.Unlock()
	allowPrivateResolved = false
	cachedAllowPrivate = false
}

// --------------------------------------------------------------------------
// IP classification helpers
// --------------------------------------------------------------------------

// isBlockedIP returns true if the IP should be blocked for SSRF protection.
func isBlockedIP(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 != nil {
		checkIP := net.IP(ip4)
		if checkIP.IsLoopback() || checkIP.IsPrivate() || checkIP.IsLinkLocalUnicast() || checkIP.IsMulticast() || checkIP.IsUnspecified() {
			return true
		}
		if isIETFReserved(ip4) {
			return true
		}
		if isCGNAT(ip4) {
			return true
		}
		return false
	}

	ip6 := ip.To16()
	if ip6 != nil {
		checkIP := net.IP(ip6)
		if checkIP.IsLoopback() || checkIP.IsLinkLocalUnicast() || checkIP.IsMulticast() || checkIP.IsUnspecified() {
			return true
		}
	}
	return false
}

// isIETFReserved checks if an IPv4 address is in 240.0.0.0/4.
func isIETFReserved(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	return ip4[0]&0xf0 == 0xf0
}

// isCGNAT checks if an IPv4 address is in 100.64.0.0/10.
func isCGNAT(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	return ip4[0] == 100 && (uint32(ip4[0])<<24|uint32(ip4[1])<<16|uint32(ip4[2])<<8|uint32(ip4[3]))&0xFFC00000 == 0x64400000
}

// isAlwaysBlockedIP returns true if the IP is a known cloud metadata endpoint.
func isAlwaysBlockedIP(ip net.IP) bool {
	ipStr := ip.String()
	if blockedIPs[ipStr] {
		return true
	}
	ip4 := ip.To4()
	if ip4 != nil {
		for _, cidr := range blockedNetworks {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}
			if network.Contains(ip4) {
				return true
			}
		}
	}
	return false
}

// allowsPrivateIPResolution returns true when a trusted HTTPS hostname may bypass IP blocking.
func allowsPrivateIPResolution(hostname, scheme string) bool {
	return scheme == "https" && trustedPrivateIPHosts[strings.ToLower(hostname)]
}

// --------------------------------------------------------------------------
// Pattern matching
// --------------------------------------------------------------------------

// matchHostAgainstRule matches a host against a pattern.
// Supports exact match, suffix match (e.g., ".example.com"), and wildcard (e.g., "*.example.com").
func matchHostAgainstRule(host, pattern string) bool {
	return urlpolicy.MatchHostAgainstRule(host, pattern)
}

// extractHostFromURL extracts and normalizes the host from a URL-like string.
func extractHostFromURL(rawURL string) string {
	return urlpolicy.ExtractHostFromURL(rawURL)
}

// stripPort strips the port from a host string, handling IPv6 correctly.
func stripPort(host string) string {
	return urlpolicy.StripPort(host)
}

// normalizeHost normalizes a hostname for comparison.
func normalizeHost(host string) string {
	return urlpolicy.NormalizeHost(host)
}

// --------------------------------------------------------------------------
// Main safety check
// --------------------------------------------------------------------------

// CheckURL performs a full URL safety check including SSRF and blocklist.
func (c *URLSafetyChecker) CheckURL(rawURL string) URLSafetyResult {
	// Fast path: check cache first.
	c.mu.RLock()
	if cached, ok := c.cache[rawURL]; ok && time.Since(c.lastRefresh) < c.cacheTTL {
		c.mu.RUnlock()
		return cached
	}
	c.mu.RUnlock()

	result := c.checkURLImpl(rawURL)

	// Cache the result.
	c.mu.Lock()
	if len(c.cache) > 1000 {
		c.cache = make(map[string]URLSafetyResult)
	}
	c.cache[rawURL] = result
	c.lastRefresh = time.Now()
	c.mu.Unlock()

	return result
}

// checkURLImpl is the internal implementation without caching.
func (c *URLSafetyChecker) checkURLImpl(rawURL string) URLSafetyResult {
	c.mu.RLock()
	enabled := c.policy.Enabled
	allowPrivate := c.policy.AllowPrivateURLs
	c.mu.RUnlock()

	if !enabled {
		return URLSafetyResult{Safe: true}
	}

	host := extractHostFromURL(rawURL)
	if host == "" {
		return URLSafetyResult{
			Safe:     false,
			Category: URLSafetyCategorySSRF,
			Reason:   "empty or invalid hostname",
			Host:     "",
		}
	}

	// Check allowlist first.
	c.mu.RLock()
	for _, entry := range c.policy.Allowlist {
		if matchHostAgainstRule(host, entry.Pattern) {
			c.mu.RUnlock()
			return URLSafetyResult{
				Safe:   true,
				Reason: "matched allowlist pattern: " + entry.Pattern,
				Host:   host,
			}
		}
	}
	c.mu.RUnlock()

	// Check blocklist.
	c.mu.RLock()
	for _, entry := range c.policy.Blocklist {
		if matchHostAgainstRule(host, entry.Pattern) {
			c.mu.RUnlock()
			return URLSafetyResult{
				Safe:     false,
				Category: entry.Category,
				Reason:   "matched blocklist pattern: " + entry.Pattern,
				Host:     host,
			}
		}
	}
	c.mu.RUnlock()

	// SSRF check: resolve hostname and check IP.
	scheme := getScheme(rawURL)

	// Always block known internal hostnames.
	if blockedHostnames[strings.ToLower(host)] {
		return URLSafetyResult{
			Safe:     false,
			Category: URLSafetyCategorySSRF,
			Reason:   "blocked internal hostname: " + host,
			Host:     host,
		}
	}

	// Check if private IP resolution is allowed for this hostname.
	allowPrivateIP := allowsPrivateIPResolution(host, scheme)

	// Try to resolve and check IP.
	ips, err := net.LookupIP(host)
	if err != nil {
		return URLSafetyResult{
			Safe:     false,
			Category: URLSafetyCategorySSRF,
			Reason:   "DNS resolution failed for: " + host,
			Host:     host,
		}
	}

	for _, ip := range ips {
		// Always block cloud metadata IPs and link-local.
		if isAlwaysBlockedIP(ip) {
			return URLSafetyResult{
				Safe:     false,
				Category: URLSafetyCategorySSRF,
				Reason:   "blocked cloud metadata address: " + ip.String(),
				Host:     host,
			}
		}

		// Check private IP blocking unless explicitly allowed.
		if !allowPrivate && !allowPrivateIP && isBlockedIP(ip) {
			return URLSafetyResult{
				Safe:     false,
				Category: URLSafetyCategoryPrivate,
				Reason:   "blocked private/internal address: " + host + " -> " + ip.String(),
				Host:     host,
			}
		}
	}

	return URLSafetyResult{
		Safe:   true,
		Reason: "URL passed all safety checks",
		Host:   host,
	}
}

// getScheme extracts the scheme from a URL string.
func getScheme(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Scheme)
}

// CheckURLStatic is a convenience function that creates a default checker and checks a URL.
func CheckURLStatic(rawURL string) URLSafetyResult {
	checker := NewURLSafetyChecker(DefaultURLSafetyPolicy())
	return checker.CheckURL(rawURL)
}

// --------------------------------------------------------------------------
// Config normalization helpers
// --------------------------------------------------------------------------

// NormalizeBlocklistRule normalizes a raw blocklist rule string.
func NormalizeBlocklistRule(raw string) string {
	return urlpolicy.NormalizeBlocklistRule(raw)
}

// --------------------------------------------------------------------------
// InvalidateCache forces the next check to bypass cache.
// --------------------------------------------------------------------------

// InvalidateCache clears the safety check cache.
func (c *URLSafetyChecker) InvalidateCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]URLSafetyResult)
	c.lastRefresh = time.Time{}
}

// SetAllowPrivateURLs updates the AllowPrivateURLs setting.
func (c *URLSafetyChecker) SetAllowPrivateURLs(allow bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.policy.AllowPrivateURLs = allow
}

// AllowPrivateURLs returns the current AllowPrivateURLs setting.
func (c *URLSafetyChecker) AllowPrivateURLs() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.policy.AllowPrivateURLs
}

// --------------------------------------------------------------------------
// Policy mutation helpers
// --------------------------------------------------------------------------

// AddBlocklistEntry adds a blocklist entry to the policy.
func (c *URLSafetyChecker) AddBlocklistEntry(pattern string, category URLSafetyCategory, source string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.policy.Blocklist = append(c.policy.Blocklist, URLSafetyBlocklistEntry{
		Pattern:  pattern,
		Category: category,
		Source:   source,
	})
	c.cache = make(map[string]URLSafetyResult)
}

// AddAllowlistEntry adds an allowlist entry to the policy.
func (c *URLSafetyChecker) AddAllowlistEntry(pattern, source string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.policy.Allowlist = append(c.policy.Allowlist, URLSafetyAllowlistEntry{
		Pattern: pattern,
		Source:  source,
	})
	c.cache = make(map[string]URLSafetyResult)
}

// LoadBlocklistFromLines parses a list of rules (one per line) into blocklist entries.
func LoadBlocklistFromLines(lines []string, category URLSafetyCategory, source string) []URLSafetyBlocklistEntry {
	var entries []URLSafetyBlocklistEntry
	for _, line := range lines {
		normalized := NormalizeBlocklistRule(line)
		if normalized == "" {
			continue
		}
		entries = append(entries, URLSafetyBlocklistEntry{
			Pattern:  normalized,
			Category: category,
			Source:   source,
		})
	}
	return entries
}

// ParseBoolLike parses a bool-like value from various Go types.
func ParseBoolLike(raw any, fallback bool) bool {
	return configparse.BoolLike(raw, fallback)
}

// ParseBoolLikeWithEvidence parses a bool-like value and returns evidence if invalid.
func ParseBoolLikeWithEvidence(raw any, fallback bool) (bool, string) {
	return configparse.BoolLikeWithEvidence(raw, fallback)
}

// ParseIntLike parses an int-like value from various Go types.
func ParseIntLike(raw any, fallback int) int {
	return configparse.IntLike(raw, fallback)
}
