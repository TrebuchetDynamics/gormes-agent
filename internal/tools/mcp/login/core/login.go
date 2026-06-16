package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/oauth"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/redaction"
)

// Session is the OAuth session material returned by a login Flow. Token fields
// are secret material; callers must render Result instead of printing this
// struct.
type Session struct {
	AccessToken  string
	RefreshToken string
	Scope        string
	Issuer       string
	ExpiresAt    time.Time
}

// Flow is the injectable login seam for `gormes mcp login <name>`.
type Flow interface {
	Login(ctx context.Context, server config.MCPServerDefinition) (*Session, error)
}

// Evidence is stable operator-facing evidence for MCP login outcomes.
type Evidence string

const (
	EvidenceSaved                  Evidence = "mcp_login_saved"
	EvidenceNoninteractiveRequired Evidence = "noninteractive_required"
	EvidenceStateStoreUnwritable   Evidence = "mcp_login_state_store_unwritable"
	EvidenceFlowFailed             Evidence = "mcp_login_flow_failed"
	EvidenceServerUnknown          Evidence = "mcp_server_unknown"
	EvidenceAuthNotOAuth           Evidence = "mcp_auth_not_oauth"
	EvidenceRedirectURIMismatch    Evidence = "mcp_login_redirect_uri_mismatch"
	EvidencePortCollision          Evidence = "mcp_login_port_collision"
	EvidenceCallbackTimeout        Evidence = "mcp_login_callback_timeout"
	EvidenceTokenExchangeFailed    Evidence = "mcp_login_token_exchange_failed"
	EvidenceBrowserUnavailable     Evidence = "mcp_login_browser_unavailable"
)

// Result is the safe, redacted output for the command and tests.
type Result struct {
	Server    string
	Evidence  Evidence
	Message   string
	Available []string
}

func (r Result) Error() string {
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

// NoninteractiveFlow returns typed guidance without opening browsers, sockets,
// callback servers, or live token exchange requests.
func NoninteractiveFlow() Flow { return noninteractiveLoginFlow{} }

type noninteractiveLoginFlow struct{}

func (noninteractiveLoginFlow) Login(context.Context, config.MCPServerDefinition) (*Session, error) {
	return nil, Result{
		Evidence: EvidenceNoninteractiveRequired,
		Message:  "interactive MCP OAuth login is not available in this noninteractive build; use 'gormes mcp remove' + 'gormes mcp add' to replace credentials, or wait for the browser-flow row",
	}
}

// Run validates the target server, delegates OAuth login to flow, and persists
// only successful sessions into store.
func Run(ctx context.Context, resolution config.MCPConfigResolution, store *oauth.Store, flow Flow, serverName string) (Result, error) {
	name := strings.TrimSpace(serverName)
	if name == "" {
		return Result{Evidence: EvidenceServerUnknown, Message: "server name required", Available: serverNames(resolution)}, nil
	}
	server, ok := resolution.Server(name)
	if !ok {
		return Result{Server: name, Evidence: EvidenceServerUnknown, Message: "unknown MCP server", Available: serverNames(resolution)}, nil
	}
	if !isServerOAuth(server) {
		return Result{Server: server.Name, Evidence: EvidenceAuthNotOAuth, Message: "MCP login only supports OAuth servers; use 'gormes mcp remove' + 'gormes mcp add' to reconfigure non-OAuth servers"}, nil
	}
	if flow == nil {
		flow = NoninteractiveFlow()
	}
	session, err := flow.Login(ctx, server)
	if err != nil {
		var typed Result
		if errors.As(err, &typed) {
			typed.Server = FirstNonEmpty(typed.Server, server.Name)
			return typed, nil
		}
		return Result{Server: server.Name, Evidence: EvidenceFlowFailed, Message: sanitizeText(err.Error())}, nil
	}
	if session == nil {
		return Result{Server: server.Name, Evidence: EvidenceFlowFailed, Message: "login flow returned no session"}, nil
	}
	if err := store.Set(server.Name, oauth.Token{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		Scope:        session.Scope,
		Issuer:       session.Issuer,
		ExpiresAt:    session.ExpiresAt,
	}); err != nil {
		return Result{Server: server.Name, Evidence: EvidenceStateStoreUnwritable, Message: sanitizeText(err.Error())}, nil
	}
	return Result{Server: server.Name, Evidence: EvidenceSaved, Message: "OAuth session saved"}, nil
}

func isServerOAuth(server config.MCPServerDefinition) bool {
	if !server.Enabled || server.Transport != config.MCPTransportHTTP || strings.TrimSpace(server.URL) == "" {
		return false
	}
	return len(server.Headers) == 0
}

func serverNames(resolution config.MCPConfigResolution) []string {
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

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func SanitizeText(text string) string {
	redacted := redaction.String(text)
	if strings.Contains(strings.ToLower(redacted), "access_token") || strings.Contains(strings.ToLower(redacted), "refresh_token") {
		return "provider returned redacted credential error"
	}
	return fmt.Sprint(redacted)
}

func sanitizeText(text string) string { return SanitizeText(text) }
