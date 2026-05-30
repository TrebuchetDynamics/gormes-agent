package tools

import (
	"context"
	"time"

	mcptools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp"
)

var ErrMCPOAuthSessionExpired = mcptools.ErrMCPOAuthSessionExpired

type MCPRefresher = mcptools.MCPRefresher

type MCPOAuthRefreshOutcome = mcptools.MCPOAuthRefreshOutcome

const (
	MCPOAuthRefreshOutcomeRefreshed              = mcptools.MCPOAuthRefreshOutcomeRefreshed
	MCPOAuthRefreshOutcomeStillValid             = mcptools.MCPOAuthRefreshOutcomeStillValid
	MCPOAuthRefreshOutcomeTokenCleared           = mcptools.MCPOAuthRefreshOutcomeTokenCleared
	MCPOAuthRefreshOutcomeNoninteractiveRequired = mcptools.MCPOAuthRefreshOutcomeNoninteractiveRequired
	MCPOAuthRefreshOutcomeRefresherUnavailable   = mcptools.MCPOAuthRefreshOutcomeRefresherUnavailable
)

type MCPOAuthRefreshResult = mcptools.MCPOAuthRefreshResult

func RefreshMCPOAuth(ctx context.Context, store *MCPOAuthStore, server string, refresher MCPRefresher, now time.Time) (MCPOAuthRefreshResult, error) {
	return mcptools.RefreshMCPOAuth(ctx, store, server, refresher, now)
}
