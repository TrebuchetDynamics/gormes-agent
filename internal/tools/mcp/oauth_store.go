package mcp

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/oauth"

// MCPOAuthState labels the operator-visible state of an MCP OAuth token slot
// without leaking secret material. Values are stable strings consumed by status
// surfaces and degraded-mode reporting.
type MCPOAuthState = oauth.State

const (
	MCPOAuthStateAbsent                 MCPOAuthState = oauth.StateAbsent
	MCPOAuthStateValid                  MCPOAuthState = oauth.StateValid
	MCPOAuthStateExpired                MCPOAuthState = oauth.StateExpired
	MCPOAuthStateNoninteractiveRequired MCPOAuthState = oauth.StateNoninteractiveRequired
)

// ErrMCPOAuthNoninteractiveRequired is returned by callers when the store is
// configured for non-interactive mode and a token is missing or otherwise
// unrecoverable without user interaction.
var ErrMCPOAuthNoninteractiveRequired = oauth.ErrNoninteractiveRequired

// MCPOAuthToken is the in-memory credential record for a single MCP server.
// AccessToken and RefreshToken are secret material; the store boundary is
// responsible for keeping them out of any operator-visible output.
type MCPOAuthToken = oauth.Token

// MCPOAuthStatus is the redacted, operator-visible read of one server's OAuth
// state. Server, State, and Evidence are safe to log and render; no secret
// material ever appears in this struct.
type MCPOAuthStatus = oauth.Status

// MCPOAuthStore is a pure in-memory state store for MCP OAuth tokens. It does
// not persist to disk, contact OAuth issuers, or open transports; refresh and
// recovery are layered on top by other components.
type MCPOAuthStore = oauth.Store

// NewMCPOAuthStore returns an empty store in interactive mode.
func NewMCPOAuthStore() *MCPOAuthStore { return oauth.NewStore() }
