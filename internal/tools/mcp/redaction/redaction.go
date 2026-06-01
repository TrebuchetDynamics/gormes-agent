package redaction

import (
	"regexp"
	"sort"
	"strings"
)

// Value is the public placeholder used in MCP operator surfaces when config
// contains credentials or token-shaped values.
const Value = "[REDACTED]"

var credentialPattern = regexp.MustCompile(`(?i)(ghp_[A-Za-z0-9_-]{1,255}|sk-[A-Za-z0-9_-]{1,255}|Bearer\s+\S+|token=[^\s&,;"']{1,255}|key=[^\s&,;"']{1,255}|API_KEY=[^\s&,;"']{1,255}|password=[^\s&,;"']{1,255}|secret=[^\s&,;"']{1,255})`)

// Map returns a copy of values with secret-looking keys and token-shaped values
// redacted for operator-visible MCP status surfaces.
func Map(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = MapValue(key, value)
	}
	return out
}

// MapValue returns the operator-visible form for one key/value pair.
func MapValue(key, value string) string {
	if IsSecretKey(key) || IsSecretValue(value) {
		return Value
	}
	return String(value)
}

var exactSecretKeyNames = []string{
	"pat",
}

var secretKeyFragments = []string{
	"authorization",
	"authheader",
	"apikey",
	"accesstoken",
	"refreshtoken",
	"personalaccesstoken",
	"token",
	"secret",
	"password",
}

// IsSecretKey reports whether key names credential-like material. It compares
// a separator-free lowercase key so common config spellings such as apiKey,
// api_key, X-API-Key, and auth-header share the same classification path.
func IsSecretKey(key string) bool {
	normalized := normalizeKeyName(key)
	for _, exact := range exactSecretKeyNames {
		if normalized == exact {
			return true
		}
	}
	for _, fragment := range secretKeyFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func normalizeKeyName(key string) string {
	var b strings.Builder
	b.Grow(len(key))
	for _, r := range key {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// IsSecretValue reports whether value contains a token-shaped credential.
func IsSecretValue(value string) bool {
	return credentialPattern.MatchString(value)
}

// String redacts token-shaped credential fragments from value.
func String(value string) string {
	return credentialPattern.ReplaceAllString(value, Value)
}

// FormatStringMap renders a stable key-sorted map with token-shaped values
// redacted, matching the legacy MCP status text format.
func FormatStringMap(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+MapValue(key, values[key]))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
