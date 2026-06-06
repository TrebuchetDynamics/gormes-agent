package navigation

import (
	"net/netip"
	"net/url"
	"strings"
)

const (
	DefaultTaskID        = "default"
	LocalSidecarSuffix   = "::local"
	PrivateSidecarReason = "private_url_local_sidecar"
)

// Route is the pure pre-navigation routing decision for a browser URL.
type Route struct {
	SessionKey string
	ForceLocal bool
	Reason     string
}

// IsPrivateBrowserHost reports whether host is a local, private, or LAN-style
// browser target that should stay off a cloud browser provider.
func IsPrivateBrowserHost(host string) bool {
	candidate := classifyBrowserHost(host)
	if candidate.localName {
		return true
	}
	if !candidate.hasAddr {
		return false
	}
	return isPrivateBrowserAddr(candidate.addr)
}

type browserHostCandidate struct {
	hostname  string
	localName bool
	addr      netip.Addr
	hasAddr   bool
}

func classifyBrowserHost(host string) browserHostCandidate {
	hostname := normalizeBrowserHost(host)
	candidate := browserHostCandidate{hostname: hostname}
	if hostname == "" {
		return candidate
	}
	if hostname == "localhost" ||
		strings.HasSuffix(hostname, ".local") ||
		strings.HasSuffix(hostname, ".lan") ||
		strings.HasSuffix(hostname, ".internal") {
		candidate.localName = true
		return candidate
	}

	addr, err := netip.ParseAddr(hostname)
	if err != nil {
		return candidate
	}
	candidate.addr = addr
	candidate.hasAddr = true
	return candidate
}

func isPrivateBrowserAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	return addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast()
}

// RouteNavigation selects the session key for an initial browser
// navigation without starting a browser or consulting runtime configuration.
func RouteNavigation(taskID, rawURL string, cloudConfigured, autoLocalForPrivateURLs, cdpOverride, camofoxMode bool) Route {
	sessionKey := NormalizeTaskID(taskID)
	route := Route{SessionKey: sessionKey}
	if !cloudConfigured || !autoLocalForPrivateURLs || cdpOverride || camofoxMode {
		return route
	}

	if !IsPrivateNavigationTarget(rawURL) {
		return route
	}

	return Route{
		SessionKey: sessionKey + LocalSidecarSuffix,
		ForceLocal: true,
		Reason:     PrivateSidecarReason,
	}
}

func browserNavigationHostname(rawURL string) (string, bool) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", false
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Hostname() != "" {
		return parsed.Hostname(), true
	}
	if strings.Contains(trimmed, "://") || strings.HasPrefix(trimmed, "/") {
		return "", false
	}
	parsed, err := url.Parse("//" + trimmed)
	if err != nil || parsed.Hostname() == "" {
		return "", false
	}
	return parsed.Hostname(), true
}

func IsPrivateNavigationTarget(rawURL string) bool {
	hostname, ok := browserNavigationHostname(rawURL)
	return ok && IsPrivateBrowserHost(hostname)
}

func NormalizeTaskID(taskID string) string {
	if taskID == "" {
		return DefaultTaskID
	}
	return taskID
}

func normalizeBrowserHost(host string) string {
	hostname := strings.TrimSpace(host)
	if strings.HasPrefix(hostname, "[") {
		if closeBracket := strings.Index(hostname, "]"); closeBracket > 0 {
			hostname = hostname[1:closeBracket]
		}
	} else if strings.Count(hostname, ":") == 1 {
		if colon := strings.LastIndex(hostname, ":"); colon > 0 {
			hostname = hostname[:colon]
		}
	}
	hostname = strings.ToLower(hostname)
	return strings.TrimSuffix(hostname, ".")
}
