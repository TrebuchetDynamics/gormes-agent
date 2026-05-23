package config

import (
	"encoding/json"
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

func TestWriteProfileConfigV2AppliesRootProfilesAtomicallyAndPreservesOtherSections(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg"))
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ConfigPath(), []byte(`
config_version = 2

[hermes]
provider = "openrouter"
model = "moonshotai/kimi-k2.6"

[profiles.main]
enabled = true
name = ""
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	main := cfg.Profiles["main"]
	main.Name = "Main desk"
	main.Workspaces = []string{"/workspace/main"}
	main.Providers = map[string]ProfileProviderCfg{
		"openrouter": {Enabled: true, Credential: "main-openrouter", DefaultModel: "openrouter/auto"},
	}
	main.Channels = map[string]ProfileChannelCfg{
		"telegram": {Enabled: true, Credential: "main-telegram", AllowedUsers: []string{"juan"}},
	}
	cfg.Profiles["main"] = main
	cfg.Credentials = map[string]CredentialCfg{
		"main-openrouter": {Kind: "provider", Provider: "openrouter", OwnerProfile: "main", SecretRef: &SecretRef{Source: SecretRefSourceEnv, ID: "GORMES_MAIN_OPENROUTER_API_KEY"}},
		"main-telegram":   {Kind: "channel", Channel: "telegram", OwnerProfile: "main", SecretRef: &SecretRef{Source: SecretRefSourceEnv, ID: "GORMES_MAIN_TELEGRAM_BOT_TOKEN"}},
	}

	if err := WriteProfileConfigV2(ConfigPath(), cfg); err != nil {
		t.Fatalf("WriteProfileConfigV2: %v", err)
	}

	raw, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	body := string(raw)
	for _, want := range []string{
		"config_version = 2",
		"provider = 'openrouter'",
		"model = 'moonshotai/kimi-k2.6'",
		"[profiles.main]",
		"name = 'Main desk'",
		"workspaces = ['/workspace/main']",
		"[profiles.main.providers.openrouter]",
		"credential = 'main-openrouter'",
		"[profiles.main.channels.telegram]",
		"allowed_users = ['juan']",
		"[credentials.main-openrouter]",
		"owner_profile = 'main'",
		"secret_ref",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("written config missing %q:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{"GORMES_MAIN_OPENROUTER_API_KEY =", "sk-", "bot-token"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("written config leaked raw secret-looking value %q:\n%s", forbidden, body)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "profiles", "main", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("WriteProfileConfigV2 must not create per-profile config.toml, stat err=%v", err)
	}

	loaded, err := Load(nil)
	if err != nil {
		t.Fatalf("Load after WriteProfileConfigV2: %v", err)
	}
	if loaded.Hermes.Provider != "openrouter" || loaded.Hermes.Model != "moonshotai/kimi-k2.6" {
		t.Fatalf("non-profile sections not preserved: %+v", loaded.Hermes)
	}
	if got := loaded.Profiles["main"].Channels["telegram"].AllowedUsers; !reflect.DeepEqual(got, []string{"juan"}) {
		t.Fatalf("allowed_users = %#v, want juan", got)
	}
	if ref := loaded.Credentials["main-openrouter"].SecretRef; ref == nil || ref.Source != SecretRefSourceEnv || ref.ID != "GORMES_MAIN_OPENROUTER_API_KEY" {
		t.Fatalf("loaded secret ref = %+v", ref)
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

func TestNavivoxProfileRoutingListsEnabledProfileDefaultsSafely(t *testing.T) {
	cfg := Config{Profiles: map[string]ProfileCfg{
		"sleeping": {
			Enabled: true,
			Name:    "Sleeping",
		},
		"mineru": {
			Enabled:    true,
			Name:       "Mineru Ops",
			Workspaces: []string{" /srv/gormes ", "", "/srv/navivox"},
			Providers: map[string]ProfileProviderCfg{
				"openai-codex": {Enabled: true, Credential: "provider-secret-ref"},
				"disabled":     {Enabled: false},
			},
			Channels: map[string]ProfileChannelCfg{
				"telegram": {Enabled: true, Credential: "telegram-token-ref"},
				"navivox":  {Enabled: true},
				"slack":    {Enabled: false},
			},
		},
		"tulin": {
			Enabled:    false,
			Name:       "Tulin",
			Workspaces: []string{"/srv/tulin"},
		},
	}}

	routing := cfg.NavivoxProfileRouting()
	want := NavivoxProfileRoutingReport{Profiles: []NavivoxProfileRoute{
		{
			ProfileID:   "mineru",
			DisplayName: "Mineru Ops",
			Workspaces:  []string{"/srv/gormes", "/srv/navivox"},
			Providers:   []string{"openai-codex"},
			Channels:    []string{"navivox", "telegram"},
		},
		{
			ProfileID:   "sleeping",
			DisplayName: "Sleeping",
		},
	}}
	if !reflect.DeepEqual(routing, want) {
		t.Fatalf("NavivoxProfileRouting() = %#v, want %#v", routing, want)
	}
	payload, err := json.Marshal(routing)
	if err != nil {
		t.Fatalf("Marshal routing: %v", err)
	}
	for _, banned := range []string{"provider-secret-ref", "telegram-token-ref"} {
		if strings.Contains(string(payload), banned) {
			t.Fatalf("routing payload leaked credential reference %q: %s", banned, payload)
		}
	}
}

func TestNavivoxProfileRoutingBuildsServerScopedProfileFleetSafely(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg"))
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ConfigPath(), []byte(`
config_version = 2

[profiles.main]
enabled = true
name = "Main Desk"
workspaces = ["/srv/main"]

[profiles.main.channels.navivox]
enabled = true
credential = "main-navivox"
servers = ["local", "tailnet"]
voice_profile = "private-voice-main"

[profiles.tulin]
enabled = true
name = "Tulin"
workspaces = ["/srv/tulin"]

[profiles.tulin.channels.navivox]
enabled = true
servers = ["local"]

[profiles.sleeping]
enabled = false
name = "Sleeping"

[navivox]
enabled = false

[navivox.servers.local]
enabled = true
bind = "127.0.0.1:8787"
profiles = ["main", "tulin", "sleeping", "missing"]
transports = ["http", "ws"]
capabilities = ["connect_and_talk"]

[navivox.servers.tailnet]
enabled = true
bind = "100.64.0.5:8787"
profiles = ["main"]
transports = ["ws"]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Navivox.Servers["local"].Bind; got != "127.0.0.1:8787" {
		t.Fatalf("navivox.servers.local.bind = %q, want 127.0.0.1:8787", got)
	}

	routing := cfg.NavivoxProfileRouting()
	if got := navivoxServerRouteIDs(routing.Servers); !reflect.DeepEqual(got, []string{"local", "tailnet"}) {
		t.Fatalf("server IDs = %#v, want local/tailnet", got)
	}
	local := navivoxServerRouteByID(t, routing.Servers, "local")
	if got := navivoxProfileRouteIDs(local.Profiles); !reflect.DeepEqual(got, []string{"main", "tulin"}) {
		t.Fatalf("local profile IDs = %#v, want main/tulin", got)
	}
	main := local.Profiles[0]
	if !main.Ready || !main.CredentialConfigured || !main.VoiceProfileConfigured || main.DisplayName != "Main Desk" {
		t.Fatalf("local main route = %+v, want ready main with redacted credential/voice evidence", main)
	}
	if !reflect.DeepEqual(main.ServerIDs, []string{"local"}) {
		t.Fatalf("main.ServerIDs = %#v, want local", main.ServerIDs)
	}
	if !navivoxRouteHasWarning(local.Warnings, "sleeping", "navivox_profile_unavailable") || !navivoxRouteHasWarning(local.Warnings, "missing", "navivox_profile_unavailable") {
		t.Fatalf("local warnings = %#v, want sleeping and missing profile unavailable warnings", local.Warnings)
	}
	tailnet := navivoxServerRouteByID(t, routing.Servers, "tailnet")
	if got := navivoxProfileRouteIDs(tailnet.Profiles); !reflect.DeepEqual(got, []string{"main"}) {
		t.Fatalf("tailnet profile IDs = %#v, want main", got)
	}
	if got := navivoxProfileRouteIDs(routing.Profiles); !reflect.DeepEqual(got, []string{"main", "tulin"}) {
		t.Fatalf("top-level routed profile IDs = %#v, want unique main/tulin", got)
	}

	payload, err := json.Marshal(routing)
	if err != nil {
		t.Fatalf("Marshal routing: %v", err)
	}
	for _, banned := range []string{"main-navivox", "private-voice-main", "default_profile"} {
		if strings.Contains(string(payload), banned) {
			t.Fatalf("server-scoped routing leaked or emitted banned value %q: %s", banned, payload)
		}
	}
}

func navivoxServerRouteIDs(routes []NavivoxServerRoute) []string {
	out := make([]string, 0, len(routes))
	for _, route := range routes {
		out = append(out, route.ServerID)
	}
	return out
}

func navivoxServerRouteByID(t *testing.T, routes []NavivoxServerRoute, id string) NavivoxServerRoute {
	t.Helper()
	for _, route := range routes {
		if route.ServerID == id {
			return route
		}
	}
	t.Fatalf("server route %q missing from %#v", id, routes)
	return NavivoxServerRoute{}
}

func navivoxProfileRouteIDs(routes []NavivoxProfileRoute) []string {
	out := make([]string, 0, len(routes))
	for _, route := range routes {
		out = append(out, route.ProfileID)
	}
	return out
}

func navivoxRouteHasWarning(warnings []NavivoxProfileRouteWarning, profileID, code string) bool {
	for _, warning := range warnings {
		if warning.ProfileID == profileID && warning.Code == code {
			return true
		}
	}
	return false
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
