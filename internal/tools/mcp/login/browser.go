package login

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/login/browser"
)

// MCPBrowserLoginOptions makes the browser OAuth flow hermetic in tests. The
// default flow binds only to 127.0.0.1, but callers still inject BrowserOpen and
// HTTPClient so tests never open real browsers or contact live token endpoints.
type BrowserOptions = browser.Options

// BrowserMCPLoginFlow implements MCPLoginFlow with a localhost callback and an
// OAuth authorization-code token exchange. It is provider-neutral and stores no
// tokens itself; RunMCPLogin persists successful sessions.
type BrowserFlow = browser.Flow

func NewBrowserFlow(opts BrowserOptions) *BrowserFlow {
	return browser.NewFlow(opts)
}
