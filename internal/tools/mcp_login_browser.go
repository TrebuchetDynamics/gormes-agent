package tools

import mcptools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp"

type MCPBrowserLoginOptions = mcptools.MCPBrowserLoginOptions
type BrowserMCPLoginFlow = mcptools.BrowserMCPLoginFlow

func NewBrowserMCPLoginFlow(opts MCPBrowserLoginOptions) *BrowserMCPLoginFlow {
	return mcptools.NewBrowserMCPLoginFlow(opts)
}
