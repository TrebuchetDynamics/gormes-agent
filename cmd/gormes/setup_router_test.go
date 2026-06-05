package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func TestSetupRouterNonInteractiveWritesRouterWithoutOverwritingProviderAndRedacts(t *testing.T) {
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

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "router", "--non-interactive")
	if err != nil {
		t.Fatalf("setup router: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Gormes Setup — Router",
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
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{"router-secret-must-not-leak", "free unlimited", "starts Ollama", "gormes-router upstream provider"} {
		if strings.Contains(stdout+stderr, forbidden) {
			t.Fatalf("setup router leaked/printed forbidden %q\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
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

func TestSetupRouterNonInteractiveRejectsSelfRoutingRoute(t *testing.T) {
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

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "router", "--non-interactive")
	if err == nil {
		t.Fatalf("setup router recursion error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "router_recursion_detected") {
		t.Fatalf("err = %v, want router_recursion_detected\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr+err.Error(), "router-secret-must-not-leak") {
		t.Fatalf("recursion diagnostic leaked API key\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
	body, readErr := os.ReadFile(config.ConfigPath())
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if strings.Contains(string(body), "[router]") || strings.Contains(string(body), "self-chat") {
		t.Fatalf("recursion failure wrote router config:\n%s", body)
	}
}

func TestSetupProviderPickerExcludesRouterAsUpstreamProvider(t *testing.T) {
	setupRouterTestHome(t, `
[hermes]
provider = "openrouter"
endpoint = "https://openrouter.ai/api/v1"
model = "moonshotai/kimi-k2.6"
`)
	fake := &setupCommandFakeSeams{isTTY: true}
	fake.chooseSetupProvider = func(_ *cobra.Command, entries []cli.ProviderMenuEntry, _ int) (int, error) {
		leaveUnchanged := -1
		for i, entry := range entries {
			id := strings.ToLower(strings.TrimSpace(entry.ID))
			label := strings.ToLower(strings.TrimSpace(entry.Label))
			if id == cli.ProviderCatalogLeaveUnchanged {
				leaveUnchanged = i
			}
			if id == "router" || id == "gormes-router" || label == "gormes router" {
				t.Fatalf("provider picker exposed Router as upstream provider: %+v", entry)
			}
		}
		if leaveUnchanged < 0 {
			t.Fatal("provider picker missing leave-unchanged entry")
		}
		return leaveUnchanged, nil
	}

	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "provider")
	if err != nil {
		t.Fatalf("setup provider: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(strings.ToLower(stdout), "gormes router") {
		t.Fatalf("setup provider output exposed router as provider:\n%s", stdout)
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
