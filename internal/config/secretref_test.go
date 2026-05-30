package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecretRefConfigLoadsProvidersFromTOML(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[secrets.defaults]
env = "default"
file = "filemain"

[secrets.providers.filemain]
source = "file"
path = "/tmp/gormes-secrets.json"
mode = "json"
allow_insecure_path = true
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Secrets.Defaults.File != "filemain" {
		t.Fatalf("Secrets.Defaults.File = %q, want filemain", cfg.Secrets.Defaults.File)
	}
	provider := cfg.Secrets.Providers["filemain"]
	if provider.Source != SecretRefSourceFile || provider.Path != "/tmp/gormes-secrets.json" || provider.Mode != SecretProviderModeJSON || !provider.AllowInsecurePath {
		t.Fatalf("Secrets provider filemain = %+v", provider)
	}
}

func TestSecretRefRuntimeConfigLoadsGatewayCredentialRefsFromTOML(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	for _, name := range []string{
		"GORMES_API_KEY",
		"GORMES_TELEGRAM_TOKEN",
		"GORMES_DISCORD_TOKEN",
		"GORMES_SLACK_ENABLED",
	} {
		t.Setenv(name, "")
	}
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[hermes]
endpoint = "https://provider.example/v1"
api_key = "stale-provider"

[hermes.api_key_ref]
source = "env"
id = "GORMES_API_KEY"

[telegram]
bot_token = "stale-telegram"
allowed_chat_id = 42

[telegram.bot_token_ref]
source = "env"
provider = "default"
id = "GORMES_TELEGRAM_TOKEN"

[discord]
token = "stale-discord"
allowed_channel_id = "C123"

[discord.token_ref]
source = "env"
id = "GORMES_DISCORD_TOKEN"

[slack]
enabled = true
bot_token = "stale-slack-bot"
app_token = "stale-slack-app"

[slack.bot_token_ref]
source = "env"
id = "GORMES_SLACK_BOT_TOKEN"

[slack.app_token_ref]
source = "env"
id = "GORMES_SLACK_APP_TOKEN"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	assertLoadedSecretRef(t, cfg.Hermes.APIKeyRef, SecretRefSourceEnv, "", "GORMES_API_KEY")
	assertLoadedSecretRef(t, cfg.Telegram.BotTokenRef, SecretRefSourceEnv, "default", "GORMES_TELEGRAM_TOKEN")
	assertLoadedSecretRef(t, cfg.Discord.TokenRef, SecretRefSourceEnv, "", "GORMES_DISCORD_TOKEN")
	assertLoadedSecretRef(t, cfg.Slack.BotTokenRef, SecretRefSourceEnv, "", "GORMES_SLACK_BOT_TOKEN")
	assertLoadedSecretRef(t, cfg.Slack.AppTokenRef, SecretRefSourceEnv, "", "GORMES_SLACK_APP_TOKEN")
}

func assertLoadedSecretRef(t *testing.T, ref *SecretRef, source SecretRefSource, provider string, id string) {
	t.Helper()
	if ref == nil {
		t.Fatalf("SecretRef %s is nil", id)
	}
	if ref.Source != source || ref.Provider != provider || ref.ID != id {
		t.Fatalf("SecretRef %s = %+v, want source=%s provider=%q id=%s", id, *ref, source, provider, id)
	}
}
