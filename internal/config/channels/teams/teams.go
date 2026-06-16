package teams

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/channels/internal/textlist"
)

const DefaultPort = 3978

type Cfg struct {
	Enabled       bool     `toml:"enabled" yaml:"enabled"`
	ClientID      string   `toml:"client_id" yaml:"client_id"`
	ClientSecret  string   `toml:"client_secret" yaml:"client_secret"`
	TenantID      string   `toml:"tenant_id" yaml:"tenant_id"`
	Port          int      `toml:"port" yaml:"port"`
	AllowedUsers  []string `toml:"allowed_users" yaml:"allowed_users"`
	AllowAllUsers bool     `toml:"allow_all_users" yaml:"allow_all_users"`
}

func (c Cfg) EffectivePort() int {
	if c.Port > 0 {
		return c.Port
	}
	return DefaultPort
}

func (c Cfg) AllowedUserIDs() []string {
	return textlist.Compact(c.AllowedUsers)
}

func (c Cfg) MissingCredentials() []string {
	return textlist.MissingBlank([]textlist.Field{
		{Name: "client_id", Value: c.ClientID},
		{Name: "client_secret", Value: c.ClientSecret},
		{Name: "tenant_id", Value: c.TenantID},
	})
}

func (c Cfg) RedactedStatus() string {
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
