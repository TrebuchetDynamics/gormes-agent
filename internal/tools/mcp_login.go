package tools

import (
	"context"

	mcptools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp"
)

type MCPSession = mcptools.MCPSession
type MCPLoginFlow = mcptools.MCPLoginFlow
type MCPLoginEvidence = mcptools.MCPLoginEvidence

const (
	MCPLoginEvidenceSaved                  = mcptools.MCPLoginEvidenceSaved
	MCPLoginEvidenceNoninteractiveRequired = mcptools.MCPLoginEvidenceNoninteractiveRequired
	MCPLoginEvidenceStateStoreUnwritable   = mcptools.MCPLoginEvidenceStateStoreUnwritable
	MCPLoginEvidenceFlowFailed             = mcptools.MCPLoginEvidenceFlowFailed
	MCPLoginEvidenceServerUnknown          = mcptools.MCPLoginEvidenceServerUnknown
	MCPLoginEvidenceAuthNotOAuth           = mcptools.MCPLoginEvidenceAuthNotOAuth
	MCPLoginEvidenceRedirectURIMismatch    = mcptools.MCPLoginEvidenceRedirectURIMismatch
	MCPLoginEvidencePortCollision          = mcptools.MCPLoginEvidencePortCollision
	MCPLoginEvidenceCallbackTimeout        = mcptools.MCPLoginEvidenceCallbackTimeout
	MCPLoginEvidenceTokenExchangeFailed    = mcptools.MCPLoginEvidenceTokenExchangeFailed
	MCPLoginEvidenceBrowserUnavailable     = mcptools.MCPLoginEvidenceBrowserUnavailable
)

type MCPLoginResult = mcptools.MCPLoginResult

func NoninteractiveLoginFlow() MCPLoginFlow { return mcptools.NoninteractiveLoginFlow() }

func RunMCPLogin(ctx context.Context, resolution MCPConfigResolution, store *MCPOAuthStore, flow MCPLoginFlow, serverName string) (MCPLoginResult, error) {
	return mcptools.RunMCPLogin(ctx, resolution, store, flow, serverName)
}
