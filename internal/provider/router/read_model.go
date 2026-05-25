package router

import (
	"context"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

const (
	DefaultListen    = "127.0.0.1:8787"
	DefaultSetupMode = "local_gateway"
	DefaultTransport = "chat_completions"

	QuotaProviderControlled = "provider_controlled_quota"
)

type RouterStatus string

const (
	RouterStatusDisabled          RouterStatus = "disabled"
	RouterStatusConfigured        RouterStatus = "configured"
	RouterStatusMissingCredential RouterStatus = "missing_credential"
	RouterStatusInvalidRoute      RouterStatus = "invalid_route"
)

type RouteStatus string

const (
	RouteStatusConfigured        RouteStatus = "configured"
	RouteStatusAvailable         RouteStatus = "available"
	RouteStatusUnavailable       RouteStatus = "unavailable"
	RouteStatusMissingCredential RouteStatus = "missing_credential"
	RouteStatusInvalidRoute      RouteStatus = "invalid_route"
)

type CredentialStatus string

const (
	CredentialConfigured CredentialStatus = "configured"
	CredentialMissing    CredentialStatus = "missing_credential"
	CredentialNotNeeded  CredentialStatus = "not_required"
)

type Options struct {
	LookupEnv    func(string) (string, bool)
	Probe        ProbeFunc
	ProbeContext context.Context
	SkipPrimary  bool
}

type ProbeFunc func(context.Context, Route) ProbeResult

type ProbeResult struct {
	Available bool
	Evidence  []string
}

type AuthStatus struct {
	Configured     bool     `json:"configured"`
	APIKeyEnv      string   `json:"api_key_env,omitempty"`
	InlineKeyCount int      `json:"inline_key_count,omitempty"`
	Evidence       []string `json:"evidence,omitempty"`
	Redacted       bool     `json:"redacted"`
}

type Status struct {
	State      RouterStatus   `json:"state"`
	Evidence   []string       `json:"evidence,omitempty"`
	ByStatus   map[string]int `json:"by_status,omitempty"`
	RouteCount int            `json:"route_count"`
}

type Route struct {
	Name             string           `json:"name"`
	Alias            string           `json:"alias"`
	Provider         string           `json:"provider"`
	Model            string           `json:"model"`
	BaseURL          string           `json:"base_url,omitempty"`
	Transport        string           `json:"transport"`
	Optional         bool             `json:"optional,omitempty"`
	Weight           int              `json:"weight,omitempty"`
	Local            bool             `json:"local,omitempty"`
	Status           RouteStatus      `json:"status"`
	CredentialStatus CredentialStatus `json:"credential_status"`
	QuotaStatus      string           `json:"quota_status,omitempty"`
	Evidence         []string         `json:"evidence,omitempty"`
}

type FallbackRule struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	On       []string `json:"on,omitempty"`
	Rejected []string `json:"rejected,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
}

type ReadModel struct {
	Enabled    bool           `json:"enabled"`
	Listen     string         `json:"listen"`
	SetupMode  string         `json:"setup_mode"`
	RedactLogs bool           `json:"redact_logs"`
	Auth       AuthStatus     `json:"auth"`
	Status     Status         `json:"status"`
	Routes     []Route        `json:"routes,omitempty"`
	Fallback   []FallbackRule `json:"fallback,omitempty"`
}

type Model struct {
	ID          string      `json:"id"`
	Provider    string      `json:"provider"`
	Model       string      `json:"model"`
	RouteName   string      `json:"route_name"`
	Status      RouteStatus `json:"status"`
	Optional    bool        `json:"optional,omitempty"`
	QuotaStatus string      `json:"quota_status,omitempty"`
}

type Registry struct {
	routes []Route
}

func BuildReadModel(cfg config.Config, opts Options) ReadModel {
	lookupEnv := opts.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	listen := strings.TrimSpace(cfg.Router.Listen)
	if listen == "" {
		listen = DefaultListen
	}
	setupMode := strings.TrimSpace(cfg.Router.SetupMode)
	if setupMode == "" {
		setupMode = DefaultSetupMode
	}
	redactLogs := cfg.Router.RedactLogs
	if !cfg.Router.Enabled && cfg.Router.Listen == "" && cfg.Router.SetupMode == "" && len(cfg.Router.Routes) == 0 && len(cfg.Router.Fallback) == 0 {
		redactLogs = true
	}

	model := ReadModel{
		Enabled:    cfg.Router.Enabled,
		Listen:     listen,
		SetupMode:  setupMode,
		RedactLogs: redactLogs,
		Auth:       buildAuthStatus(cfg.Router, lookupEnv),
	}
	if !cfg.Router.Enabled {
		model.Status = Status{State: RouterStatusDisabled, Evidence: []string{"router_disabled"}}
		return model
	}

	routes := make([]config.RouterRouteCfg, 0, len(cfg.Router.Routes)+1)
	if !opts.SkipPrimary && !hasPrimaryRoute(cfg.Router.Routes) {
		if primary, ok := primaryRoute(cfg); ok {
			routes = append(routes, primary)
		}
	}
	routes = append(routes, cfg.Router.Routes...)
	model.Routes = buildRoutes(cfg, routes, lookupEnv, opts)
	model.Fallback = buildFallbackRules(cfg.Router.Fallback)
	model.Status = buildStatus(model.Routes)
	return model
}

func NewRegistry(model ReadModel) Registry {
	routes := append([]Route(nil), model.Routes...)
	sort.SliceStable(routes, func(i, j int) bool {
		if routes[i].Alias == routes[j].Alias {
			return routes[i].Name < routes[j].Name
		}
		return routes[i].Alias < routes[j].Alias
	})
	return Registry{routes: routes}
}

func (r Registry) Models() []Model {
	models := make([]Model, 0, len(r.routes))
	for _, route := range r.routes {
		if route.Alias == "" {
			continue
		}
		models = append(models, Model{
			ID:          route.Alias,
			Provider:    route.Provider,
			Model:       route.Model,
			RouteName:   route.Name,
			Status:      route.Status,
			Optional:    route.Optional,
			QuotaStatus: route.QuotaStatus,
		})
	}
	return models
}

func (r Registry) Resolve(model string) []Route {
	want := strings.TrimSpace(model)
	if want == "" {
		return nil
	}
	out := make([]Route, 0, 1)
	for _, route := range r.routes {
		if route.Alias == want || route.Model == want || route.Name == want {
			out = append(out, route)
		}
	}
	return out
}

func buildAuthStatus(cfg config.RouterCfg, lookupEnv func(string) (string, bool)) AuthStatus {
	status := AuthStatus{APIKeyEnv: strings.TrimSpace(cfg.APIKeyEnv), InlineKeyCount: len(compactStrings(cfg.APIKeys)), Redacted: true}
	if status.InlineKeyCount > 0 {
		status.Configured = true
		status.Evidence = append(status.Evidence, "inline_api_keys_configured_redacted")
	}
	if status.APIKeyEnv != "" {
		if value, ok := lookupEnv(status.APIKeyEnv); ok && strings.TrimSpace(value) != "" {
			status.Configured = true
			status.Evidence = append(status.Evidence, "api_key_env_configured")
		} else {
			status.Evidence = append(status.Evidence, "api_key_env_missing")
		}
	}
	if !status.Configured {
		status.Evidence = append(status.Evidence, "inbound_api_key_not_configured")
	}
	return status
}

func hasPrimaryRoute(routes []config.RouterRouteCfg) bool {
	for _, route := range routes {
		if strings.EqualFold(strings.TrimSpace(route.Name), "primary-provider") || strings.EqualFold(strings.TrimSpace(route.Alias), "primary-chat") {
			return true
		}
	}
	return false
}

func primaryRoute(cfg config.Config) (config.RouterRouteCfg, bool) {
	provider := strings.TrimSpace(cfg.Hermes.Provider)
	model := strings.TrimSpace(cfg.Hermes.Model)
	if provider == "" || model == "" || model == "hermes-agent" {
		return config.RouterRouteCfg{}, false
	}
	return config.RouterRouteCfg{
		Name:      "primary-provider",
		Alias:     "primary-chat",
		Provider:  provider,
		Model:     model,
		BaseURL:   cfg.Hermes.Endpoint,
		APIKeyRef: cfg.Hermes.APIKeyRef,
		Transport: "",
	}, true
}

func buildRoutes(cfg config.Config, rawRoutes []config.RouterRouteCfg, lookupEnv func(string) (string, bool), opts Options) []Route {
	routes := make([]Route, 0, len(rawRoutes))
	for _, raw := range rawRoutes {
		route := buildRoute(cfg, raw, lookupEnv, opts)
		routes = append(routes, route)
	}
	return routes
}

func buildRoute(cfg config.Config, raw config.RouterRouteCfg, lookupEnv func(string) (string, bool), opts Options) Route {
	provider := canonicalProvider(raw.Provider)
	transport := normalizeTransport(raw.Transport, provider)
	baseURL := normalizeBaseURL(firstNonEmpty(raw.BaseURL, manifestBaseURL(provider)))
	route := Route{
		Name:      firstNonEmpty(strings.TrimSpace(raw.Name), strings.TrimSpace(raw.Alias), strings.TrimSpace(raw.Model)),
		Alias:     firstNonEmpty(strings.TrimSpace(raw.Alias), strings.TrimSpace(raw.Name), strings.TrimSpace(raw.Model)),
		Provider:  provider,
		Model:     strings.TrimSpace(raw.Model),
		BaseURL:   baseURL,
		Transport: transport,
		Optional:  raw.Optional,
		Weight:    raw.Weight,
		Status:    RouteStatusConfigured,
	}
	if route.Transport == "" {
		route.Transport = DefaultTransport
	}
	if route.Weight < 0 {
		route.Weight = 0
	}
	if isProviderControlledQuotaCandidate(route) {
		route.QuotaStatus = QuotaProviderControlled
		route.Evidence = append(route.Evidence, "provider_controlled_quota")
	}
	if route.Provider == "" {
		route.Status = RouteStatusInvalidRoute
		route.CredentialStatus = CredentialMissing
		route.Evidence = append(route.Evidence, "provider_required")
		return route
	}
	if route.Model == "" {
		route.Status = RouteStatusInvalidRoute
		route.CredentialStatus = CredentialMissing
		route.Evidence = append(route.Evidence, "model_required")
		return route
	}
	if route.BaseURL == "" && route.Provider == "custom" {
		route.Status = RouteStatusInvalidRoute
		route.CredentialStatus = CredentialMissing
		route.Evidence = append(route.Evidence, "base_url_required_for_custom_provider")
		return route
	}
	if route.Transport != DefaultTransport {
		route.Status = RouteStatusInvalidRoute
		route.CredentialStatus = CredentialMissing
		route.Evidence = append(route.Evidence, "unsupported_transport:"+route.Transport)
		return route
	}
	if evidence, ok := recursionEvidence(cfg.Router.Listen, route.BaseURL); ok {
		route.Status = RouteStatusInvalidRoute
		route.CredentialStatus = CredentialMissing
		route.Evidence = append(route.Evidence, evidence)
		return route
	}

	route.Local = isLocalRoute(route.Provider, route.BaseURL)
	route.CredentialStatus = credentialStatus(cfg, raw, route, lookupEnv)
	if route.CredentialStatus == CredentialMissing {
		route.Status = RouteStatusMissingCredential
		route.Evidence = append(route.Evidence, "credential_missing")
		if route.QuotaStatus == QuotaProviderControlled {
			route.Evidence = append(route.Evidence, "free_tier_requires_user_owned_credential")
		}
		return route
	}
	if route.QuotaStatus == QuotaProviderControlled {
		route.Evidence = append(route.Evidence, "free_tier_requires_user_owned_credential")
	}
	if route.Optional && route.Local {
		return applyOptionalLocalProbe(route, opts)
	}
	return route
}

func credentialStatus(cfg config.Config, raw config.RouterRouteCfg, route Route, lookupEnv func(string) (string, bool)) CredentialStatus {
	if route.Local && route.Optional && strings.TrimSpace(raw.APIKeyEnv) == "" && raw.APIKeyRef == nil {
		return CredentialNotNeeded
	}
	if strings.TrimSpace(raw.APIKeyEnv) != "" {
		if value, ok := lookupEnv(strings.TrimSpace(raw.APIKeyEnv)); ok && strings.TrimSpace(value) != "" {
			return CredentialConfigured
		}
		return CredentialMissing
	}
	if raw.APIKeyRef != nil {
		return secretRefCredentialStatus(*raw.APIKeyRef, lookupEnv)
	}
	if route.Name == "primary-provider" {
		if strings.TrimSpace(cfg.Hermes.APIKey) != "" {
			return CredentialConfigured
		}
		if cfg.Hermes.APIKeyRef != nil {
			return secretRefCredentialStatus(*cfg.Hermes.APIKeyRef, lookupEnv)
		}
	}
	if providerEnvConfigured(route.Provider, lookupEnv) {
		return CredentialConfigured
	}
	return CredentialMissing
}

func secretRefCredentialStatus(ref config.SecretRef, lookupEnv func(string) (string, bool)) CredentialStatus {
	if strings.EqualFold(string(ref.Source), string(config.SecretRefSourceEnv)) {
		if value, ok := lookupEnv(strings.TrimSpace(ref.ID)); ok && strings.TrimSpace(value) != "" {
			return CredentialConfigured
		}
		return CredentialMissing
	}
	// File and future secret providers are redacted handles. This read model does
	// not open secret files or execute helpers; setup/status rows can resolve them
	// under their own side-effect budget.
	if strings.TrimSpace(ref.ID) != "" {
		return CredentialConfigured
	}
	return CredentialMissing
}

func providerEnvConfigured(provider string, lookupEnv func(string) (string, bool)) bool {
	entry, ok := hermes.ResolveProviderManifestEntry(provider)
	if !ok {
		return false
	}
	for _, env := range entry.EnvVars {
		if value, ok := lookupEnv(env); ok && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func applyOptionalLocalProbe(route Route, opts Options) Route {
	ctx := opts.ProbeContext
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.Probe == nil {
		route.Status = RouteStatusUnavailable
		route.Evidence = append(route.Evidence, "optional_local_route_unprobed_unavailable")
		return route
	}
	result := opts.Probe(ctx, route)
	route.Evidence = append(route.Evidence, result.Evidence...)
	if result.Available {
		route.Status = RouteStatusAvailable
	} else {
		route.Status = RouteStatusUnavailable
	}
	return route
}

func buildFallbackRules(rawRules []config.RouterFallbackCfg) []FallbackRule {
	rules := make([]FallbackRule, 0, len(rawRules))
	for _, raw := range rawRules {
		rule := FallbackRule{From: strings.TrimSpace(raw.From), To: strings.TrimSpace(raw.To)}
		for _, class := range raw.On {
			normalized, ok := normalizeFallbackClass(class)
			if ok {
				rule.On = append(rule.On, normalized)
				continue
			}
			rejected := strings.ToLower(strings.TrimSpace(class))
			if rejected == "" {
				continue
			}
			rule.Rejected = append(rule.Rejected, rejected)
			rule.Evidence = append(rule.Evidence, "fallback_class_rejected:"+rejected)
		}
		rules = append(rules, rule)
	}
	return rules
}

func normalizeFallbackClass(class string) (string, bool) {
	class = strings.ToLower(strings.TrimSpace(class))
	class = strings.TrimPrefix(class, "http_")
	switch class {
	case "rate_limit", "429", "too_many_requests":
		return "rate_limit", true
	case "timeout", "request_timeout", "408":
		return "timeout", true
	case "server_error", "500", "502", "503", "504", "5xx":
		return "server_error", true
	case "connection_failure", "connection_error", "connect_error", "network_error":
		return "connection_failure", true
	default:
		return "", false
	}
}

func buildStatus(routes []Route) Status {
	status := Status{State: RouterStatusConfigured, RouteCount: len(routes), ByStatus: map[string]int{}}
	for _, route := range routes {
		status.ByStatus[string(route.Status)]++
	}
	if status.ByStatus[string(RouteStatusInvalidRoute)] > 0 {
		status.State = RouterStatusInvalidRoute
		status.Evidence = append(status.Evidence, "invalid_route")
		return status
	}
	if len(routes) > 0 && status.ByStatus[string(RouteStatusConfigured)] == 0 && status.ByStatus[string(RouteStatusAvailable)] == 0 && status.ByStatus[string(RouteStatusUnavailable)] == 0 {
		status.State = RouterStatusMissingCredential
		status.Evidence = append(status.Evidence, "missing_credential")
		return status
	}
	status.Evidence = append(status.Evidence, "router_configured")
	return status
}

func canonicalProvider(provider string) string {
	provider = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(provider)), "_", "-")
	if entry, ok := hermes.ResolveProviderManifestEntry(provider); ok {
		return entry.ID
	}
	return provider
}

func normalizeTransport(transport, provider string) string {
	transport = strings.ToLower(strings.TrimSpace(transport))
	if transport == "" {
		entry, ok := hermes.ResolveProviderManifestEntry(provider)
		if !ok || entry.TransportFamily == "" || entry.TransportFamily == "openai_chat" {
			return DefaultTransport
		}
		return entry.TransportFamily
	}
	switch transport {
	case "chat_completions", "openai_chat":
		return DefaultTransport
	default:
		return transport
	}
}

func manifestBaseURL(provider string) string {
	entry, ok := hermes.ResolveProviderManifestEntry(provider)
	if !ok {
		return ""
	}
	return strings.TrimSpace(entry.BaseURLOverride)
}

func normalizeBaseURL(raw string) string {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func isLocalRoute(provider, baseURL string) bool {
	switch provider {
	case "local", "lmstudio":
		return true
	}
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isProviderControlledQuotaCandidate(route Route) bool {
	needle := strings.ToLower(strings.Join([]string{route.Name, route.Alias, route.Provider, route.Model}, " "))
	if strings.Contains(needle, ":free") || strings.Contains(needle, "free-tier") || strings.Contains(needle, "free_tier") {
		return true
	}
	return false
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
