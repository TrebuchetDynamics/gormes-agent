package safety

import (
	"net"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/configparse"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/urlpolicy"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/urlsafety"
)

// URLSafetyCategory classifies the type of blocked content.
type URLSafetyCategory = urlsafety.URLSafetyCategory

const (
	URLSafetyCategoryPhishing URLSafetyCategory = urlsafety.URLSafetyCategoryPhishing
	URLSafetyCategoryMalware  URLSafetyCategory = urlsafety.URLSafetyCategoryMalware
	URLSafetyCategoryAdult    URLSafetyCategory = urlsafety.URLSafetyCategoryAdult
	URLSafetyCategorySSRF     URLSafetyCategory = urlsafety.URLSafetyCategorySSRF
	URLSafetyCategoryPrivate  URLSafetyCategory = urlsafety.URLSafetyCategoryPrivate
)

// URLSafetyResult holds the outcome of a URL safety check.
type URLSafetyResult = urlsafety.URLSafetyResult

// URLSafetyBlocklistEntry represents a single blocklist rule.
type URLSafetyBlocklistEntry = urlsafety.URLSafetyBlocklistEntry

// URLSafetyAllowlistEntry represents an allowlist override.
type URLSafetyAllowlistEntry = urlsafety.URLSafetyAllowlistEntry

// URLSafetyPolicy defines the policy for URL checking.
type URLSafetyPolicy = urlsafety.URLSafetyPolicy

// URLSafetyChecker performs URL safety checks.
type URLSafetyChecker = urlsafety.URLSafetyChecker

func NewURLSafetyChecker(policy URLSafetyPolicy) *URLSafetyChecker {
	return urlsafety.NewURLSafetyChecker(policy)
}

func DefaultURLSafetyPolicy() URLSafetyPolicy {
	return urlsafety.DefaultURLSafetyPolicy()
}

func GlobalAllowPrivateURLs() bool {
	return urlsafety.GlobalAllowPrivateURLs()
}

func ResetAllowPrivateCache() {
	urlsafety.ResetAllowPrivateCache()
}

func CheckURLStatic(rawURL string) URLSafetyResult {
	return urlsafety.CheckURLStatic(rawURL)
}

func NormalizeBlocklistRule(raw string) string {
	return urlpolicy.NormalizeBlocklistRule(raw)
}

func LoadBlocklistFromLines(lines []string, category URLSafetyCategory, source string) []URLSafetyBlocklistEntry {
	return urlsafety.LoadBlocklistFromLines(lines, category, source)
}

func ParseBoolLike(raw any, fallback bool) bool {
	return configparse.BoolLike(raw, fallback)
}

func ParseBoolLikeWithEvidence(raw any, fallback bool) (bool, string) {
	return configparse.BoolLikeWithEvidence(raw, fallback)
}

func ParseIntLike(raw any, fallback int) int {
	return configparse.IntLike(raw, fallback)
}

func matchHostAgainstRule(host, pattern string) bool {
	return urlpolicy.MatchHostAgainstRule(host, pattern)
}

func extractHostFromURL(rawURL string) string {
	return urlpolicy.ExtractHostFromURL(rawURL)
}

func stripPort(host string) string {
	return urlpolicy.StripPort(host)
}

func normalizeHost(host string) string {
	return urlpolicy.NormalizeHost(host)
}

func isBlockedIP(ip net.IP) bool {
	return urlsafety.IsBlockedIP(ip)
}
