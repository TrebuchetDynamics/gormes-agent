package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers"
	"github.com/spf13/cobra"
)

const (
	setupRouterAPIKeyEnvDefault = "GORMES_ROUTER_API_KEY"
)

func runSetupRouterSection(cmd *cobra.Command, _ setupCommandSeams, _ bool) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup router: load config: %w", err)
	}

	routerCfg, authEvidence, err := buildSetupRouterConfig(cfg)
	if err != nil {
		return err
	}
	if err := providermodule.ValidateRouterNoRecursion(routerCfg); err != nil {
		return err
	}
	if authEvidence.generatedKey != "" {
		if err := config.WriteEnvValue(config.EnvPath(), authEvidence.apiKeyEnv, authEvidence.generatedKey); err != nil {
			return fmt.Errorf("setup router: write local API key: %w", err)
		}
	}
	if err := config.WriteRouterConfig(config.ConfigPath(), routerCfg); err != nil {
		return fmt.Errorf("setup router: write config: %w", err)
	}
	writeSetupRouterReceipt(cmd, routerCfg, authEvidence)
	return nil
}

type setupRouterAuthEvidence struct {
	apiKeyEnv    string
	generatedKey string
}

func buildSetupRouterConfig(cfg config.Config) (config.RouterCfg, setupRouterAuthEvidence, error) {
	listen := firstNonEmptySetup(os.Getenv("GORMES_ROUTER_LISTEN"), cfg.Router.Listen, providermodule.RouterDefaultListen)
	apiKeyEnv := firstNonEmptySetup(os.Getenv("GORMES_ROUTER_API_KEY_ENV"), cfg.Router.APIKeyEnv, setupRouterAPIKeyEnvDefault)
	auth := setupRouterAuthEvidence{apiKeyEnv: apiKeyEnv}
	if value := strings.TrimSpace(os.Getenv(apiKeyEnv)); value == "" {
		key, err := generateSetupRouterAPIKey()
		if err != nil {
			return config.RouterCfg{}, setupRouterAuthEvidence{}, err
		}
		auth.generatedKey = key
	}

	routes, err := setupRouterRoutes(cfg)
	if err != nil {
		return config.RouterCfg{}, setupRouterAuthEvidence{}, err
	}
	fallback := setupRouterFallbackRules(routes)
	return config.RouterCfg{
		Enabled:    true,
		Listen:     listen,
		APIKeyEnv:  apiKeyEnv,
		RedactLogs: true,
		SetupMode:  providermodule.RouterDefaultSetupMode,
		Routes:     routes,
		Fallback:   fallback,
	}, auth, nil
}

func setupRouterRoutes(cfg config.Config) ([]config.RouterRouteCfg, error) {
	routes := make([]config.RouterRouteCfg, 0, 4)
	if primary, ok := setupRouterPrimaryRoute(cfg); ok {
		routes = append(routes, primary)
	}
	fallbackCfg, err := providermodule.LoadFallbackConfig(config.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("setup router: load fallback providers: %w", err)
	}
	for i, entry := range fallbackCfg.Chain {
		if route, ok := setupRouterRouteFromFallback(entry, i+1); ok {
			routes = append(routes, route)
		}
	}
	if route, ok, err := setupRouterRouteFromEnv(); err != nil {
		return nil, err
	} else if ok {
		routes = append(routes, route)
	}
	for _, existing := range cfg.Router.Routes {
		routes = appendSetupRouterRoute(routes, existing)
	}
	return routes, nil
}

func setupRouterPrimaryRoute(cfg config.Config) (config.RouterRouteCfg, bool) {
	provider := setupCanonicalProviderID(cfg.Hermes.Provider)
	model := strings.TrimSpace(cfg.Hermes.Model)
	if provider == "" || model == "" || model == "hermes-agent" {
		return config.RouterRouteCfg{}, false
	}
	return config.RouterRouteCfg{
		Name:      "primary-provider",
		Alias:     "primary-chat",
		Provider:  provider,
		Model:     model,
		BaseURL:   cleanSetupProviderEndpoint(cfg.Hermes.Endpoint),
		APIKeyRef: cfg.Hermes.APIKeyRef,
		Transport: providermodule.RouterDefaultTransport,
	}, true
}

func setupRouterRouteFromFallback(entry providermodule.FallbackEntry, ordinal int) (config.RouterRouteCfg, bool) {
	provider := setupCanonicalProviderID(entry.Provider)
	model := strings.TrimSpace(entry.Model)
	if provider == "" || model == "" {
		return config.RouterRouteCfg{}, false
	}
	alias := fmt.Sprintf("fallback-%s-%d", setupRouterSlug(provider), ordinal)
	return config.RouterRouteCfg{
		Name:      alias,
		Alias:     alias,
		Provider:  provider,
		Model:     model,
		BaseURL:   firstNonEmptySetup(cleanSetupProviderEndpoint(entry.BaseURL), setupProviderEndpointDefault(provider)),
		Transport: providermodule.RouterDefaultTransport,
	}, true
}

func setupRouterRouteFromEnv() (config.RouterRouteCfg, bool, error) {
	provider := setupCanonicalProviderID(os.Getenv("GORMES_ROUTER_ROUTE_PROVIDER"))
	model := strings.TrimSpace(os.Getenv("GORMES_ROUTER_ROUTE_MODEL"))
	alias := strings.TrimSpace(os.Getenv("GORMES_ROUTER_ROUTE_ALIAS"))
	baseURL := cleanSetupProviderEndpoint(os.Getenv("GORMES_ROUTER_ROUTE_BASE_URL"))
	apiKeyEnv := strings.TrimSpace(os.Getenv("GORMES_ROUTER_ROUTE_API_KEY_ENV"))
	optionalRaw := strings.TrimSpace(os.Getenv("GORMES_ROUTER_ROUTE_OPTIONAL"))
	anySet := provider != "" || model != "" || alias != "" || baseURL != "" || apiKeyEnv != "" || optionalRaw != ""
	if !anySet {
		return config.RouterRouteCfg{}, false, nil
	}
	if provider == "" {
		return config.RouterRouteCfg{}, false, fmt.Errorf("setup router: GORMES_ROUTER_ROUTE_PROVIDER is required when route env is set")
	}
	if model == "" {
		model = setupProviderModelDefault(cli.ProviderModel{}, provider)
	}
	if model == "" {
		return config.RouterRouteCfg{}, false, fmt.Errorf("setup router: GORMES_ROUTER_ROUTE_MODEL is required for provider %s", provider)
	}
	if baseURL == "" {
		baseURL = setupProviderEndpointDefault(provider)
	}
	if provider == "custom" && baseURL == "" {
		return config.RouterRouteCfg{}, false, fmt.Errorf("setup router: GORMES_ROUTER_ROUTE_BASE_URL is required for custom routes")
	}
	if alias == "" {
		alias = "route-" + setupRouterSlug(provider)
	}
	optional := false
	if optionalRaw != "" {
		parsed, err := strconv.ParseBool(optionalRaw)
		if err != nil {
			return config.RouterRouteCfg{}, false, fmt.Errorf("setup router: GORMES_ROUTER_ROUTE_OPTIONAL: %w", err)
		}
		optional = parsed
	}
	return config.RouterRouteCfg{
		Name:      alias,
		Alias:     alias,
		Provider:  provider,
		Model:     model,
		BaseURL:   baseURL,
		APIKeyEnv: apiKeyEnv,
		Transport: providermodule.RouterDefaultTransport,
		Optional:  optional,
	}, true, nil
}

func appendSetupRouterRoute(routes []config.RouterRouteCfg, route config.RouterRouteCfg) []config.RouterRouteCfg {
	alias := strings.TrimSpace(route.Alias)
	name := strings.TrimSpace(route.Name)
	for _, existing := range routes {
		if alias != "" && strings.EqualFold(strings.TrimSpace(existing.Alias), alias) {
			return routes
		}
		if name != "" && strings.EqualFold(strings.TrimSpace(existing.Name), name) {
			return routes
		}
	}
	return append(routes, route)
}

func setupRouterFallbackRules(routes []config.RouterRouteCfg) []config.RouterFallbackCfg {
	return gormescli.SetupRouterFallbackRules(routes)
}

func writeSetupRouterReceipt(cmd *cobra.Command, cfg config.RouterCfg, auth setupRouterAuthEvidence) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Gormes Router configured.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Connection")
	fmt.Fprintln(out, "  Role: local client endpoint, not an upstream provider")
	fmt.Fprintf(out, "  Listen: %s\n", cfg.Listen)
	fmt.Fprintf(out, "  OpenAI base URL: %s\n", setupRouterOpenAIBaseURL(cfg.Listen))
	fmt.Fprintf(out, "  Config: %s\n", config.ConfigPath())
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Authentication")
	if auth.generatedKey != "" {
		fmt.Fprintf(out, "  API key: generated into %s as %s (redacted)\n", config.EnvPath(), auth.apiKeyEnv)
	} else {
		fmt.Fprintf(out, "  API key: referenced via %s (redacted)\n", auth.apiKeyEnv)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Routes")
	if len(cfg.Routes) == 0 {
		fmt.Fprintln(out, "  No upstream routes yet. Run `gormes setup provider` or set GORMES_ROUTER_ROUTE_* env vars.")
	} else {
		for _, route := range cfg.Routes {
			fmt.Fprintf(out, "  %s -> %s/%s", firstNonEmptySetup(route.Alias, route.Name), route.Provider, route.Model)
			labels := setupRouterRouteLabels(route)
			if len(labels) > 0 {
				fmt.Fprintf(out, " (%s)", strings.Join(labels, "; "))
			}
			fmt.Fprintln(out)
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Next steps")
	fmt.Fprintln(out, "  1. Configure upstream credentials with: gormes setup provider")
	fmt.Fprintln(out, "  2. Point OpenAI-compatible tools at the OpenAI base URL above.")
	fmt.Fprintln(out, "  3. The provider picker remains: gormes setup provider")
}

func setupRouterRouteLabels(route config.RouterRouteCfg) []string {
	return gormescli.SetupRouterRouteLabels(route)
}

func setupRouterOpenAIBaseURL(listen string) string {
	return gormescli.SetupRouterOpenAIBaseURL(listen, providermodule.RouterDefaultListen)
}

func setupRouterSlug(value string) string {
	return gormescli.SetupRouterSlug(value)
}

func generateSetupRouterAPIKey() (string, error) {
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("setup router: generate local API key: %w", err)
	}
	return "grt_" + hex.EncodeToString(raw[:]), nil
}
