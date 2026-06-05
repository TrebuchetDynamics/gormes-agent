package router

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"
)

// Config mirrors Gormes router config TOML/YAML fields.
type Config struct {
	Enabled    bool          `toml:"enabled" yaml:"enabled"`
	Listen     string        `toml:"listen" yaml:"listen"`
	APIKeys    []string      `toml:"api_keys" yaml:"api_keys"`
	APIKeyEnv  string        `toml:"api_key_env" yaml:"api_key_env"`
	RedactLogs bool          `toml:"redact_logs" yaml:"redact_logs"`
	SetupMode  string        `toml:"setup_mode" yaml:"setup_mode"`
	Routes     []RouteConfig `toml:"routes" yaml:"routes"`
	Fallback   []Fallback    `toml:"fallback" yaml:"fallback"`
}

// RouteConfig describes a provider route entry in router config.
type RouteConfig struct {
	Name      string                 `toml:"name" yaml:"name"`
	Provider  string                 `toml:"provider" yaml:"provider"`
	Model     string                 `toml:"model" yaml:"model"`
	Alias     string                 `toml:"alias" yaml:"alias"`
	BaseURL   string                 `toml:"base_url" yaml:"base_url"`
	APIKeyEnv string                 `toml:"api_key_env" yaml:"api_key_env"`
	APIKeyRef *credentials.SecretRef `toml:"api_key_ref" yaml:"api_key_ref" json:"api_key_ref,omitempty"`
	Transport string                 `toml:"transport" yaml:"transport"`
	Optional  bool                   `toml:"optional" yaml:"optional"`
	Weight    int                    `toml:"weight" yaml:"weight"`
}

// Fallback describes a router fallback rule.
type Fallback struct {
	From string   `toml:"from" yaml:"from"`
	To   string   `toml:"to" yaml:"to"`
	On   []string `toml:"on" yaml:"on"`
}

// Document converts router config into the TOML document shape written to disk.
func Document(router Config) map[string]any {
	out := map[string]any{
		"enabled":     router.Enabled,
		"redact_logs": router.RedactLogs,
	}
	if strings.TrimSpace(router.Listen) != "" {
		out["listen"] = strings.TrimSpace(router.Listen)
	}
	if keys := cleanStrings(router.APIKeys); len(keys) > 0 {
		out["api_keys"] = keys
	}
	if strings.TrimSpace(router.APIKeyEnv) != "" {
		out["api_key_env"] = strings.TrimSpace(router.APIKeyEnv)
	}
	if strings.TrimSpace(router.SetupMode) != "" {
		out["setup_mode"] = strings.TrimSpace(router.SetupMode)
	}
	if routes := routeDocuments(router.Routes); len(routes) > 0 {
		out["routes"] = routes
	}
	if fallback := fallbackDocuments(router.Fallback); len(fallback) > 0 {
		out["fallback"] = fallback
	}
	return out
}

func routeDocuments(routes []RouteConfig) []map[string]any {
	out := make([]map[string]any, 0, len(routes))
	for _, route := range routes {
		item := map[string]any{}
		if strings.TrimSpace(route.Name) != "" {
			item["name"] = strings.TrimSpace(route.Name)
		}
		if strings.TrimSpace(route.Provider) != "" {
			item["provider"] = strings.TrimSpace(route.Provider)
		}
		if strings.TrimSpace(route.Model) != "" {
			item["model"] = strings.TrimSpace(route.Model)
		}
		if strings.TrimSpace(route.Alias) != "" {
			item["alias"] = strings.TrimSpace(route.Alias)
		}
		if strings.TrimSpace(route.BaseURL) != "" {
			item["base_url"] = strings.TrimSpace(route.BaseURL)
		}
		if strings.TrimSpace(route.APIKeyEnv) != "" {
			item["api_key_env"] = strings.TrimSpace(route.APIKeyEnv)
		}
		if route.APIKeyRef != nil {
			item["api_key_ref"] = route.APIKeyRef
		}
		if strings.TrimSpace(route.Transport) != "" {
			item["transport"] = strings.TrimSpace(route.Transport)
		}
		if route.Optional {
			item["optional"] = true
		}
		if route.Weight != 0 {
			item["weight"] = int64(route.Weight)
		}
		if len(item) > 0 {
			out = append(out, item)
		}
	}
	return out
}

func fallbackDocuments(rules []Fallback) []map[string]any {
	out := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		item := map[string]any{}
		if strings.TrimSpace(rule.From) != "" {
			item["from"] = strings.TrimSpace(rule.From)
		}
		if strings.TrimSpace(rule.To) != "" {
			item["to"] = strings.TrimSpace(rule.To)
		}
		if on := cleanStrings(rule.On); len(on) > 0 {
			item["on"] = on
		}
		if len(item) > 0 {
			out = append(out, item)
		}
	}
	return out
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}
