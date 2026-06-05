package router

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestRouterConfigReadModelParsesRoutesAndFallback(t *testing.T) {
	cfg := loadRouterFixture(t, `
[hermes]
provider = "deepseek"
model = "deepseek-chat"
endpoint = "https://api.deepseek.com/v1"
api_key = "primary-secret-fixture"

[router]
enabled = true
listen = "127.0.0.1:8787"
api_keys = ["dev-local-secret"]
api_key_env = "GORMES_ROUTER_LOCAL_KEY"
redact_logs = true
setup_mode = "local_gateway"

[[router.routes]]
name = "budget-openrouter"
provider = "openrouter"
model = "moonshotai/kimi-k2:free"
alias = "budget-chat"
base_url = "https://openrouter.ai/api/v1"
api_key_env = "ROUTER_OPENROUTER_KEY"
transport = "chat_completions"
optional = true

[[router.routes]]
name = "local-openai-compatible"
provider = "custom"
model = "llama3.2"
alias = "local-chat"
base_url = "http://127.0.0.1:11434/v1"
transport = "chat_completions"
optional = true

[[router.routes]]
name = "custom-compatible"
provider = "custom"
model = "my-model"
alias = "custom-chat"
base_url = "https://llm.example/v1"
api_key_env = "ROUTER_CUSTOM_KEY"
transport = "chat_completions"

[[router.fallback]]
from = "primary-chat"
to = "budget-chat"
on = ["rate_limit", "auth", "server_error", "policy", "timeout", "malformed_request", "HTTP_502"]
`)

	model := BuildReadModel(cfg, Options{LookupEnv: mapLookup(map[string]string{
		"GORMES_ROUTER_LOCAL_KEY": "router-secret",
		"ROUTER_OPENROUTER_KEY":   "budget-secret-fixture",
		"ROUTER_CUSTOM_KEY":       "custom-secret-fixture",
	})})

	if !model.Enabled {
		t.Fatalf("Router Enabled = false")
	}
	if model.Listen != "127.0.0.1:8787" {
		t.Fatalf("Listen = %q", model.Listen)
	}
	if !model.RedactLogs || model.SetupMode != "local_gateway" {
		t.Fatalf("router defaults/setup = redact:%v mode:%q", model.RedactLogs, model.SetupMode)
	}
	if !model.Auth.Configured || model.Auth.InlineKeyCount != 1 || model.Auth.APIKeyEnv != "GORMES_ROUTER_LOCAL_KEY" {
		t.Fatalf("Auth = %+v, want configured redacted env + one inline key count", model.Auth)
	}
	if got, want := len(model.Routes), 4; got != want {
		t.Fatalf("routes = %d, want %d: %+v", got, want, model.Routes)
	}

	fallback := model.Fallback[0]
	wantOn := []string{"rate_limit", "server_error", "timeout", "server_error"}
	if strings.Join(fallback.On, ",") != strings.Join(wantOn, ",") {
		t.Fatalf("fallback.On = %#v, want %#v", fallback.On, wantOn)
	}
	for _, rejected := range []string{"auth", "policy", "malformed_request"} {
		if !contains(fallback.Rejected, rejected) {
			t.Fatalf("fallback rejected classes = %#v, want %q rejected", fallback.Rejected, rejected)
		}
	}
}

func TestRouterRegistryListsPrimaryFreeTierLocalAndCustomAliases(t *testing.T) {
	cfg := routerFixtureConfig()
	model := BuildReadModel(cfg, Options{LookupEnv: mapLookup(map[string]string{
		"ROUTER_OPENROUTER_KEY": "budget-secret-fixture",
		"ROUTER_CUSTOM_KEY":     "custom-secret-fixture",
	})})

	registry := NewRegistry(model)
	models := registry.Models()
	assertModelRoute(t, models, "primary-chat", "deepseek", "deepseek-chat", RouteStatusConfigured, "")
	assertModelRoute(t, models, "budget-chat", "openrouter", "moonshotai/kimi-k2:free", RouteStatusConfigured, QuotaProviderControlled)
	assertModelRoute(t, models, "local-chat", "custom", "llama3.2", RouteStatusUnavailable, "")
	assertModelRoute(t, models, "custom-chat", "custom", "my-model", RouteStatusConfigured, "")

	resolved := registry.Resolve("budget-chat")
	if len(resolved) != 1 || resolved[0].Provider != "openrouter" {
		t.Fatalf("Resolve(budget-chat) = %+v", resolved)
	}
}

func TestRouterStatusRedactsSecretsAndDistinguishesStates(t *testing.T) {
	disabled := BuildReadModel(config.Config{}, Options{})
	if disabled.Status.State != RouterStatusDisabled {
		t.Fatalf("disabled status = %q", disabled.Status.State)
	}

	cfg := routerFixtureConfig()
	model := BuildReadModel(cfg, Options{LookupEnv: mapLookup(map[string]string{
		"ROUTER_OPENROUTER_KEY": "budget-secret-fixture",
		"ROUTER_CUSTOM_KEY":     "custom-secret-fixture",
	})})
	if model.Status.State != RouterStatusConfigured {
		t.Fatalf("router status = %q, want %q", model.Status.State, RouterStatusConfigured)
	}
	assertRouteStatus(t, model.Routes, "local-chat", RouteStatusUnavailable)

	missing := cfg
	missing.Router.Routes = []config.RouterRouteCfg{{
		Name:     "missing-cloud",
		Provider: "custom",
		Model:    "my-model",
		Alias:    "missing-chat",
		BaseURL:  "https://llm.example/v1",
	}}
	missingModel := BuildReadModel(missing, Options{SkipPrimary: true})
	assertRouteStatus(t, missingModel.Routes, "missing-chat", RouteStatusMissingCredential)

	invalid := cfg
	invalid.Router.Routes = []config.RouterRouteCfg{{Name: "bad-route", Alias: "bad-chat", Provider: "custom"}}
	invalidModel := BuildReadModel(invalid, Options{SkipPrimary: true})
	assertRouteStatus(t, invalidModel.Routes, "bad-chat", RouteStatusInvalidRoute)

	blob, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	text := string(blob)
	for _, secret := range []string{"primary-secret-fixture", "dev-local-secret", "router-secret", "budget-secret-fixture", "custom-secret-fixture"} {
		if strings.Contains(text, secret) {
			t.Fatalf("read model leaked secret %q in JSON: %s", secret, text)
		}
	}
}

func TestRouterRecursionDetectedForRoutePointingAtLocalListen(t *testing.T) {
	cfg := config.Config{
		Router: config.RouterCfg{
			Enabled:   true,
			Listen:    "127.0.0.1:8787",
			APIKeyEnv: "GORMES_ROUTER_API_KEY",
			Routes: []config.RouterRouteCfg{{
				Name:     "self-route",
				Alias:    "self-chat",
				Provider: "custom",
				Model:    "self-model",
				BaseURL:  "http://127.0.0.1:8787/v1?key=must-not-leak",
			}},
		},
	}

	if err := ValidateNoRecursion(cfg.Router); err == nil || !strings.Contains(err.Error(), "router_recursion_detected") || strings.Contains(err.Error(), "must-not-leak") {
		t.Fatalf("ValidateNoRecursion err = %v, want redacted router_recursion_detected", err)
	}
	model := BuildReadModel(cfg, Options{SkipPrimary: true})
	if model.Status.State != RouterStatusInvalidRoute {
		t.Fatalf("router status = %q, want %q (model=%+v)", model.Status.State, RouterStatusInvalidRoute, model)
	}
	assertRouteStatus(t, model.Routes, "self-chat", RouteStatusInvalidRoute)
	var evidence string
	for _, route := range model.Routes {
		if route.Alias == "self-chat" {
			evidence = strings.Join(route.Evidence, ",")
		}
	}
	if !strings.Contains(evidence, "router_recursion_detected") || strings.Contains(evidence, "must-not-leak") {
		t.Fatalf("route evidence = %q, want redacted router_recursion_detected", evidence)
	}
}

func TestRouterFallbackRejectsAuthPolicyAndMalformedClasses(t *testing.T) {
	cfg := routerFixtureConfig()
	cfg.Router.Fallback = []config.RouterFallbackCfg{{
		From: "primary-chat",
		To:   "budget-chat",
		On:   []string{"429", "408", "500", "502", "503", "504", "connection_failure", "auth", "policy", "malformed_request"},
	}}
	model := BuildReadModel(cfg, Options{})
	fallback := model.Fallback[0]
	want := []string{"rate_limit", "timeout", "server_error", "server_error", "server_error", "server_error", "connection_failure"}
	if strings.Join(fallback.On, ",") != strings.Join(want, ",") {
		t.Fatalf("fallback.On = %#v, want %#v", fallback.On, want)
	}
	for _, rejected := range []string{"auth", "policy", "malformed_request"} {
		if !contains(fallback.Rejected, rejected) {
			t.Fatalf("fallback rejected classes = %#v, want %q rejected", fallback.Rejected, rejected)
		}
	}
}

func loadRouterFixture(t *testing.T, body string) config.Config {
	t.Helper()
	cfgHome := t.TempDir()
	gormesHome := filepath.Join(cfgHome, "gormes")
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("GORMES_API_KEY", "")
	if err := os.MkdirAll(gormesHome, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func routerFixtureConfig() config.Config {
	return config.Config{
		Hermes: config.HermesCfg{
			Provider: "deepseek",
			Model:    "deepseek-chat",
			Endpoint: "https://api.deepseek.com/v1",
			APIKey:   "primary-secret-fixture",
		},
		Router: config.RouterCfg{
			Enabled:    true,
			Listen:     "127.0.0.1:8787",
			APIKeys:    []string{"dev-local-secret"},
			APIKeyEnv:  "GORMES_ROUTER_LOCAL_KEY",
			RedactLogs: true,
			SetupMode:  "local_gateway",
			Routes: []config.RouterRouteCfg{
				{Name: "budget-openrouter", Provider: "openrouter", Model: "moonshotai/kimi-k2:free", Alias: "budget-chat", BaseURL: "https://openrouter.ai/api/v1", APIKeyEnv: "ROUTER_OPENROUTER_KEY", Optional: true},
				{Name: "local-openai-compatible", Provider: "custom", Model: "llama3.2", Alias: "local-chat", BaseURL: "http://127.0.0.1:11434/v1", Optional: true},
				{Name: "custom-compatible", Provider: "custom", Model: "my-model", Alias: "custom-chat", BaseURL: "https://llm.example/v1", APIKeyEnv: "ROUTER_CUSTOM_KEY"},
			},
			Fallback: []config.RouterFallbackCfg{{From: "primary-chat", To: "budget-chat", On: []string{"rate_limit", "server_error", "timeout"}}},
		},
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func assertModelRoute(t *testing.T, models []Model, alias, provider, upstreamModel string, status RouteStatus, quota string) {
	t.Helper()
	for _, model := range models {
		if model.ID == alias {
			if model.Provider != provider || model.Model != upstreamModel || model.Status != status || model.QuotaStatus != quota {
				t.Fatalf("model %q = %+v", alias, model)
			}
			return
		}
	}
	t.Fatalf("model alias %q not found in %+v", alias, models)
}

func assertRouteStatus(t *testing.T, routes []Route, alias string, status RouteStatus) {
	t.Helper()
	for _, route := range routes {
		if route.Alias == alias {
			if route.Status != status {
				t.Fatalf("route %q status = %q, want %q (route=%+v)", alias, route.Status, status, route)
			}
			return
		}
	}
	t.Fatalf("route alias %q not found in %+v", alias, routes)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
