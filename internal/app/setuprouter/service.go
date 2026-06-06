package setuprouter

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	appfallback "github.com/TrebuchetDynamics/gormes-agent/internal/app/fallback"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	routerpkg "github.com/TrebuchetDynamics/gormes-agent/internal/provider/router"
)

const apiKeyEnvDefault = "GORMES_ROUTER_API_KEY"

var aliasSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

var knownProviderEndpoints = map[string]string{
	"openai":       "https://api.openai.com/v1",
	"anthropic":    "https://api.anthropic.com/v1",
	"deepseek":     "https://api.deepseek.com/v1",
	"groq":         "https://api.groq.com/openai/v1",
	"ollama":       "http://localhost:11434/v1",
	"openai-codex": "https://chatgpt.com/backend-api/codex",
	"openrouter":   llm.OpenRouterDefaultBaseURL,
	"opencode":     "https://opencode.ai/zen/v1",
	"opencode-go":  "https://opencode.ai/zen/go/v1",
}

var knownProviderModels = map[string]string{
	"openai":       "gpt-4o",
	"anthropic":    "claude-sonnet-4-20250514",
	"deepseek":     "deepseek-chat",
	"groq":         "llama-3.3-70b-versatile",
	"ollama":       "llama3",
	"openai-codex": "gpt-5.2",
	"openrouter":   "moonshotai/kimi-k2.6",
	"opencode":     "gpt-5.2",
	"opencode-go":  "gpt-5.2",
}

// Options are setup-router command-local seams. The defaults preserve the
// root command's current config paths, env lookup, and TOML write behavior.
type Options struct {
	LoadConfig         func() (config.Config, error)
	ConfigPath         func() string
	EnvPath            func() string
	LookupEnv          func(string) string
	WriteEnvValue      func(path, key, value string) error
	WriteRouterConfig  func(path string, cfg config.RouterCfg) error
	LoadFallbackConfig func(path string) (appfallback.FallbackConfig, error)
	GenerateAPIKey     func() (string, error)
}

type AuthEvidence struct {
	apiKeyEnv    string
	generatedKey string
}

func (opts Options) withDefaults() Options {
	if opts.LoadConfig == nil {
		opts.LoadConfig = func() (config.Config, error) { return config.Load(nil) }
	}
	if opts.ConfigPath == nil {
		opts.ConfigPath = config.ConfigPath
	}
	if opts.EnvPath == nil {
		opts.EnvPath = config.EnvPath
	}
	if opts.LookupEnv == nil {
		opts.LookupEnv = os.Getenv
	}
	if opts.WriteEnvValue == nil {
		opts.WriteEnvValue = config.WriteEnvValue
	}
	if opts.WriteRouterConfig == nil {
		opts.WriteRouterConfig = config.WriteRouterConfig
	}
	if opts.LoadFallbackConfig == nil {
		opts.LoadFallbackConfig = appfallback.LoadFallbackConfig
	}
	if opts.GenerateAPIKey == nil {
		opts.GenerateAPIKey = GenerateAPIKey
	}
	return opts
}

// Run configures the local OpenAI-compatible router setup section and writes
// the operator receipt. It is command-local behavior; Cobra wiring lives in
// internal/platform/cli/gormescli.
func Run(out io.Writer, opts Options) error {
	opts = opts.withDefaults()
	if out == nil {
		out = io.Discard
	}
	cfg, err := opts.LoadConfig()
	if err != nil {
		return fmt.Errorf("setup router: load config: %w", err)
	}

	routerCfg, auth, err := BuildConfig(cfg, opts)
	if err != nil {
		return err
	}
	if err := routerpkg.ValidateNoRecursion(routerCfg); err != nil {
		return err
	}
	if auth.generatedKey != "" {
		if err := opts.WriteEnvValue(opts.EnvPath(), auth.apiKeyEnv, auth.generatedKey); err != nil {
			return fmt.Errorf("setup router: write local API key: %w", err)
		}
	}
	if err := opts.WriteRouterConfig(opts.ConfigPath(), routerCfg); err != nil {
		return fmt.Errorf("setup router: write config: %w", err)
	}
	PrintReceipt(out, routerCfg, auth, opts.ConfigPath())
	return nil
}

// BuildConfig returns the router config and local-auth evidence without
// mutating files. The supplied Options may include env/config seams.
func BuildConfig(cfg config.Config, opts Options) (config.RouterCfg, AuthEvidence, error) {
	opts = opts.withDefaults()
	listen := firstNonEmpty(opts.LookupEnv("GORMES_ROUTER_LISTEN"), cfg.Router.Listen, routerpkg.DefaultListen)
	apiKeyEnv := firstNonEmpty(opts.LookupEnv("GORMES_ROUTER_API_KEY_ENV"), cfg.Router.APIKeyEnv, apiKeyEnvDefault)
	auth := AuthEvidence{apiKeyEnv: apiKeyEnv}
	if value := strings.TrimSpace(opts.LookupEnv(apiKeyEnv)); value == "" {
		key, err := opts.GenerateAPIKey()
		if err != nil {
			return config.RouterCfg{}, AuthEvidence{}, err
		}
		auth.generatedKey = key
	}

	routes, err := Routes(cfg, opts)
	if err != nil {
		return config.RouterCfg{}, AuthEvidence{}, err
	}
	return config.RouterCfg{
		Enabled:    true,
		Listen:     listen,
		APIKeyEnv:  apiKeyEnv,
		RedactLogs: true,
		SetupMode:  routerpkg.DefaultSetupMode,
		Routes:     routes,
		Fallback:   FallbackRules(routes),
	}, auth, nil
}

// Routes builds the router upstream routes from the current primary provider,
// fallback providers, explicit setup env vars, and existing router routes.
func Routes(cfg config.Config, opts Options) ([]config.RouterRouteCfg, error) {
	opts = opts.withDefaults()
	routes := make([]config.RouterRouteCfg, 0, 4)
	if primary, ok := PrimaryRoute(cfg); ok {
		routes = append(routes, primary)
	}
	fallbackCfg, err := opts.LoadFallbackConfig(opts.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("setup router: load fallback providers: %w", err)
	}
	for i, entry := range fallbackCfg.Chain {
		if route, ok := RouteFromFallback(entry, i+1, opts); ok {
			routes = append(routes, route)
		}
	}
	if route, ok, err := RouteFromEnv(opts); err != nil {
		return nil, err
	} else if ok {
		routes = append(routes, route)
	}
	for _, existing := range cfg.Router.Routes {
		routes = AppendRoute(routes, existing)
	}
	return routes, nil
}

func PrimaryRoute(cfg config.Config) (config.RouterRouteCfg, bool) {
	provider := CanonicalProviderID(cfg.Hermes.Provider)
	model := strings.TrimSpace(cfg.Hermes.Model)
	if provider == "" || model == "" || model == "hermes-agent" {
		return config.RouterRouteCfg{}, false
	}
	return config.RouterRouteCfg{
		Name:      "primary-provider",
		Alias:     "primary-chat",
		Provider:  provider,
		Model:     model,
		BaseURL:   CleanEndpoint(cfg.Hermes.Endpoint),
		APIKeyRef: cfg.Hermes.APIKeyRef,
		Transport: routerpkg.DefaultTransport,
	}, true
}

func RouteFromFallback(entry appfallback.FallbackEntry, ordinal int, opts Options) (config.RouterRouteCfg, bool) {
	opts = opts.withDefaults()
	provider := CanonicalProviderID(entry.Provider)
	model := strings.TrimSpace(entry.Model)
	if provider == "" || model == "" {
		return config.RouterRouteCfg{}, false
	}
	alias := fmt.Sprintf("fallback-%s-%d", Slug(provider), ordinal)
	return config.RouterRouteCfg{
		Name:      alias,
		Alias:     alias,
		Provider:  provider,
		Model:     model,
		BaseURL:   firstNonEmpty(CleanEndpoint(entry.BaseURL), ProviderEndpointDefault(provider, opts.LookupEnv)),
		Transport: routerpkg.DefaultTransport,
	}, true
}

func RouteFromEnv(opts Options) (config.RouterRouteCfg, bool, error) {
	opts = opts.withDefaults()
	provider := CanonicalProviderID(opts.LookupEnv("GORMES_ROUTER_ROUTE_PROVIDER"))
	model := strings.TrimSpace(opts.LookupEnv("GORMES_ROUTER_ROUTE_MODEL"))
	alias := strings.TrimSpace(opts.LookupEnv("GORMES_ROUTER_ROUTE_ALIAS"))
	baseURL := CleanEndpoint(opts.LookupEnv("GORMES_ROUTER_ROUTE_BASE_URL"))
	apiKeyEnv := strings.TrimSpace(opts.LookupEnv("GORMES_ROUTER_ROUTE_API_KEY_ENV"))
	optionalRaw := strings.TrimSpace(opts.LookupEnv("GORMES_ROUTER_ROUTE_OPTIONAL"))
	anySet := provider != "" || model != "" || alias != "" || baseURL != "" || apiKeyEnv != "" || optionalRaw != ""
	if !anySet {
		return config.RouterRouteCfg{}, false, nil
	}
	if provider == "" {
		return config.RouterRouteCfg{}, false, fmt.Errorf("setup router: GORMES_ROUTER_ROUTE_PROVIDER is required when route env is set")
	}
	if model == "" {
		model = ProviderModelDefault(provider)
	}
	if model == "" {
		return config.RouterRouteCfg{}, false, fmt.Errorf("setup router: GORMES_ROUTER_ROUTE_MODEL is required for provider %s", provider)
	}
	if baseURL == "" {
		baseURL = ProviderEndpointDefault(provider, opts.LookupEnv)
	}
	if provider == "custom" && baseURL == "" {
		return config.RouterRouteCfg{}, false, fmt.Errorf("setup router: GORMES_ROUTER_ROUTE_BASE_URL is required for custom routes")
	}
	if alias == "" {
		alias = "route-" + Slug(provider)
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
		Transport: routerpkg.DefaultTransport,
		Optional:  optional,
	}, true, nil
}

func AppendRoute(routes []config.RouterRouteCfg, route config.RouterRouteCfg) []config.RouterRouteCfg {
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

// FallbackRules builds primary-chat fallback rules for every other route alias.
func FallbackRules(routes []config.RouterRouteCfg) []config.RouterFallbackCfg {
	if len(routes) < 2 {
		return nil
	}
	primary := ""
	for _, route := range routes {
		if strings.EqualFold(strings.TrimSpace(route.Alias), "primary-chat") {
			primary = "primary-chat"
			break
		}
	}
	if primary == "" {
		return nil
	}
	rules := make([]config.RouterFallbackCfg, 0, len(routes)-1)
	for _, route := range routes {
		alias := strings.TrimSpace(route.Alias)
		if alias == "" || alias == primary {
			continue
		}
		rules = append(rules, config.RouterFallbackCfg{
			From: primary,
			To:   alias,
			On:   []string{"rate_limit", "server_error", "timeout", "connection_failure"},
		})
	}
	return rules
}

func PrintReceipt(out io.Writer, cfg config.RouterCfg, auth AuthEvidence, configPath string) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Gormes Router configured.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Connection")
	fmt.Fprintln(out, "  Role: local client endpoint, not an upstream provider")
	fmt.Fprintf(out, "  Listen: %s\n", cfg.Listen)
	fmt.Fprintf(out, "  OpenAI base URL: %s\n", OpenAIBaseURL(cfg.Listen, routerpkg.DefaultListen))
	fmt.Fprintf(out, "  Config: %s\n", configPath)
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
			fmt.Fprintf(out, "  %s -> %s/%s", firstNonEmpty(route.Alias, route.Name), route.Provider, route.Model)
			labels := RouteLabels(route)
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

// RouteLabels returns operator-facing caution labels for setup router receipts.
func RouteLabels(route config.RouterRouteCfg) []string {
	labels := []string{}
	text := strings.ToLower(strings.Join([]string{route.Name, route.Alias, route.Provider, route.Model}, " "))
	if strings.Contains(text, ":free") || strings.Contains(text, "free-tier") || strings.Contains(text, "free_tier") {
		labels = append(labels, "requires your provider account/API key; quotas are provider-controlled")
	}
	if route.Optional {
		labels = append(labels, "optional; only enabled if already installed and healthy")
	}
	return labels
}

// OpenAIBaseURL converts a router listen address into its OpenAI-compatible /v1 base URL.
func OpenAIBaseURL(listen, defaultListen string) string {
	listen = strings.TrimSpace(listen)
	if listen == "" {
		listen = defaultListen
	}
	if !strings.Contains(listen, "://") {
		listen = "http://" + listen
	}
	parsed, err := url.Parse(listen)
	if err != nil || parsed.Host == "" {
		return strings.TrimRight(listen, "/") + "/v1"
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/v1"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.User = nil
	return parsed.String()
}

// Slug normalizes a provider or route label for generated aliases.
func Slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = aliasSanitizer.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "route"
	}
	return value
}

func CanonicalProviderID(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "custom-endpoint" {
		return "custom"
	}
	if entry, ok := llm.ResolveProviderManifestEntry(provider); ok {
		return strings.TrimSpace(entry.ID)
	}
	return provider
}

func ProviderEndpointDefault(provider string, lookupEnv func(string) string) string {
	if lookupEnv == nil {
		lookupEnv = os.Getenv
	}
	provider = CanonicalProviderID(provider)
	if endpoint := providerEndpointEnvDefault(provider, lookupEnv); endpoint != "" {
		return endpoint
	}
	if endpoint := providerBaseURL(provider, ""); strings.TrimSpace(endpoint) != "" {
		return CleanEndpoint(endpoint)
	}
	if endpoint := knownProviderEndpoints[provider]; strings.TrimSpace(endpoint) != "" {
		return CleanEndpoint(endpoint)
	}
	if entry, ok := llm.ResolveProviderManifestEntry(provider); ok {
		if endpoint := providerEndpointEnvDefault(entry.ID, lookupEnv); endpoint != "" {
			return endpoint
		}
		if endpoint := providerBaseURL(entry.ID, ""); strings.TrimSpace(endpoint) != "" {
			return CleanEndpoint(endpoint)
		}
		if endpoint := knownProviderEndpoints[entry.ID]; strings.TrimSpace(endpoint) != "" {
			return CleanEndpoint(endpoint)
		}
		if endpoint := strings.TrimSpace(entry.BaseURLOverride); endpoint != "" {
			return CleanEndpoint(endpoint)
		}
	}
	return ""
}

func providerEndpointEnvDefault(provider string, lookupEnv func(string) string) string {
	if entry, ok := llm.ResolveProviderManifestEntry(provider); ok {
		if endpoint := CleanEndpoint(lookupEnv(entry.BaseURLEnvVar)); endpoint != "" {
			return endpoint
		}
	}
	return ""
}

func CleanEndpoint(endpoint string) string {
	return strings.TrimRight(strings.TrimSpace(endpoint), "/")
}

func ProviderModelDefault(provider string) string {
	provider = CanonicalProviderID(provider)
	if resolved := llm.ResolveProviderDefaultModel(provider, llm.ProviderDefaultModelOptions{}); strings.TrimSpace(resolved.Model) != "" {
		return strings.TrimSpace(resolved.Model)
	}
	if model := knownProviderModels[provider]; strings.TrimSpace(model) != "" {
		return strings.TrimSpace(model)
	}
	return ""
}

func providerBaseURL(provider, override string) string {
	if baseURL := CleanEndpoint(override); baseURL != "" {
		return baseURL
	}
	switch provider {
	case "openrouter":
		return llm.OpenRouterDefaultBaseURL
	case config.AnthropicProvider:
		return "https://api.anthropic.com"
	case config.CodexOAuthProvider:
		return "https://chatgpt.com/backend-api/codex"
	case config.NousOAuthProvider:
		return "https://inference-api.nousresearch.com/v1"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/openai"
	case "groq":
		return "https://api.groq.com/openai/v1"
	case "novita":
		return "https://api.novita.ai/openai/v1"
	case "google-gemini-cli":
		return "cloudcode-pa://google"
	case "qwen-oauth":
		return "https://portal.qwen.ai/v1"
	default:
		return ""
	}
}

func GenerateAPIKey() (string, error) {
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("setup router: generate local API key: %w", err)
	}
	return "grt_" + hex.EncodeToString(raw[:]), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
