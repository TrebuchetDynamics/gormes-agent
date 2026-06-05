package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/browser"

const (
	defaultBrowserTaskID        = "default"
	localBrowserSidecarSuffix   = "::local"
	privateBrowserSidecarReason = "private_url_local_sidecar"
)

// BrowserRoute is the pure pre-navigation routing decision for a browser URL.
type BrowserRoute = browser.BrowserRoute

// IsPrivateBrowserHost reports whether host is a local, private, or LAN-style
// browser target that should stay off a cloud browser provider.
func IsPrivateBrowserHost(host string) bool {
	return browser.IsPrivateBrowserHost(host)
}

// RouteBrowserNavigation selects the session key for an initial browser
// navigation without starting a browser or consulting runtime configuration.
func RouteBrowserNavigation(taskID, rawURL string, cloudConfigured, autoLocalForPrivateURLs, cdpOverride, camofoxMode bool) BrowserRoute {
	return browser.RouteBrowserNavigation(taskID, rawURL, cloudConfigured, autoLocalForPrivateURLs, cdpOverride, camofoxMode)
}

func normalizeBrowserTaskID(taskID string) string {
	if taskID == "" {
		return defaultBrowserTaskID
	}
	return taskID
}
