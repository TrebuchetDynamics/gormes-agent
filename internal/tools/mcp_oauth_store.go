package tools

import mcptools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp"

type MCPOAuthState = mcptools.MCPOAuthState

const (
	MCPOAuthStateAbsent                 = mcptools.MCPOAuthStateAbsent
	MCPOAuthStateValid                  = mcptools.MCPOAuthStateValid
	MCPOAuthStateExpired                = mcptools.MCPOAuthStateExpired
	MCPOAuthStateNoninteractiveRequired = mcptools.MCPOAuthStateNoninteractiveRequired
)

var ErrMCPOAuthNoninteractiveRequired = mcptools.ErrMCPOAuthNoninteractiveRequired

type MCPOAuthToken = mcptools.MCPOAuthToken
type MCPOAuthStatus = mcptools.MCPOAuthStatus
type MCPOAuthStore = mcptools.MCPOAuthStore

func NewMCPOAuthStore() *MCPOAuthStore {
	return mcptools.NewMCPOAuthStore()
}
