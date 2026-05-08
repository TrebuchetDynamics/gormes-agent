package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
)

func TestMain(m *testing.M) {
	hermesHome, err := os.MkdirTemp("", "gormes-config-test-hermes-home-*")
	if err != nil {
		panic(err)
	}
	gormesHome, err := os.MkdirTemp("", "gormes-config-test-gormes-home-*")
	if err != nil {
		panic(err)
	}
	oldHermesHome, hadHermesHome := os.LookupEnv("HERMES_HOME")
	oldGormesHome, hadGormesHome := os.LookupEnv("GORMES_HOME")
	os.Setenv("HERMES_HOME", hermesHome)
	os.Setenv("GORMES_HOME", gormesHome)
	code := m.Run()
	if hadHermesHome {
		os.Setenv("HERMES_HOME", oldHermesHome)
	} else {
		os.Unsetenv("HERMES_HOME")
	}
	if hadGormesHome {
		os.Setenv("GORMES_HOME", oldGormesHome)
	} else {
		os.Unsetenv("GORMES_HOME")
	}
	_ = os.RemoveAll(hermesHome)
	_ = os.RemoveAll(gormesHome)
	os.Exit(code)
}

func TestLoad_BuiltinDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_API_KEY", "")
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hermes.Endpoint != "" {
		t.Errorf("default endpoint = %q, want empty so gateway does not silently dial implicit localhost", cfg.Hermes.Endpoint)
	}
	if cfg.Hermes.Model != "hermes-agent" {
		t.Errorf("default model = %q", cfg.Hermes.Model)
	}
	if cfg.Input.MaxBytes != 200_000 || cfg.Input.MaxLines != 10_000 {
		t.Errorf("default input limits = %d/%d", cfg.Input.MaxBytes, cfg.Input.MaxLines)
	}
	if !cfg.TUI.MouseTracking {
		t.Error("default TUI mouse tracking = false, want true")
	}
}

func TestLoad_TUIMouseTrackingDefaultsOnWhenTUISectionOmitsIt(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[tui]
theme = "light"
`), 0o644)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.TUI.MouseTracking {
		t.Error("TUI.MouseTracking = false with legacy [tui] section, want default-on compatibility")
	}
}

func TestLoad_TUIMouseTrackingFromFile(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[tui]
mouse_tracking = false
`), 0o644)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TUI.MouseTracking {
		t.Error("TUI.MouseTracking = true, want false from config.toml")
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[hermes]
endpoint = "http://file:8642"
`), 0o644)
	t.Setenv("GORMES_ENDPOINT", "http://env:8642")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hermes.Endpoint != "http://env:8642" {
		t.Errorf("endpoint = %q, want env", cfg.Hermes.Endpoint)
	}
}

func TestLoad_GatewayProxyURLFromConfigNormalizesTrailingSlash(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	t.Setenv("GATEWAY_PROXY_URL", "")
	dir := filepath.Join(cfgHome, "gormes")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[gateway]
proxy_url = "http://config-proxy:8642/"
`), 0o644)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Gateway.ProxyURL != "http://config-proxy:8642" {
		t.Errorf("Gateway.ProxyURL = %q, want normalized config proxy URL", cfg.Gateway.ProxyURL)
	}
}

func TestLoad_GatewayProxyEnvOverridesConfigAndBlankEnvIsUnset(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[gateway]
proxy_url = "http://config-proxy:8642/"
proxy_key = "config-secret"
`), 0o644)

	t.Setenv("GATEWAY_PROXY_URL", "  ")
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Gateway.ProxyURL != "http://config-proxy:8642" {
		t.Fatalf("blank env Gateway.ProxyURL = %q, want config fallback", cfg.Gateway.ProxyURL)
	}

	t.Setenv("GATEWAY_PROXY_URL", "http://env-proxy:8642/")
	t.Setenv("GATEWAY_PROXY_KEY", "env-secret")
	cfg, err = Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Gateway.ProxyURL != "http://env-proxy:8642" {
		t.Errorf("Gateway.ProxyURL = %q, want env proxy URL without trailing slash", cfg.Gateway.ProxyURL)
	}
	if cfg.Gateway.ProxyKey != "env-secret" {
		t.Errorf("Gateway.ProxyKey = %q, want env secret", cfg.Gateway.ProxyKey)
	}
}

func TestLoad_FlagOverridesEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GORMES_ENDPOINT", "http://env:8642")
	cfg, err := Load([]string{"--endpoint", "http://flag:8642"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hermes.Endpoint != "http://flag:8642" {
		t.Errorf("endpoint = %q, want flag", cfg.Hermes.Endpoint)
	}
}

func TestLoad_SecretsNeverOnFlags(t *testing.T) {
	// Sanity: --api-key must NOT be a valid flag (secrets live in env/TOML only).
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := Load([]string{"--api-key", "nope"})
	if err == nil {
		t.Error("expected --api-key to be rejected as an unknown flag")
	}
}

func TestLoad_TelegramDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Telegram.CoalesceMs != 1000 {
		t.Errorf("CoalesceMs default = %d, want 1000", cfg.Telegram.CoalesceMs)
	}
	if !cfg.Telegram.FirstRunDiscovery {
		t.Error("FirstRunDiscovery default = false, want true")
	}
}

func TestLoad_TelegramRequireMentionFields(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[telegram]
require_mention = true
bot_username = "gormes_bot"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Telegram.RequireMention {
		t.Error("RequireMention = false, want true")
	}
	if cfg.Telegram.BotUsername != "gormes_bot" {
		t.Errorf("BotUsername = %q, want %q", cfg.Telegram.BotUsername, "gormes_bot")
	}
}

func TestLoad_TelegramRequireMentionDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Telegram.RequireMention {
		t.Error("RequireMention default = true, want false")
	}
	if cfg.Telegram.BotUsername != "" {
		t.Errorf("BotUsername default = %q, want empty", cfg.Telegram.BotUsername)
	}
}

func TestLoad_TelegramRequireMentionInvalidType(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[telegram]
require_mention = "yes"
bot_username = "gormes_bot"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err == nil {
		t.Fatal("Load err = nil, want require_mention type error")
	}
	if !strings.Contains(err.Error(), "RequireMention") && !strings.Contains(err.Error(), "require_mention") {
		t.Fatalf("Load err = %v, want require_mention evidence", err)
	}
	if cfg.Telegram.RequireMention {
		t.Error("RequireMention = true after malformed config, want disabled")
	}
}

func TestLoad_TelegramFreshFinalAfterSeconds(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		cfg, err := Load(nil)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Telegram.FreshFinalAfterSeconds != 60 {
			t.Errorf("FreshFinalAfterSeconds default = %v, want 60", cfg.Telegram.FreshFinalAfterSeconds)
		}
	})

	t.Run("explicit zero disables", func(t *testing.T) {
		cfgHome := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", cfgHome)
		t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
		dir := filepath.Join(cfgHome, "gormes")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[telegram]
fresh_final_after_seconds = 0
`), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(nil)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Telegram.FreshFinalAfterSeconds != 0 {
			t.Errorf("FreshFinalAfterSeconds = %v, want explicit 0", cfg.Telegram.FreshFinalAfterSeconds)
		}
	})

	t.Run("nonzero toml", func(t *testing.T) {
		cfgHome := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", cfgHome)
		t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
		dir := filepath.Join(cfgHome, "gormes")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[telegram]
fresh_final_after_seconds = 12.5
`), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(nil)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Telegram.FreshFinalAfterSeconds != 12.5 {
			t.Errorf("FreshFinalAfterSeconds = %v, want 12.5", cfg.Telegram.FreshFinalAfterSeconds)
		}
	})
}

func TestLoad_TelegramEnvOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GORMES_TELEGRAM_TOKEN", "abc:xyz")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "99999")
	t.Setenv("GORMES_TELEGRAM_ALLOWED_USERS", "111, 222")
	t.Setenv("TELEGRAM_BOT_TOKEN", "ignored:alias")
	t.Setenv("TELEGRAM_HOME_CHANNEL", "88888")
	t.Setenv("TELEGRAM_ALLOWED_USERS", "333")
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Telegram.BotToken != "abc:xyz" {
		t.Errorf("BotToken = %q", cfg.Telegram.BotToken)
	}
	if cfg.Telegram.AllowedChatID != 99999 {
		t.Errorf("AllowedChatID = %d", cfg.Telegram.AllowedChatID)
	}
	if got, want := cfg.Telegram.AllowedUserIDs, []int64{111, 222}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("AllowedUserIDs = %v, want %v", got, want)
	}
}

func TestLoad_TelegramHermesEnvAliases(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("GORMES_TELEGRAM_ALLOWED_USERS", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "123:hermes-token")
	t.Setenv("TELEGRAM_HOME_CHANNEL", "-10042")
	t.Setenv("TELEGRAM_ALLOWED_USERS", "6586915095, 12345, invalid")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Telegram.BotToken != "123:hermes-token" {
		t.Errorf("BotToken = %q", cfg.Telegram.BotToken)
	}
	if cfg.Telegram.AllowedChatID != -10042 {
		t.Errorf("AllowedChatID = %d", cfg.Telegram.AllowedChatID)
	}
	if got, want := cfg.Telegram.AllowedUserIDs, []int64{6586915095, 12345}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("AllowedUserIDs = %v, want %v", got, want)
	}
}

func TestLoad_IgnoresHermesConfigYAML(t *testing.T) {
	cfgHome := t.TempDir()
	hermesHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	t.Setenv("HERMES_HOME", hermesHome)
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("GORMES_API_KEY", "")
	t.Setenv("GORMES_WEB_BACKEND", "")
	t.Setenv("GORMES_WEB_USE_GATEWAY", "")
	t.Setenv("GORMES_BROWSER_CDP_URL", "")
	t.Setenv("BROWSER_CDP_URL", "")
	t.Setenv("CHROME_REMOTE_DEBUGGING_URL", "")
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("GORMES_TELEGRAM_ALLOWED_USERS", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("TELEGRAM_HOME_CHANNEL", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_ALLOWED_USERS", "")
	t.Setenv("GORMES_DISCORD_TOKEN", "")
	t.Setenv("GORMES_DISCORD_CHANNEL_ID", "")
	t.Setenv("GORMES_SLACK_ENABLED", "")
	t.Setenv("GORMES_SLACK_BOT_TOKEN", "")
	t.Setenv("GORMES_SLACK_APP_TOKEN", "")
	t.Setenv("GORMES_SLACK_CHANNEL_ID", "")
	if err := os.WriteFile(filepath.Join(hermesHome, "config.yaml"), []byte(`
model:
  default: yaml-model
  provider: yaml-provider
web:
  backend: parallel
  use_gateway: true
browser:
  cdp_url: ws://yaml-browser
display:
  tool_progress_command: true
  tool_progress: new
  platforms:
    telegram:
      tool_progress: false
  tool_progress_overrides:
    discord: off
security:
  website_blocklist:
    enabled: true
    domains:
      - example.com
    shared_files:
      - community-blocklist.txt
platforms:
  telegram:
    enabled: true
    token: yaml-token
    home_channel:
      chat_id: "123456"
    extra:
      require_mention: true
      bot_username: gormes_bot
  discord:
    enabled: true
    token: discord-token
    home_channel:
      chat_id: D-home
  slack:
    enabled: true
    token: slack-bot-token
    home_channel:
      chat_id: C-home
    extra:
      app_token: slack-app-token
streaming:
  fresh_final_after_seconds: 17.5
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hermes.Model != "hermes-agent" {
		t.Fatalf("Hermes.Model = %q, want built-in default instead of Hermes config.yaml", cfg.Hermes.Model)
	}
	if cfg.Hermes.Provider != "" || cfg.Hermes.Endpoint != "" {
		t.Fatalf("Hermes provider/endpoint = %q/%q, want no Hermes config.yaml import", cfg.Hermes.Provider, cfg.Hermes.Endpoint)
	}
	if cfg.Telegram.BotToken != "" || cfg.Telegram.AllowedChatID != 0 || cfg.Telegram.RequireMention || cfg.Telegram.BotUsername != "" {
		t.Fatalf("Telegram config imported from Hermes YAML: %+v", cfg.Telegram)
	}
	if cfg.Telegram.FreshFinalAfterSeconds != 60 {
		t.Fatalf("Telegram.FreshFinalAfterSeconds = %v, want default 60", cfg.Telegram.FreshFinalAfterSeconds)
	}
	if cfg.Discord.Token != "" || cfg.Discord.AllowedChannelID != "" {
		t.Fatalf("Discord config imported from Hermes YAML: %+v", cfg.Discord)
	}
	if cfg.Slack.Enabled || cfg.Slack.BotToken != "" || cfg.Slack.AppToken != "" || cfg.Slack.AllowedChannelID != "" {
		t.Fatalf("Slack config imported from Hermes YAML: %+v", cfg.Slack)
	}
	if cfg.Display.ToolProgress != "" || cfg.Display.ToolProgressCommand || len(cfg.Display.Platforms) != 0 {
		t.Fatalf("Display config imported from Hermes YAML: %+v", cfg.Display)
	}
	if cfg.Web.Backend != "" || cfg.Web.UseGateway {
		t.Fatalf("Web config imported from Hermes YAML: %+v", cfg.Web)
	}
	if cfg.Browser.CDPURL != "" {
		t.Fatalf("Browser.CDPURL = %q, want no Hermes config.yaml import", cfg.Browser.CDPURL)
	}
	if cfg.Security.WebsiteBlocklist.Enabled || len(cfg.Security.WebsiteBlocklist.Domains) != 0 || len(cfg.Security.WebsiteBlocklist.SharedFiles) != 0 {
		t.Fatalf("Security website blocklist imported from Hermes YAML: %+v", cfg.Security.WebsiteBlocklist)
	}
}

func TestSetGormesDisplayPlatformToolProgressWritesNativePlatformOverride(t *testing.T) {
	cfgHome := t.TempDir()
	hermesHome := t.TempDir()
	gormesHome := filepath.Join(cfgHome, "gormes")
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("HERMES_HOME", hermesHome)
	if err := os.MkdirAll(gormesHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ConfigPath(), []byte(`
[display]
tool_progress = "all"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hermesHome, "config.yaml"), []byte(`
display:
  tool_progress: off
`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := SetGormesDisplayPlatformToolProgress("Telegram", "new"); err != nil {
		t.Fatalf("SetGormesDisplayPlatformToolProgress: %v", err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Display.ToolProgress != "all" {
		t.Fatalf("Display.ToolProgress = %q, want existing native global mode preserved", cfg.Display.ToolProgress)
	}
	if got := cfg.Display.Platforms["telegram"].ToolProgress; got != "new" {
		t.Fatalf("Display.Platforms[telegram].ToolProgress = %q, want persisted native override", got)
	}
	hermesBody, err := os.ReadFile(filepath.Join(hermesHome, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(hermesBody), "Telegram") || strings.Contains(string(hermesBody), "new") {
		t.Fatalf("Hermes config.yaml mutated by native display writer:\n%s", hermesBody)
	}
}

func TestLoad_DiscordDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Discord.CoalesceMs != 1000 {
		t.Errorf("Discord.CoalesceMs default = %d, want 1000", cfg.Discord.CoalesceMs)
	}
	if cfg.Discord.Enabled() {
		t.Error("Discord should be disabled when no token is set")
	}
}

func TestLoad_DiscordFromFile(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	cfgDir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(`
[discord]
token = "bot-abc"
allowed_channel_id = "9999"
coalesce_ms = 500
first_run_discovery = true
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Discord.Token != "bot-abc" {
		t.Errorf("Token = %q", cfg.Discord.Token)
	}
	if cfg.Discord.AllowedChannelID != "9999" {
		t.Errorf("AllowedChannelID = %q", cfg.Discord.AllowedChannelID)
	}
	if cfg.Discord.CoalesceMs != 500 {
		t.Errorf("CoalesceMs = %d", cfg.Discord.CoalesceMs)
	}
	if !cfg.Discord.FirstRunDiscovery {
		t.Error("FirstRunDiscovery = false, want true")
	}
	if !cfg.Discord.Enabled() {
		t.Error("Discord should be enabled with token + channel id")
	}
}

func TestLoad_DiscordAdmissionPolicyConfigFromFile(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	cfgDir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(`
[discord]
token = "bot-abc"
allowed_channels = ["111", "222"]
ignored_channels = "333,444"
free_response_channels = 555
no_thread_channels = ["666"]
require_mention = false
auto_thread = false
reply_to_mode = "all"
allow_bots = "mentions"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.Discord.AllowedChannelIDs(), []string{"111", "222"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("AllowedChannelIDs = %v, want %v", got, want)
	}
	if got, want := cfg.Discord.IgnoredChannelIDs(), []string{"333", "444"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("IgnoredChannelIDs = %v, want %v", got, want)
	}
	if got, want := cfg.Discord.FreeResponseChannelIDs(), []string{"555"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("FreeResponseChannelIDs = %v, want %v", got, want)
	}
	if got, want := cfg.Discord.NoThreadChannelIDs(), []string{"666"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("NoThreadChannelIDs = %v, want %v", got, want)
	}
	if cfg.Discord.RequireMentionValue(true) {
		t.Fatal("RequireMentionValue = true, want false from config")
	}
	if cfg.Discord.AutoThreadValue(true) {
		t.Fatal("AutoThreadValue = true, want false from config")
	}
	if cfg.Discord.ReplyToModeValue() != "all" {
		t.Fatalf("ReplyToModeValue = %q, want all", cfg.Discord.ReplyToModeValue())
	}
	if cfg.Discord.AllowBotsValue() != "mentions" {
		t.Fatalf("AllowBotsValue = %q, want mentions", cfg.Discord.AllowBotsValue())
	}
	if !cfg.Discord.Enabled() {
		t.Fatal("Discord Enabled = false, want token plus allowed_channels to enable")
	}
}

func TestLoad_DiscordAdmissionPolicyEnvOverrides(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	t.Setenv("DISCORD_ALLOWED_CHANNELS", "env-1, env-2")
	t.Setenv("DISCORD_IGNORED_CHANNELS", "*")
	t.Setenv("DISCORD_FREE_RESPONSE_CHANNELS", "free-env")
	t.Setenv("DISCORD_NO_THREAD_CHANNELS", "direct-env")
	t.Setenv("DISCORD_REQUIRE_MENTION", "false")
	t.Setenv("DISCORD_AUTO_THREAD", "false")
	t.Setenv("DISCORD_REPLY_TO_MODE", "off")
	t.Setenv("DISCORD_ALLOW_BOTS", "all")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.Discord.AllowedChannelIDs(), []string{"env-1", "env-2"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("AllowedChannelIDs = %v, want %v", got, want)
	}
	if got := cfg.Discord.IgnoredChannelIDs(); len(got) != 1 || got[0] != "*" {
		t.Fatalf("IgnoredChannelIDs = %v, want [*]", got)
	}
	if cfg.Discord.RequireMentionValue(true) {
		t.Fatal("RequireMentionValue = true, want false from env")
	}
	if cfg.Discord.AutoThreadValue(true) {
		t.Fatal("AutoThreadValue = true, want false from env")
	}
	if cfg.Discord.ReplyToModeValue() != "off" {
		t.Fatalf("ReplyToModeValue = %q, want off", cfg.Discord.ReplyToModeValue())
	}
	if cfg.Discord.AllowBotsValue() != "all" {
		t.Fatalf("AllowBotsValue = %q, want all", cfg.Discord.AllowBotsValue())
	}
}

func TestLoad_SlackFromFile(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	cfgDir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(`
[slack]
enabled = true
bot_token = "xoxb-file"
app_token = "xapp-file"
allowed_channel_id = "C123"
coalesce_ms = 750
first_run_discovery = true
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Slack.Enabled {
		t.Error("Slack.Enabled = false, want true from config.toml")
	}
	if cfg.Slack.BotToken != "xoxb-file" {
		t.Errorf("Slack.BotToken = %q", cfg.Slack.BotToken)
	}
	if cfg.Slack.AppToken != "xapp-file" {
		t.Errorf("Slack.AppToken = %q", cfg.Slack.AppToken)
	}
	if cfg.Slack.AllowedChannelID != "C123" {
		t.Errorf("Slack.AllowedChannelID = %q", cfg.Slack.AllowedChannelID)
	}
	if cfg.Slack.CoalesceMs != 750 {
		t.Errorf("Slack.CoalesceMs = %d", cfg.Slack.CoalesceMs)
	}
	if !cfg.Slack.FirstRunDiscovery {
		t.Error("Slack.FirstRunDiscovery = false, want true")
	}
}

func TestLoad_SlackMentionPolicyConfigFromFile(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	cfgDir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(`
[slack]
enabled = true
bot_token = "xoxb-file"
app_token = "xapp-file"
allowed_channel_id = "C123"
require_mention = false
strict_mention = true
reply_in_thread = false
free_response_channels = ["C-free", "C-other"]
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := cfg.Slack.RequireMention.(bool); !ok || got {
		t.Fatalf("Slack.RequireMention = %#v, want false bool", cfg.Slack.RequireMention)
	}
	if got, ok := cfg.Slack.StrictMention.(bool); !ok || !got {
		t.Fatalf("Slack.StrictMention = %#v, want true bool", cfg.Slack.StrictMention)
	}
	if cfg.Slack.ReplyInThread {
		t.Fatal("Slack.ReplyInThread = true, want false from config")
	}
	gotChannels, ok := cfg.Slack.FreeResponseChannels.([]any)
	if !ok || len(gotChannels) != 2 || gotChannels[0] != "C-free" || gotChannels[1] != "C-other" {
		t.Fatalf("Slack.FreeResponseChannels = %#v, want C-free/C-other", cfg.Slack.FreeResponseChannels)
	}
}

func TestLoad_SlackEnvOverridesFile(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	cfgDir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(`
[slack]
enabled = false
bot_token = "xoxb-file"
app_token = "xapp-file"
allowed_channel_id = "C-file"
coalesce_ms = 1000
first_run_discovery = false
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GORMES_SLACK_ENABLED", "true")
	t.Setenv("GORMES_SLACK_BOT_TOKEN", "xoxb-env")
	t.Setenv("GORMES_SLACK_APP_TOKEN", "xapp-env")
	t.Setenv("GORMES_SLACK_CHANNEL_ID", "C-env")
	t.Setenv("GORMES_SLACK_COALESCE_MS", "250")
	t.Setenv("GORMES_SLACK_FIRST_RUN_DISCOVERY", "true")
	t.Setenv("GORMES_SLACK_REQUIRE_MENTION", "false")
	t.Setenv("GORMES_SLACK_STRICT_MENTION", "true")
	t.Setenv("GORMES_SLACK_FREE_RESPONSE_CHANNELS", "C-env-free,C-env-other")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Slack.Enabled {
		t.Error("Slack.Enabled = false, want true from env")
	}
	if cfg.Slack.BotToken != "xoxb-env" {
		t.Errorf("Slack.BotToken = %q", cfg.Slack.BotToken)
	}
	if cfg.Slack.AppToken != "xapp-env" {
		t.Errorf("Slack.AppToken = %q", cfg.Slack.AppToken)
	}
	if cfg.Slack.AllowedChannelID != "C-env" {
		t.Errorf("Slack.AllowedChannelID = %q", cfg.Slack.AllowedChannelID)
	}
	if cfg.Slack.CoalesceMs != 250 {
		t.Errorf("Slack.CoalesceMs = %d", cfg.Slack.CoalesceMs)
	}
	if !cfg.Slack.FirstRunDiscovery {
		t.Error("Slack.FirstRunDiscovery = false, want true from env")
	}
	if cfg.Slack.RequireMention != "false" {
		t.Fatalf("Slack.RequireMention = %#v, want env string false", cfg.Slack.RequireMention)
	}
	if cfg.Slack.StrictMention != "true" {
		t.Fatalf("Slack.StrictMention = %#v, want env string true", cfg.Slack.StrictMention)
	}
	if cfg.Slack.FreeResponseChannels != "C-env-free,C-env-other" {
		t.Fatalf("Slack.FreeResponseChannels = %#v, want env CSV", cfg.Slack.FreeResponseChannels)
	}
}

func TestLoad_ResumeFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load([]string{"--resume", "sess-abc123"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Resume != "sess-abc123" {
		t.Errorf("Resume = %q, want %q", cfg.Resume, "sess-abc123")
	}
}

func TestLoad_ResumeFlagEmptyDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil) // nil means skip flag parsing — existing contract
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Resume != "" {
		t.Errorf("Resume (no flags) = %q, want \"\"", cfg.Resume)
	}
}

func TestLoad_DelegationMaxWaitingFromConfig(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[delegation]
max_waiting = 7
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Delegation.MaxWaiting != 7 {
		t.Fatalf("Delegation.MaxWaiting = %d, want 7", cfg.Delegation.MaxWaiting)
	}
}

func TestLoad_DelegationMaxWaitingRejectsNegative(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[delegation]
max_waiting = -1
`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(nil)
	if err == nil || !strings.Contains(err.Error(), "delegation.max_waiting") {
		t.Fatalf("Load err = %v, want delegation.max_waiting validation error", err)
	}
}

func TestSessionDBPath_HonorsXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/gormes-test-xdg")
	t.Setenv("GORMES_HOME", "/tmp/gormes-test-home")
	got := SessionDBPath()
	want := "/tmp/gormes-test-home/sessions.db"
	if got != want {
		t.Errorf("SessionDBPath() = %q, want %q", got, want)
	}
}

func TestSessionDBPath_DefaultsToHomeDotGormes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("GORMES_HOME", "")
	t.Setenv("HOME", home)
	got := SessionDBPath()
	want := filepath.Join(home, ".gormes", "sessions.db")
	if got != want {
		t.Errorf("SessionDBPath() with empty GORMES_HOME = %q, want %q", got, want)
	}
}

func TestLoad_MemoryQueueCapDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Telegram.MemoryQueueCap != 1024 {
		t.Errorf("MemoryQueueCap default = %d, want 1024", cfg.Telegram.MemoryQueueCap)
	}
}

func TestMemoryDBPath_HonorsXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/gormes-test-memxdg")
	t.Setenv("GORMES_HOME", "/tmp/gormes-test-home")
	got := MemoryDBPath()
	want := "/tmp/gormes-test-home/memory.db"
	if got != want {
		t.Errorf("MemoryDBPath() = %q, want %q", got, want)
	}
}

func TestMemoryDBPath_DefaultsToHomeDotGormes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("GORMES_HOME", "")
	t.Setenv("HOME", home)
	got := MemoryDBPath()
	want := filepath.Join(home, ".gormes", "memory.db")
	if got != want {
		t.Errorf("MemoryDBPath() default = %q, want %q", got, want)
	}
}

func TestSkillsRoot_DefaultsToHomeDotGormes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("GORMES_HOME", "")
	t.Setenv("HOME", home)
	var cfg Config
	got := cfg.SkillsRoot()
	want := filepath.Join(home, ".gormes", "skills")
	if got != want {
		t.Errorf("SkillsRoot() default = %q, want %q", got, want)
	}
}

func TestSkillsRoot_HonorsXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/gormes-test-skills")
	t.Setenv("GORMES_HOME", "/tmp/gormes-test-home")
	var cfg Config
	got := cfg.SkillsRoot()
	want := "/tmp/gormes-test-home/skills"
	if got != want {
		t.Errorf("SkillsRoot() = %q, want %q", got, want)
	}
}

func TestHooksRoot_DefaultsToHomeDotGormes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("GORMES_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	got := HooksRoot()
	want := filepath.Join(home, ".gormes", "hooks")
	if got != want {
		t.Errorf("HooksRoot() default = %q, want %q", got, want)
	}
}

func TestHooksRoot_HonorsXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/gormes-test-hooks")
	t.Setenv("GORMES_HOME", "/tmp/gormes-test-home")
	got := HooksRoot()
	want := "/tmp/gormes-test-home/hooks"
	if got != want {
		t.Errorf("HooksRoot() = %q, want %q", got, want)
	}
}

func TestGatewayRuntimeStatusPath_HonorsXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/gormes-test-status")
	t.Setenv("GORMES_HOME", "/tmp/gormes-test-home")
	got := GatewayRuntimeStatusPath()
	want := "/tmp/gormes-test-home/gateway_state.json"
	if got != want {
		t.Errorf("GatewayRuntimeStatusPath() = %q, want %q", got, want)
	}
}

func TestGatewayLockDir_HonorsXDGStateHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/gormes-test-state")
	t.Setenv("GORMES_HOME", "/tmp/gormes-test-home")
	got := GatewayLockDir()
	want := "/tmp/gormes-test-home/gateway-locks"
	if got != want {
		t.Errorf("GatewayLockDir() = %q, want %q", got, want)
	}
}

func TestGatewayLockDir_DefaultsToHomeDotGormes(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("GORMES_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	got := GatewayLockDir()
	want := filepath.Join(home, ".gormes", "gateway-locks")
	if got != want {
		t.Errorf("GatewayLockDir() default = %q, want %q", got, want)
	}
}

func TestBootPath_DefaultsToHomeDotGormes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("GORMES_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	got := BootPath()
	want := filepath.Join(home, ".gormes", "BOOT.md")
	if got != want {
		t.Errorf("BootPath() default = %q, want %q", got, want)
	}
}

func TestBootPath_HonorsXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/gormes-test-boot")
	t.Setenv("GORMES_HOME", "/tmp/gormes-test-home")
	got := BootPath()
	want := "/tmp/gormes-test-home/BOOT.md"
	if got != want {
		t.Errorf("BootPath() = %q, want %q", got, want)
	}
}

func TestLoad_SkillsRootEnvOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GORMES_SKILLS_ROOT", "/tmp/custom-skills")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.SkillsRoot(); got != "/tmp/custom-skills" {
		t.Fatalf("SkillsRoot() = %q, want %q", got, "/tmp/custom-skills")
	}
}

func TestLoad_ExtractorDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Telegram.ExtractorBatchSize != 5 {
		t.Errorf("ExtractorBatchSize default = %d, want 5", cfg.Telegram.ExtractorBatchSize)
	}
	if cfg.Telegram.ExtractorPollInterval != 10*time.Second {
		t.Errorf("ExtractorPollInterval default = %v, want 10s", cfg.Telegram.ExtractorPollInterval)
	}
}

func TestLoad_RecallDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Telegram.RecallEnabled {
		t.Errorf("RecallEnabled default = false, want true")
	}
	if cfg.Telegram.RecallWeightThreshold != 1.0 {
		t.Errorf("RecallWeightThreshold = %v, want 1.0", cfg.Telegram.RecallWeightThreshold)
	}
	if cfg.Telegram.RecallMaxFacts != 10 {
		t.Errorf("RecallMaxFacts = %d, want 10", cfg.Telegram.RecallMaxFacts)
	}
	if cfg.Telegram.RecallDepth != 2 {
		t.Errorf("RecallDepth = %d, want 2", cfg.Telegram.RecallDepth)
	}
}

func TestLoad_SemanticDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}

	// Semantic is opt-in: everything off by default.
	if cfg.Telegram.SemanticEnabled {
		t.Errorf("SemanticEnabled default = true, want false (opt-in)")
	}
	if cfg.Telegram.SemanticModel != "" {
		t.Errorf("SemanticModel default = %q, want empty", cfg.Telegram.SemanticModel)
	}
	// But tunables have usable defaults so a single `semantic_enabled = true`
	// + `semantic_model = "..."` in TOML is enough to light things up.
	if cfg.Telegram.SemanticTopK != 3 {
		t.Errorf("SemanticTopK default = %d, want 3", cfg.Telegram.SemanticTopK)
	}
	if cfg.Telegram.SemanticMinSimilarity != 0.35 {
		t.Errorf("SemanticMinSimilarity default = %v, want 0.35", cfg.Telegram.SemanticMinSimilarity)
	}
	if cfg.Telegram.EmbedderPollInterval != 30*time.Second {
		t.Errorf("EmbedderPollInterval default = %v, want 30s", cfg.Telegram.EmbedderPollInterval)
	}
	if cfg.Telegram.EmbedderBatchSize != 10 {
		t.Errorf("EmbedderBatchSize default = %d, want 10", cfg.Telegram.EmbedderBatchSize)
	}
	if cfg.Telegram.EmbedderCallTimeout != 10*time.Second {
		t.Errorf("EmbedderCallTimeout default = %v, want 10s", cfg.Telegram.EmbedderCallTimeout)
	}
	if cfg.Telegram.QueryEmbedTimeout != 60*time.Millisecond {
		t.Errorf("QueryEmbedTimeout default = %v, want 60ms", cfg.Telegram.QueryEmbedTimeout)
	}
}

func TestLoad_ConfigVersionDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConfigVersion != CurrentConfigVersion {
		t.Errorf("ConfigVersion = %d, want %d (defaults())", cfg.ConfigVersion, CurrentConfigVersion)
	}
}

func TestLoad_ConfigVersionMissingInFileTreatedAsV1(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))

	cfgDir := filepath.Join(cfgHome, "gormes")
	_ = os.MkdirAll(cfgDir, 0o755)
	// TOML file with no _config_version key.
	tomlBody := "[hermes]\nendpoint = \"http://1.2.3.4:5678\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlBody), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Missing version should auto-promote to CurrentConfigVersion after
	// migrations run; endpoint override from file must be preserved.
	if cfg.ConfigVersion != CurrentConfigVersion {
		t.Errorf("ConfigVersion = %d, want %d (post-migration)", cfg.ConfigVersion, CurrentConfigVersion)
	}
	if cfg.Hermes.Endpoint != "http://1.2.3.4:5678" {
		t.Errorf("Endpoint = %q, want the file value", cfg.Hermes.Endpoint)
	}
}

func TestDelegationCfgDecode(t *testing.T) {
	const tomlText = `
[delegation]
enabled                 = true
max_depth               = 2
max_concurrent_children = 3
default_max_iterations  = 50
	default_timeout         = "1h"
run_log_path            = "/tmp/subagents/runs.jsonl"
`
	var cfg Config
	if err := toml.NewDecoder(strings.NewReader(tomlText)).EnableUnmarshalerInterface().Decode(&cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !cfg.Delegation.Enabled {
		t.Errorf("Enabled: want true, got false")
	}
	if cfg.Delegation.MaxDepth != 2 {
		t.Errorf("MaxDepth: want 2, got %d", cfg.Delegation.MaxDepth)
	}
	if cfg.Delegation.MaxConcurrentChildren != 3 {
		t.Errorf("MaxConcurrentChildren: want 3, got %d", cfg.Delegation.MaxConcurrentChildren)
	}
	if cfg.Delegation.DefaultMaxIterations != 50 {
		t.Errorf("DefaultMaxIterations: want 50, got %d", cfg.Delegation.DefaultMaxIterations)
	}
	if cfg.Delegation.DefaultTimeout != time.Hour {
		t.Errorf("DefaultTimeout: want 1h, got %v", cfg.Delegation.DefaultTimeout)
	}
	if cfg.Delegation.RunLogPath != "/tmp/subagents/runs.jsonl" {
		t.Errorf("RunLogPath: want %q, got %q", "/tmp/subagents/runs.jsonl", cfg.Delegation.RunLogPath)
	}
}

func TestDelegationCfgResolvedRunLogPathHonorsOverride(t *testing.T) {
	cfg := DelegationCfg{RunLogPath: "/tmp/custom-runs.jsonl"}
	if got := cfg.ResolvedRunLogPath(); got != "/tmp/custom-runs.jsonl" {
		t.Errorf("ResolvedRunLogPath() = %q, want %q", got, "/tmp/custom-runs.jsonl")
	}
}

func TestDelegationCfgResolvedRunLogPathDefaultsToGormesHome(t *testing.T) {
	t.Setenv("GORMES_HOME", "/tmp/gormes-home")

	var cfg DelegationCfg
	want := "/tmp/gormes-home/subagents/runs.jsonl"
	if got := cfg.ResolvedRunLogPath(); got != want {
		t.Errorf("ResolvedRunLogPath() = %q, want %q", got, want)
	}
}

func TestLoad_ConfigVersionFromFutureBinaryErrors(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))

	cfgDir := filepath.Join(cfgHome, "gormes")
	_ = os.MkdirAll(cfgDir, 0o755)
	tomlBody := "_config_version = 9999\n[hermes]\nendpoint = \"http://1.2.3.4:5678\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlBody), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(nil)
	if err == nil {
		t.Fatal("expected error for config from future binary")
	}
	if !strings.Contains(err.Error(), "_config_version=9999") {
		t.Errorf("err = %v, want mention of version 9999", err)
	}
}

func TestLoad_RecallDecayHorizonDays(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Telegram.RecallDecayHorizonDays != 180 {
		t.Errorf("RecallDecayHorizonDays default = %d, want 180",
			cfg.Telegram.RecallDecayHorizonDays)
	}
}

// TestLoad_UpdatesDefaults proves the built-in defaults for the
// `[updates]` table: pre_update_backup is false (opt-in) and backup_keep
// is 5 (matches Hermes' default retention budget). Operators who never
// touch the table should get the silent-default behavior the lifecycle
// already documents.
func TestLoad_UpdatesDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Updates.PreUpdateBackup {
		t.Errorf("Updates.PreUpdateBackup default = true, want false (opt-in)")
	}
	if cfg.Updates.BackupKeep != 5 {
		t.Errorf("Updates.BackupKeep default = %d, want 5", cfg.Updates.BackupKeep)
	}
}

// TestLoad_UpdatesFromFile proves the `[updates]` TOML table populates
// the typed Config.Updates field. This is the policy seam consumed by
// `gormes update` to decide whether to take a pre-update backup and how
// many older zips to keep.
func TestLoad_UpdatesFromFile(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[updates]
pre_update_backup = true
backup_keep = 3
`), 0o644)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Updates.PreUpdateBackup {
		t.Errorf("Updates.PreUpdateBackup = false, want true from config.toml")
	}
	if cfg.Updates.BackupKeep != 3 {
		t.Errorf("Updates.BackupKeep = %d, want 3 from config.toml", cfg.Updates.BackupKeep)
	}
}

func TestLoad_CronDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Cron.Enabled {
		t.Errorf("Cron.Enabled default = true, want false (opt-in)")
	}
	if cfg.Cron.CallTimeout != 60*time.Second {
		t.Errorf("Cron.CallTimeout default = %v, want 60s", cfg.Cron.CallTimeout)
	}
	if cfg.Cron.MirrorInterval != 30*time.Second {
		t.Errorf("Cron.MirrorInterval default = %v, want 30s", cfg.Cron.MirrorInterval)
	}
	if cfg.Cron.MirrorPath != "" {
		t.Errorf("Cron.MirrorPath default = %q, want empty (caller resolves XDG)", cfg.Cron.MirrorPath)
	}
}
