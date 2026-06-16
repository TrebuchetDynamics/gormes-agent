package urlpolicy

import (
	"net/url"
	"strings"
)

// MatchHostAgainstRule matches a normalized or raw host against a policy pattern.
// It supports exact match, suffix match (example.com matches sub.example.com),
// and wildcard subdomain match (*.example.com).
func MatchHostAgainstRule(host, pattern string) bool {
	if host == "" || pattern == "" {
		return false
	}
	hostLower := strings.ToLower(host)
	patternLower := strings.ToLower(pattern)

	if strings.HasPrefix(patternLower, "*.") {
		suffix := patternLower[2:]
		return strings.HasSuffix(hostLower, "."+suffix) && hostLower != suffix
	}

	return strings.HasSuffix(hostLower, "."+patternLower) || hostLower == patternLower
}

// ExtractHostFromURL extracts and normalizes the host from a URL-like string.
func ExtractHostFromURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	parsed, err := url.Parse(rawURL)
	if err == nil && parsed.Hostname() != "" {
		return NormalizeHost(parsed.Hostname())
	}

	if idx := strings.Index(rawURL, "://"); idx == -1 {
		candidate := stripURLTail(rawURL)
		if strings.Count(candidate, ":") > 1 && !strings.HasPrefix(candidate, "[") {
			return NormalizeHost(candidate)
		}
		parsed, err = url.Parse("//" + rawURL)
		if err == nil && parsed.Hostname() != "" {
			return NormalizeHost(parsed.Hostname())
		}
	}

	host := stripURLTail(rawURL)
	host = StripPort(host)
	return NormalizeHost(host)
}

func stripURLTail(raw string) string {
	if idx := strings.IndexAny(raw, "/?#"); idx >= 0 {
		return raw[:idx]
	}
	return raw
}

// StripPort strips the port from a host string, handling IPv6 correctly.
func StripPort(host string) string {
	host = strings.TrimSpace(host)
	if strings.HasPrefix(host, "[") {
		if closeBracket := strings.Index(host, "]"); closeBracket >= 0 {
			return host[:closeBracket+1]
		}
	}

	if strings.Count(host, ":") > 1 {
		return host
	}

	if colonIdx := strings.LastIndex(host, ":"); colonIdx > 0 {
		return host[:colonIdx]
	}
	return host
}

// NormalizeHost normalizes a hostname for comparison.
func NormalizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.ToLower(host)
	host = strings.TrimSuffix(host, ".")
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	if strings.HasPrefix(host, "www.") {
		host = host[4:]
	}
	return host
}

// NormalizeBlocklistRule normalizes a raw URL/host blocklist rule to host form.
func NormalizeBlocklistRule(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return ""
	}
	if idx := strings.Index(raw, "://"); idx != -1 {
		parsed, err := url.Parse(raw)
		if err == nil && parsed.Hostname() != "" {
			raw = parsed.Hostname()
		}
	}
	if slashIdx := strings.Index(raw, "/"); slashIdx != -1 {
		raw = raw[:slashIdx]
	}
	return NormalizeHost(raw)
}
