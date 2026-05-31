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

// IsSecretKey reports whether key names credential-like material.
func IsSecretKey(key string) bool {
	lower := strings.ToLower(key)
	secretFragments := []string{
		"authorization",
		"auth_header",
		"api_key",
		"access_token",
		"refresh_token",
		"personal_access_token",
		"token",
		"secret",
		"password",
	}
	for _, fragment := range secretFragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
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
