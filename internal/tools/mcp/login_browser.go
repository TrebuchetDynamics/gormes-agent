package mcp

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/login"

type MCPBrowserLoginOptions = login.BrowserOptions

type BrowserMCPLoginFlow = login.BrowserFlow

func NewBrowserMCPLoginFlow(opts MCPBrowserLoginOptions) *BrowserMCPLoginFlow {
	return login.NewBrowserFlow(opts)
}
