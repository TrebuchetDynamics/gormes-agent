package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTeamsConfigAndPluginRegistration(t *testing.T) {
	t.Run("explicit config defaults port and redacts missing credentials", func(t *testing.T) {
		setupTeamsConfigTestHome(t)
		writeTeamsConfig(t, `
[teams]
enabled = true
client_id = "cfg-client"
client_secret = "cfg-secret"
tenant_id = "cfg-tenant"
allowed_users = ["aad-1", "aad-2"]
`)

		cfg, err := Load(nil)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if !cfg.Teams.Enabled || cfg.Teams.ClientID != "cfg-client" || cfg.Teams.ClientSecret != "cfg-secret" || cfg.Teams.TenantID != "cfg-tenant" {
			t.Fatalf("teams config = %+v", cfg.Teams)
		}
		if cfg.Teams.EffectivePort() != TeamsDefaultPort {
			t.Fatalf("EffectivePort = %d, want default %d", cfg.Teams.EffectivePort(), TeamsDefaultPort)
		}
		if !reflect.DeepEqual(cfg.Teams.AllowedUserIDs(), []string{"aad-1", "aad-2"}) {
			t.Fatalf("AllowedUserIDs = %v", cfg.Teams.AllowedUserIDs())
		}
		if status := cfg.Teams.RedactedStatus(); strings.Contains(status, "cfg-secret") || !strings.Contains(status, "configured") {
			t.Fatalf("RedactedStatus = %q", status)
		}
	})

	t.Run("teams env wins over file config", func(t *testing.T) {
		setupTeamsConfigTestHome(t)
		writeTeamsConfig(t, `
[teams]
enabled = false
client_id = "cfg-client"
client_secret = "cfg-secret"
tenant_id = "cfg-tenant"
port = 4000
`)
		t.Setenv("GORMES_TEAMS_ENABLED", "true")
		t.Setenv("TEAMS_CLIENT_ID", "env-client")
		t.Setenv("TEAMS_CLIENT_SECRET", "env-secret")
		t.Setenv("TEAMS_TENANT_ID", "env-tenant")
		t.Setenv("TEAMS_PORT", "5000")
		t.Setenv("TEAMS_ALLOWED_USERS", "aad-env-1, aad-env-2")

		cfg, err := Load(nil)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if !cfg.Teams.Enabled || cfg.Teams.ClientID != "env-client" || cfg.Teams.ClientSecret != "env-secret" || cfg.Teams.TenantID != "env-tenant" {
			t.Fatalf("teams env config = %+v", cfg.Teams)
		}
		if cfg.Teams.EffectivePort() != 5000 {
			t.Fatalf("EffectivePort = %d, want 5000", cfg.Teams.EffectivePort())
		}
		if !reflect.DeepEqual(cfg.Teams.AllowedUserIDs(), []string{"aad-env-1", "aad-env-2"}) {
			t.Fatalf("AllowedUserIDs = %v", cfg.Teams.AllowedUserIDs())
		}
	})

	t.Run("invalid env port returns evidence", func(t *testing.T) {
		setupTeamsConfigTestHome(t)
		t.Setenv("GORMES_TEAMS_ENABLED", "true")
		t.Setenv("TEAMS_PORT", "not-an-int")
		if _, err := Load(nil); err == nil || !strings.Contains(err.Error(), "TEAMS_PORT") {
			t.Fatalf("Load err = %v, want TEAMS_PORT parse error", err)
		}
	})
}

func setupTeamsConfigTestHome(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
}

func writeTeamsConfig(t *testing.T, text string) {
	t.Helper()
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
