package setuprouter

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestFallbackRulesFromPrimaryChat(t *testing.T) {
	rules := FallbackRules([]config.RouterRouteCfg{
		{Alias: "primary-chat"},
		{Alias: "fallback-a"},
		{Alias: "fallback-b"},
	})
	if len(rules) != 2 || rules[0].From != "primary-chat" || rules[0].To != "fallback-a" || rules[1].To != "fallback-b" {
		t.Fatalf("rules = %+v", rules)
	}
}

func TestOpenAIBaseURLNormalizesListenAddress(t *testing.T) {
	for _, tt := range []struct {
		listen string
		want   string
	}{
		{listen: "127.0.0.1:8787", want: "http://127.0.0.1:8787/v1"},
		{listen: "https://example.test/router?x=1#frag", want: "https://example.test/router/v1"},
		{listen: "", want: "http://127.0.0.1:9999/v1"},
	} {
		if got := OpenAIBaseURL(tt.listen, "127.0.0.1:9999"); got != tt.want {
			t.Fatalf("OpenAIBaseURL(%q) = %q, want %q", tt.listen, got, tt.want)
		}
	}
}

func TestSlugSanitizesRouteAlias(t *testing.T) {
	if got := Slug(" Open_AI Free/Tier "); got != "open-ai-free-tier" {
		t.Fatalf("Slug = %q", got)
	}
	if got := Slug("!!!"); got != "route" {
		t.Fatalf("empty Slug = %q", got)
	}
}

func TestRouteLabels(t *testing.T) {
	got := RouteLabels(config.RouterRouteCfg{Name: "free-tier", Optional: true})
	want := []string{"requires your provider account/API key; quotas are provider-controlled", "optional; only enabled if already installed and healthy"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RouteLabels = %#v, want %#v", got, want)
	}
}

func TestRunWritesRouterWithoutOverwritingProviderAndRedacts(t *testing.T) {
	setupRouterTestHome(t, `
[hermes]
provider = "deepseek"
endpoint = "https://api.deepseek.com/v1"
model = "deepseek-chat"

[[fallback_providers]]
provider = "openrouter"
model = "moonshotai/kimi-k2:free"
`)
	t.Setenv("GORMES_ROUTER_LISTEN", "127.0.0.1:9888")
	t.Setenv("GORMES_ROUTER_API_KEY", "router-secret-must-not-leak")
	t.Setenv("GORMES_ROUTER_ROUTE_PROVIDER", "custom")
	t.Setenv("GORMES_ROUTER_ROUTE_MODEL", "env-model")
	t.Setenv("GORMES_ROUTER_ROUTE_ALIAS", "env-chat")
	t.Setenv("GORMES_ROUTER_ROUTE_BASE_URL", "https://llm.example/v1")
	t.Setenv("GORMES_ROUTER_ROUTE_API_KEY_ENV", "ROUTER_ENV_ROUTE_KEY")
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("GORMES_API_KEY", "")

	var stdout bytes.Buffer
	if err := Run(&stdout, Options{}); err != nil {
		t.Fatalf("setup router: %v\nstdout=%s", err, stdout.String())
	}
	for _, want := range []string{
		"Gormes Router configured.",
		"local client endpoint, not an upstream provider",
		"OpenAI base URL: http://127.0.0.1:9888/v1",
		"API key: referenced via GORMES_ROUTER_API_KEY (redacted)",
		"primary-chat -> deepseek/deepseek-chat",
		"fallback-openrouter-1 -> openrouter/moonshotai/kimi-k2:free",
		"env-chat -> custom/env-model",
		"requires your provider account/API key; quotas are provider-controlled",
		"provider picker remains: gormes setup provider",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"router-secret-must-not-leak", "free unlimited", "starts Ollama", "gormes-router upstream provider"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("setup router leaked/printed forbidden %q\nstdout=%s", forbidden, stdout.String())
		}
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Hermes.Provider != "deepseek" || cfg.Hermes.Model != "deepseek-chat" || cfg.Hermes.Endpoint != "https://api.deepseek.com/v1" {
		t.Fatalf("hermes provider was overwritten: %+v", cfg.Hermes)
	}
	if !cfg.Router.Enabled || cfg.Router.Listen != "127.0.0.1:9888" || cfg.Router.APIKeyEnv != "GORMES_ROUTER_API_KEY" || !cfg.Router.RedactLogs || cfg.Router.SetupMode != "local_gateway" {
		t.Fatalf("router config = %+v", cfg.Router)
	}
	if len(cfg.Router.Routes) != 3 {
		t.Fatalf("router routes = %+v, want primary + fallback + env route", cfg.Router.Routes)
	}
	if cfg.Router.Routes[0].Alias != "primary-chat" || cfg.Router.Routes[1].Alias != "fallback-openrouter-1" || cfg.Router.Routes[2].Alias != "env-chat" {
		t.Fatalf("router route aliases = %+v", cfg.Router.Routes)
	}
	if cfg.Router.Routes[2].APIKeyEnv != "ROUTER_ENV_ROUTE_KEY" || cfg.Router.Routes[2].BaseURL != "https://llm.example/v1" {
		t.Fatalf("env route = %+v", cfg.Router.Routes[2])
	}
	if len(cfg.Router.Fallback) != 2 || cfg.Router.Fallback[0].From != "primary-chat" || cfg.Router.Fallback[0].To != "fallback-openrouter-1" || cfg.Router.Fallback[1].To != "env-chat" {
		t.Fatalf("router fallback = %+v", cfg.Router.Fallback)
	}
	body, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(body), "router-secret-must-not-leak") {
		t.Fatalf("config leaked router API key:\n%s", body)
	}
}

func TestRunRejectsSelfRoutingRouteBeforeWritingConfig(t *testing.T) {
	setupRouterTestHome(t, `
[hermes]
provider = "deepseek"
endpoint = "https://api.deepseek.com/v1"
model = "deepseek-chat"
`)
	t.Setenv("GORMES_ROUTER_LISTEN", "127.0.0.1:9899")
	t.Setenv("GORMES_ROUTER_API_KEY", "router-secret-must-not-leak")
	t.Setenv("GORMES_ROUTER_ROUTE_PROVIDER", "custom")
	t.Setenv("GORMES_ROUTER_ROUTE_MODEL", "self-model")
	t.Setenv("GORMES_ROUTER_ROUTE_ALIAS", "self-chat")
	t.Setenv("GORMES_ROUTER_ROUTE_BASE_URL", "http://127.0.0.1:9899/v1")
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("GORMES_API_KEY", "")

	var stdout bytes.Buffer
	err := Run(&stdout, Options{})
	if err == nil {
		t.Fatalf("setup router recursion error = nil\nstdout=%s", stdout.String())
	}
	if !strings.Contains(err.Error(), "router_recursion_detected") {
		t.Fatalf("err = %v, want router_recursion_detected\nstdout=%s", err, stdout.String())
	}
	if strings.Contains(stdout.String()+err.Error(), "router-secret-must-not-leak") {
		t.Fatalf("recursion diagnostic leaked API key\nstdout=%s\nerr=%v", stdout.String(), err)
	}
	body, readErr := os.ReadFile(config.ConfigPath())
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if strings.Contains(string(body), "[router]") || strings.Contains(string(body), "self-chat") {
		t.Fatalf("recursion failure wrote router config:\n%s", body)
	}
}

func setupRouterTestHome(t *testing.T, body string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg-data"))
	for _, key := range []string{
		"GORMES_ROUTER_LISTEN",
		"GORMES_ROUTER_API_KEY",
		"GORMES_ROUTER_API_KEY_ENV",
		"GORMES_ROUTER_ROUTE_PROVIDER",
		"GORMES_ROUTER_ROUTE_MODEL",
		"GORMES_ROUTER_ROUTE_ALIAS",
		"GORMES_ROUTER_ROUTE_BASE_URL",
		"GORMES_ROUTER_ROUTE_API_KEY_ENV",
		"GORMES_ROUTER_ROUTE_OPTIONAL",
	} {
		t.Setenv(key, "")
	}
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(strings.TrimSpace(body)+"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return home
}
