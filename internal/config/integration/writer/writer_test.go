package integration_test

import (
	. "github.com/TrebuchetDynamics/gormes-agent/internal/config"

	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigWriter_WriteTOMLValueAcceptsUpdatesSection proves
// `gormes config set updates.pre_update_backup true` and
// `... updates.backup_keep 7` round-trip through both the writer (TOML
// shape) and Load (typed struct) with no extra interpretation. Without
// this, operators who learned about the `[updates]` table from `gormes
// update --help` or release notes would hit "unknown section".
func TestConfigWriter_WriteTOMLValueAcceptsUpdatesSection(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.toml")

	if err := WriteTOMLValue(path, "updates.pre_update_backup", "true"); err != nil {
		t.Fatalf("WriteTOMLValue updates.pre_update_backup: %v", err)
	}
	if err := WriteTOMLValue(path, "updates.backup_keep", "7"); err != nil {
		t.Fatalf("WriteTOMLValue updates.backup_keep: %v", err)
	}
	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), "[updates]") {
		t.Fatalf("config.toml missing [updates] section:\n%s", body)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load after writes: %v", err)
	}
	if !cfg.Updates.PreUpdateBackup {
		t.Errorf("after WriteTOMLValue updates.pre_update_backup=true, Load returned false")
	}
	if cfg.Updates.BackupKeep != 7 {
		t.Errorf("after WriteTOMLValue updates.backup_keep=7, Load returned %d", cfg.Updates.BackupKeep)
	}
}

func TestConfigWriter_WriteTOMLValueNestedAccountAllowedChannelIDsStayStrings(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	t.Setenv("GORMES_DISCORD_TOKEN", "")
	t.Setenv("GORMES_SLACK_ENABLED", "")
	t.Setenv("GORMES_SLACK_BOT_TOKEN", "")
	t.Setenv("GORMES_SLACK_APP_TOKEN", "")
	t.Setenv("GORMES_SLACK_CHANNEL_ID", "")

	const snowflake = "123456789012345678"
	path := ConfigPath()
	for _, key := range []string{
		"discord.accounts.main.allowed_channel_id",
		"slack.accounts.main.allowed_channel_id",
	} {
		if err := WriteTOMLValue(path, key, snowflake); err != nil {
			t.Fatalf("WriteTOMLValue %s: %v", key, err)
		}
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load after nested account channel writes: %v", err)
	}
	if got := cfg.Discord.Accounts["main"].AllowedChannelID; got != snowflake {
		t.Fatalf("Discord account allowed_channel_id = %q, want %q", got, snowflake)
	}
	if got := cfg.Slack.Accounts["main"].AllowedChannelID; got != snowflake {
		t.Fatalf("Slack account allowed_channel_id = %q, want %q", got, snowflake)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(body), "allowed_channel_id = "+snowflake) {
		t.Fatalf("nested allowed_channel_id was written as a TOML integer:\n%s", body)
	}
}
