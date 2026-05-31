package mcp

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/login"
)

type MCPSession = login.Session

type MCPLoginFlow = login.Flow

type MCPLoginEvidence = login.Evidence

const (
	MCPLoginEvidenceSaved                  = login.EvidenceSaved
	MCPLoginEvidenceNoninteractiveRequired = login.EvidenceNoninteractiveRequired
	MCPLoginEvidenceStateStoreUnwritable   = login.EvidenceStateStoreUnwritable
	MCPLoginEvidenceFlowFailed             = login.EvidenceFlowFailed
	MCPLoginEvidenceServerUnknown          = login.EvidenceServerUnknown
	MCPLoginEvidenceAuthNotOAuth           = login.EvidenceAuthNotOAuth
	MCPLoginEvidenceRedirectURIMismatch    = login.EvidenceRedirectURIMismatch
	MCPLoginEvidencePortCollision          = login.EvidencePortCollision
	MCPLoginEvidenceCallbackTimeout        = login.EvidenceCallbackTimeout
	MCPLoginEvidenceTokenExchangeFailed    = login.EvidenceTokenExchangeFailed
	MCPLoginEvidenceBrowserUnavailable     = login.EvidenceBrowserUnavailable
)

type MCPLoginResult = login.Result

func NoninteractiveLoginFlow() MCPLoginFlow { return login.NoninteractiveFlow() }

func RunMCPLogin(ctx context.Context, resolution MCPConfigResolution, store *MCPOAuthStore, flow MCPLoginFlow, serverName string) (MCPLoginResult, error) {
	return login.Run(ctx, resolution, store, flow, serverName)
}

func firstNonEmptyMCPLogin(values ...string) string { return login.FirstNonEmpty(values...) }

func sanitizeMCPLoginText(text string) string { return login.SanitizeText(text) }
