package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProfileConfigV2FreshSeedDocument(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg"))

	if err := EnsureConfigFile(ConfigPath()); err != nil {
		t.Fatalf("EnsureConfigFile: %v", err)
	}

	body, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	text := string(body)
	for _, want := range []string{
		"config_version = 2",
		"[profiles.main]",
		"enabled = true",
		"name = ",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("fresh v2 config missing %q:\n%s", want, text)
		}
	}
	for _, banned := range []string{"_config_version", "active_profile", "default_profile"} {
		if strings.Contains(text, banned) {
			t.Fatalf("fresh v2 config must not contain %q:\n%s", banned, text)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "profiles", "main", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("fresh v2 config must not write per-profile config.toml, stat err=%v", err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ConfigVersion != 2 {
		t.Fatalf("ConfigVersion = %d, want 2", cfg.ConfigVersion)
	}
	main, ok := cfg.Profiles["main"]
	if !ok {
		t.Fatalf("profiles.main missing from loaded config: %+v", cfg.Profiles)
	}
	if !main.Enabled || main.Name != "" {
		t.Fatalf("profiles.main = %+v, want enabled true with empty display name", main)
	}
}

func TestProfileConfigV2LoadsAllEnabledProfilesAndCredentials(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg"))
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte(`
config_version = 2

[profiles.main]
enabled = true
name = "Yunobo"
workspaces = ["/srv/arenaton", "/srv/gormes"]

[profiles.main.providers.openrouter]
enabled = true
credential = "main-openrouter"
default_model = "openrouter/auto"
allowed_models = ["openai/gpt-5.2", "anthropic/claude-sonnet-4.5"]

[profiles.main.channels.telegram]
enabled = true
credential = "main-telegram"
allowed_chats = ["12345"]
allowed_users = ["juan"]
tool_progress = "new"

[profiles.tulin]
enabled = true
name = "Tulin"

[profiles.sleeping]
enabled = false
name = "Sleeping"

[credentials.main-openrouter]
kind = "provider"
provider = "openrouter"
owner_profile = "main"
secret_ref = { source = "env", id = "GORMES_MAIN_OPENROUTER_API_KEY" }

[credentials.main-telegram]
kind = "channel"
channel = "telegram"
owner_profile = "main"
secret_ref = { source = "env", id = "GORMES_MAIN_TELEGRAM_BOT_TOKEN" }
`)
	if err := os.WriteFile(ConfigPath(), body, 0o600); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.ProfileConfigV2Available() {
		t.Fatal("ProfileConfigV2Available = false, want true")
	}
	services := cfg.EnabledProfileServices()
	var gotIDs []string
	for _, service := range services {
		gotIDs = append(gotIDs, service.ID)
	}
	if want := []string{"main", "tulin"}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("EnabledProfileServices IDs = %#v, want %#v", gotIDs, want)
	}
	main := cfg.Profiles["main"]
	if got := main.Providers["openrouter"].Credential; got != "main-openrouter" {
		t.Fatalf("profiles.main.providers.openrouter.credential = %q", got)
	}
	if got := main.Channels["telegram"].AllowedChats; !reflect.DeepEqual(got, []string{"12345"}) {
		t.Fatalf("profiles.main.channels.telegram.allowed_chats = %#v", got)
	}
	cred := cfg.Credentials["main-openrouter"]
	if cred.OwnerProfile != "main" || cred.Provider != "openrouter" {
		t.Fatalf("credential owner/provider = %+v", cred)
	}
	if cred.SecretRef == nil || cred.SecretRef.Source != SecretRefSourceEnv || cred.SecretRef.ID != "GORMES_MAIN_OPENROUTER_API_KEY" {
		t.Fatalf("credential SecretRef = %+v", cred.SecretRef)
	}
	if strings.Contains(string(body), "sk-") || strings.Contains(string(body), "bot-token") {
		t.Fatalf("fixture should model secret refs only, got raw secret-looking data:\n%s", body)
	}
}

func TestProfileConfigV2RejectsInvalidProfileIDButAllowsDisplayNameDrift(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg"))
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ConfigPath(), []byte(`
config_version = 2

[profiles."../bad"]
enabled = true
name = "Bad"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "profile id") {
		t.Fatalf("Load invalid profile id err = %v, want profile id validation error", err)
	}

	if err := os.WriteFile(ConfigPath(), []byte(`
config_version = 2

[profiles.yunobo]
enabled = true
name = "Juan's builder"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load valid profile/display-name drift: %v", err)
	}
	if cfg.Profiles["yunobo"].Name != "Juan's builder" {
		t.Fatalf("display name was forced to profile id: %+v", cfg.Profiles["yunobo"])
	}
}

func TestProfileConfigV2LegacyFallbackRemainsReadable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg"))
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ConfigPath(), []byte(`
_config_version = 1

[hermes]
endpoint = "https://example.invalid/v1"
model = "legacy-model"
provider = "openai"

[agents.defaults]
workspaces = ["/legacy/workspace"]
channels = ["telegram"]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load legacy config: %v", err)
	}
	if cfg.ProfileConfigV2Available() {
		t.Fatal("legacy config without [profiles] reported v2 profile config available")
	}
	if cfg.Hermes.Endpoint != "https://example.invalid/v1" || cfg.Hermes.Model != "legacy-model" {
		t.Fatalf("legacy hermes fields not preserved: %+v", cfg.Hermes)
	}
	if got := cfg.Agents.Defaults.Workspaces; !reflect.DeepEqual(got, []string{"/legacy/workspace"}) {
		t.Fatalf("legacy agents.defaults.workspaces = %#v", got)
	}
}
