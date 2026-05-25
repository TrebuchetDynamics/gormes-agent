package router

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// RecursionDiagnostic records a router route whose upstream origin is the same
// local origin the Router itself listens on. Such a route would make Gormes call
// its own OpenAI-compatible endpoint instead of a real upstream provider.
type RecursionDiagnostic struct {
	RouteName string
	Alias     string
	BaseURL   string
	Evidence  string
}

func (d RecursionDiagnostic) Error() string {
	name := firstNonEmpty(d.Alias, d.RouteName, "route")
	baseURL := d.BaseURL
	if baseURL == "" {
		baseURL = "(unknown)"
	}
	evidence := strings.TrimSpace(d.Evidence)
	if evidence == "" {
		evidence = "router_recursion_detected"
	}
	return fmt.Sprintf("%s: route=%s base_url=%s", evidence, name, baseURL)
}

// ValidateNoRecursion fails when any configured route points back at the
// Router listen address. Diagnostics deliberately redact userinfo, query, and
// fragment data before they reach CLI output or status JSON.
func ValidateNoRecursion(cfg config.RouterCfg) error {
	if diag, ok := FirstRecursionDiagnostic(cfg); ok {
		return diag
	}
	return nil
}

func FirstRecursionDiagnostic(cfg config.RouterCfg) (RecursionDiagnostic, bool) {
	listen := routerListenOrigin(cfg.Listen)
	if listen == "" {
		listen = routerListenOrigin(DefaultListen)
	}
	if listen == "" {
		return RecursionDiagnostic{}, false
	}
	for _, route := range cfg.Routes {
		base := sanitizeRouteBaseURL(route.BaseURL)
		if base == "" {
			continue
		}
		if routeOrigin(base) == listen {
			return RecursionDiagnostic{
				RouteName: strings.TrimSpace(route.Name),
				Alias:     strings.TrimSpace(route.Alias),
				BaseURL:   base,
				Evidence:  "router_recursion_detected",
			}, true
		}
	}
	return RecursionDiagnostic{}, false
}

func recursionEvidence(listen, baseURL string) (string, bool) {
	cfg := config.RouterCfg{Listen: listen, Routes: []config.RouterRouteCfg{{BaseURL: baseURL}}}
	diag, ok := FirstRecursionDiagnostic(cfg)
	if !ok {
		return "", false
	}
	return diag.Evidence + ":" + diag.BaseURL, true
}

func routerListenOrigin(listen string) string {
	listen = strings.TrimSpace(listen)
	if listen == "" {
		return ""
	}
	if !strings.Contains(listen, "://") {
		listen = "http://" + listen
	}
	return routeOrigin(listen)
}

func routeOrigin(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return ""
	}
	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if port == "" {
		switch strings.ToLower(parsed.Scheme) {
		case "https":
			port = "443"
		case "http", "":
			port = "80"
		}
	}
	if isLoopbackHost(host) {
		host = "loopback"
	}
	return host + ":" + port
}

func isLoopbackHost(host string) bool {
	if host == "localhost" || host == "ip6-localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func sanitizeRouteBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return raw
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}
