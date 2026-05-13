package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety"

type URLSafetyCategory = safety.URLSafetyCategory

const (
	URLSafetyCategoryPhishing URLSafetyCategory = safety.URLSafetyCategoryPhishing
	URLSafetyCategoryMalware  URLSafetyCategory = safety.URLSafetyCategoryMalware
	URLSafetyCategoryAdult    URLSafetyCategory = safety.URLSafetyCategoryAdult
	URLSafetyCategorySSRF     URLSafetyCategory = safety.URLSafetyCategorySSRF
	URLSafetyCategoryPrivate  URLSafetyCategory = safety.URLSafetyCategoryPrivate
)

type URLSafetyResult = safety.URLSafetyResult
type URLSafetyBlocklistEntry = safety.URLSafetyBlocklistEntry
type URLSafetyAllowlistEntry = safety.URLSafetyAllowlistEntry
type URLSafetyPolicy = safety.URLSafetyPolicy
type URLSafetyChecker = safety.URLSafetyChecker
type WebsitePolicyError = safety.WebsitePolicyError
type WebsitePolicyResult = safety.WebsitePolicyResult
type WebsitePolicyRule = safety.WebsitePolicyRule
type WebsitePolicy = safety.WebsitePolicy
type CheckWebsiteAccessResult = safety.CheckWebsiteAccessResult
type WebSearchURLSafetyResult = safety.WebSearchURLSafetyResult

func NewURLSafetyChecker(policy URLSafetyPolicy) *URLSafetyChecker {
	return safety.NewURLSafetyChecker(policy)
}

func DefaultURLSafetyPolicy() URLSafetyPolicy {
	return safety.DefaultURLSafetyPolicy()
}

func GlobalAllowPrivateURLs() bool {
	return safety.GlobalAllowPrivateURLs()
}

func ResetAllowPrivateCache() {
	safety.ResetAllowPrivateCache()
}

func CheckURLStatic(rawURL string) URLSafetyResult {
	return safety.CheckURLStatic(rawURL)
}

func NormalizeBlocklistRule(raw string) string {
	return safety.NormalizeBlocklistRule(raw)
}

func LoadBlocklistFromLines(lines []string, category URLSafetyCategory, source string) []URLSafetyBlocklistEntry {
	return safety.LoadBlocklistFromLines(lines, category, source)
}

func ParseBoolLike(raw any, fallback bool) bool {
	return safety.ParseBoolLike(raw, fallback)
}

func ParseBoolLikeWithEvidence(raw any, fallback bool) (bool, string) {
	return safety.ParseBoolLikeWithEvidence(raw, fallback)
}

func ParseIntLike(raw any, fallback int) int {
	return safety.ParseIntLike(raw, fallback)
}

func DefaultWebsitePolicyConfigPath() string {
	return safety.DefaultWebsitePolicyConfigPath()
}

func LoadWebsitePolicy(configPath string) (*WebsitePolicy, error) {
	return safety.LoadWebsitePolicy(configPath)
}

func InvalidateWebsitePolicyCache() {
	safety.InvalidateWebsitePolicyCache()
}

func CheckWebsiteAccess(rawURL string, configPath string) *WebsitePolicyResult {
	return safety.CheckWebsiteAccess(rawURL, configPath)
}

func NewWebsitePolicy(enabled bool, domains []string, sharedFiles []string, source string) *WebsitePolicy {
	return safety.NewWebsitePolicy(enabled, domains, sharedFiles, source)
}

func CheckWebSearchURL(rawURL string, safetyPolicy URLSafetyPolicy, policyConfigPath string) *WebSearchURLSafetyResult {
	return safety.CheckWebSearchURL(rawURL, safetyPolicy, policyConfigPath)
}

func CheckWebSearchURLStatic(rawURL string) *WebSearchURLSafetyResult {
	return safety.CheckWebSearchURLStatic(rawURL)
}
