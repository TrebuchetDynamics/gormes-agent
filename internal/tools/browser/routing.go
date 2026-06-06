package browser

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/browser/navigation"

const (
	defaultBrowserTaskID        = navigation.DefaultTaskID
	localBrowserSidecarSuffix   = navigation.LocalSidecarSuffix
	privateBrowserSidecarReason = navigation.PrivateSidecarReason
)

// BrowserRoute is the pure pre-navigation routing decision for a browser URL.
type BrowserRoute = navigation.Route

// IsPrivateBrowserHost reports whether host is a local, private, or LAN-style
// browser target that should stay off a cloud browser provider.
func IsPrivateBrowserHost(host string) bool {
	return navigation.IsPrivateBrowserHost(host)
}

// RouteBrowserNavigation selects the session key for an initial browser
// navigation without starting a browser or consulting runtime configuration.
func RouteBrowserNavigation(taskID, rawURL string, cloudConfigured, autoLocalForPrivateURLs, cdpOverride, camofoxMode bool) BrowserRoute {
	return navigation.RouteNavigation(taskID, rawURL, cloudConfigured, autoLocalForPrivateURLs, cdpOverride, camofoxMode)
}
