package mcp

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// MCPSession is the OAuth session material returned by an MCPLoginFlow. Token
// fields are secret material; callers must render MCPLoginResult instead of
// printing this struct.
type MCPSession struct {
	AccessToken  string
	RefreshToken string
	Scope        string
	Issuer       string
	ExpiresAt    time.Time
}

// MCPLoginFlow is the injectable login seam for `gormes mcp login <name>`.
// The default implementation is intentionally non-interactive: browser and
// localhost callback behavior belongs to the follow-up browser-flow row.
type MCPLoginFlow interface {
	Login(ctx context.Context, server MCPServerDefinition) (*MCPSession, error)
}

// MCPLoginEvidence is stable operator-facing evidence for MCP login outcomes.
type MCPLoginEvidence string

const (
	MCPLoginEvidenceSaved                  MCPLoginEvidence = "mcp_login_saved"
	MCPLoginEvidenceNoninteractiveRequired MCPLoginEvidence = "noninteractive_required"
	MCPLoginEvidenceStateStoreUnwritable   MCPLoginEvidence = "mcp_login_state_store_unwritable"
	MCPLoginEvidenceFlowFailed             MCPLoginEvidence = "mcp_login_flow_failed"
	MCPLoginEvidenceServerUnknown          MCPLoginEvidence = "mcp_server_unknown"
	MCPLoginEvidenceAuthNotOAuth           MCPLoginEvidence = "mcp_auth_not_oauth"
	MCPLoginEvidenceRedirectURIMismatch    MCPLoginEvidence = "mcp_login_redirect_uri_mismatch"
	MCPLoginEvidencePortCollision          MCPLoginEvidence = "mcp_login_port_collision"
	MCPLoginEvidenceCallbackTimeout        MCPLoginEvidence = "mcp_login_callback_timeout"
	MCPLoginEvidenceTokenExchangeFailed    MCPLoginEvidence = "mcp_login_token_exchange_failed"
	MCPLoginEvidenceBrowserUnavailable     MCPLoginEvidence = "mcp_login_browser_unavailable"
)

// MCPLoginResult is the safe, redacted output for the command and tests.
type MCPLoginResult struct {
	Server    string
	Evidence  MCPLoginEvidence
	Message   string
	Available []string
}

func (r MCPLoginResult) Error() string {
	parts := []string{"evidence=" + string(r.Evidence)}
	if strings.TrimSpace(r.Server) != "" {
		parts = append(parts, "server="+r.Server)
	}
	if msg := strings.TrimSpace(r.Message); msg != "" {
		parts = append(parts, msg)
	}
	if len(r.Available) > 0 {
		parts = append(parts, "available="+strings.Join(r.Available, ","))
	}
	return strings.Join(parts, " ")
}

// NoninteractiveLoginFlow returns typed guidance without opening browsers,
// sockets, callback servers, or live token exchange requests.
func NoninteractiveLoginFlow() MCPLoginFlow { return noninteractiveLoginFlow{} }

type noninteractiveLoginFlow struct{}

func (noninteractiveLoginFlow) Login(context.Context, MCPServerDefinition) (*MCPSession, error) {
	return nil, MCPLoginResult{
		Evidence: MCPLoginEvidenceNoninteractiveRequired,
		Message:  "interactive MCP OAuth login is not available in this noninteractive build; use 'gormes mcp remove' + 'gormes mcp add' to replace credentials, or wait for the browser-flow row",
	}
}

// RunMCPLogin validates the target server, delegates OAuth login to flow, and
// persists only successful sessions into store. It is pure and testable: no
// network, browser, filesystem, or live MCP server is touched here.
func RunMCPLogin(ctx context.Context, resolution MCPConfigResolution, store *MCPOAuthStore, flow MCPLoginFlow, serverName string) (MCPLoginResult, error) {
	name := strings.TrimSpace(serverName)
	if name == "" {
		return MCPLoginResult{Evidence: MCPLoginEvidenceServerUnknown, Message: "server name required", Available: mcpServerNames(resolution)}, nil
	}
	server, ok := resolution.Server(name)
	if !ok {
		return MCPLoginResult{Server: name, Evidence: MCPLoginEvidenceServerUnknown, Message: "unknown MCP server", Available: mcpServerNames(resolution)}, nil
	}
	if !isMCPServerOAuth(server) {
		return MCPLoginResult{Server: server.Name, Evidence: MCPLoginEvidenceAuthNotOAuth, Message: "MCP login only supports OAuth servers; use 'gormes mcp remove' + 'gormes mcp add' to reconfigure non-OAuth servers"}, nil
	}
	if flow == nil {
		flow = NoninteractiveLoginFlow()
	}
	session, err := flow.Login(ctx, server)
	if err != nil {
		var typed MCPLoginResult
		if errors.As(err, &typed) {
			typed.Server = firstNonEmptyMCPLogin(typed.Server, server.Name)
			return typed, nil
		}
		return MCPLoginResult{Server: server.Name, Evidence: MCPLoginEvidenceFlowFailed, Message: sanitizeMCPLoginText(err.Error())}, nil
	}
	if session == nil {
		return MCPLoginResult{Server: server.Name, Evidence: MCPLoginEvidenceFlowFailed, Message: "login flow returned no session"}, nil
	}
	if err := store.Set(server.Name, MCPOAuthToken{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		Scope:        session.Scope,
		Issuer:       session.Issuer,
		ExpiresAt:    session.ExpiresAt,
	}); err != nil {
		return MCPLoginResult{Server: server.Name, Evidence: MCPLoginEvidenceStateStoreUnwritable, Message: sanitizeMCPLoginText(err.Error())}, nil
	}
	return MCPLoginResult{Server: server.Name, Evidence: MCPLoginEvidenceSaved, Message: "OAuth session saved"}, nil
}

func isMCPServerOAuth(server MCPServerDefinition) bool {
	if !server.Enabled || server.Transport != MCPTransportHTTP || strings.TrimSpace(server.URL) == "" {
		return false
	}
	return len(server.Headers) == 0
}

func mcpServerNames(resolution MCPConfigResolution) []string {
	seen := map[string]struct{}{}
	for _, server := range resolution.Servers {
		if server.Name != "" {
			seen[server.Name] = struct{}{}
		}
	}
	for _, status := range resolution.Statuses {
		if status.Name != "" {
			seen[status.Name] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func firstNonEmptyMCPLogin(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sanitizeMCPLoginText(text string) string {
	redacted := RedactString(text)
	if strings.Contains(strings.ToLower(redacted), "access_token") || strings.Contains(strings.ToLower(redacted), "refresh_token") {
		return "provider returned redacted credential error"
	}
	return fmt.Sprint(redacted)
}
