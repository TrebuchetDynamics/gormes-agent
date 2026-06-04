package setuprouter

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

var aliasSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

// FallbackRules builds primary-chat fallback rules for every other route alias.
func FallbackRules(routes []config.RouterRouteCfg) []config.RouterFallbackCfg {
	if len(routes) < 2 {
		return nil
	}
	primary := ""
	for _, route := range routes {
		if strings.EqualFold(strings.TrimSpace(route.Alias), "primary-chat") {
			primary = "primary-chat"
			break
		}
	}
	if primary == "" {
		return nil
	}
	rules := make([]config.RouterFallbackCfg, 0, len(routes)-1)
	for _, route := range routes {
		alias := strings.TrimSpace(route.Alias)
		if alias == "" || alias == primary {
			continue
		}
		rules = append(rules, config.RouterFallbackCfg{
			From: primary,
			To:   alias,
			On:   []string{"rate_limit", "server_error", "timeout", "connection_failure"},
		})
	}
	return rules
}

// RouteLabels returns operator-facing caution labels for setup router receipts.
func RouteLabels(route config.RouterRouteCfg) []string {
	labels := []string{}
	text := strings.ToLower(strings.Join([]string{route.Name, route.Alias, route.Provider, route.Model}, " "))
	if strings.Contains(text, ":free") || strings.Contains(text, "free-tier") || strings.Contains(text, "free_tier") {
		labels = append(labels, "requires your provider account/API key; quotas are provider-controlled")
	}
	if route.Optional {
		labels = append(labels, "optional; only enabled if already installed and healthy")
	}
	return labels
}

// OpenAIBaseURL converts a router listen address into its OpenAI-compatible /v1 base URL.
func OpenAIBaseURL(listen, defaultListen string) string {
	listen = strings.TrimSpace(listen)
	if listen == "" {
		listen = defaultListen
	}
	if !strings.Contains(listen, "://") {
		listen = "http://" + listen
	}
	parsed, err := url.Parse(listen)
	if err != nil || parsed.Host == "" {
		return strings.TrimRight(listen, "/") + "/v1"
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/v1"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.User = nil
	return parsed.String()
}

// Slug normalizes a provider or route label for generated aliases.
func Slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = aliasSanitizer.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "route"
	}
	return value
}
