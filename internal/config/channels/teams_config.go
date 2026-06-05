package channels

import (
	"fmt"
	"strings"
)

const TeamsDefaultPort = 3978

type TeamsCfg struct {
	Enabled       bool     `toml:"enabled" yaml:"enabled"`
	ClientID      string   `toml:"client_id" yaml:"client_id"`
	ClientSecret  string   `toml:"client_secret" yaml:"client_secret"`
	TenantID      string   `toml:"tenant_id" yaml:"tenant_id"`
	Port          int      `toml:"port" yaml:"port"`
	AllowedUsers  []string `toml:"allowed_users" yaml:"allowed_users"`
	AllowAllUsers bool     `toml:"allow_all_users" yaml:"allow_all_users"`
}

func (c TeamsCfg) EffectivePort() int {
	if c.Port > 0 {
		return c.Port
	}
	return TeamsDefaultPort
}

func (c TeamsCfg) AllowedUserIDs() []string {
	return compactStrings(c.AllowedUsers)
}

func (c TeamsCfg) MissingCredentials() []string {
	missing := []string{}
	if strings.TrimSpace(c.ClientID) == "" {
		missing = append(missing, "client_id")
	}
	if strings.TrimSpace(c.ClientSecret) == "" {
		missing = append(missing, "client_secret")
	}
	if strings.TrimSpace(c.TenantID) == "" {
		missing = append(missing, "tenant_id")
	}
	return missing
}

func (c TeamsCfg) RedactedStatus() string {
	parts := []string{}
	if missing := c.MissingCredentials(); len(missing) > 0 {
		parts = append(parts, "missing_credentials="+strings.Join(missing, ","))
	} else {
		parts = append(parts, "configured")
	}
	parts = append(parts, fmt.Sprintf("port=%d", c.EffectivePort()))
	if allowed := c.AllowedUserIDs(); len(allowed) > 0 {
		parts = append(parts, fmt.Sprintf("allowed_users=%d", len(allowed)))
	}
	if c.AllowAllUsers {
		parts = append(parts, "allow_all_users=true")
	}
	return strings.Join(parts, " ")
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
