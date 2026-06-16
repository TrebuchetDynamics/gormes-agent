package login

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/login/core"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/oauth"
)

// MCPSession is the OAuth session material returned by an MCPLoginFlow. Token
// fields are secret material; callers must render MCPLoginResult instead of
// printing this struct.
type Session = core.Session

// MCPLoginFlow is the injectable login seam for `gormes mcp login <name>`.
type Flow = core.Flow

// MCPLoginEvidence is stable operator-facing evidence for MCP login outcomes.
type Evidence = core.Evidence

const (
	EvidenceSaved                  = core.EvidenceSaved
	EvidenceNoninteractiveRequired = core.EvidenceNoninteractiveRequired
	EvidenceStateStoreUnwritable   = core.EvidenceStateStoreUnwritable
	EvidenceFlowFailed             = core.EvidenceFlowFailed
	EvidenceServerUnknown          = core.EvidenceServerUnknown
	EvidenceAuthNotOAuth           = core.EvidenceAuthNotOAuth
	EvidenceRedirectURIMismatch    = core.EvidenceRedirectURIMismatch
	EvidencePortCollision          = core.EvidencePortCollision
	EvidenceCallbackTimeout        = core.EvidenceCallbackTimeout
	EvidenceTokenExchangeFailed    = core.EvidenceTokenExchangeFailed
	EvidenceBrowserUnavailable     = core.EvidenceBrowserUnavailable
)

// MCPLoginResult is the safe, redacted output for the command and tests.
type Result = core.Result

// NoninteractiveLoginFlow returns typed guidance without opening browsers,
// sockets, callback servers, or live token exchange requests.
func NoninteractiveFlow() Flow { return core.NoninteractiveFlow() }

// RunMCPLogin validates the target server, delegates OAuth login to flow, and
// persists only successful sessions into store. It is pure and testable: no
// network, browser, filesystem, or live MCP server is touched here.
func Run(ctx context.Context, resolution config.MCPConfigResolution, store *oauth.Store, flow Flow, serverName string) (Result, error) {
	return core.Run(ctx, resolution, store, flow, serverName)
}

func FirstNonEmpty(values ...string) string { return core.FirstNonEmpty(values...) }

func SanitizeText(text string) string { return core.SanitizeText(text) }

func sanitizeText(text string) string { return SanitizeText(text) }
