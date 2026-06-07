package channels

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/channels/teams"

const TeamsDefaultPort = teams.DefaultPort

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
	return c.asTeamsCfg().EffectivePort()
}

func (c TeamsCfg) AllowedUserIDs() []string {
	return c.asTeamsCfg().AllowedUserIDs()
}

func (c TeamsCfg) MissingCredentials() []string {
	return c.asTeamsCfg().MissingCredentials()
}

func (c TeamsCfg) RedactedStatus() string {
	return c.asTeamsCfg().RedactedStatus()
}

func (c TeamsCfg) asTeamsCfg() teams.Cfg {
	return teams.Cfg{
		Enabled:       c.Enabled,
		ClientID:      c.ClientID,
		ClientSecret:  c.ClientSecret,
		TenantID:      c.TenantID,
		Port:          c.Port,
		AllowedUsers:  c.AllowedUsers,
		AllowAllUsers: c.AllowAllUsers,
	}
}
