package mcp

import (
	"context"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/oauth"
)

// ErrMCPOAuthSessionExpired marks a 401-equivalent refresh failure that means
// the stored OAuth token can no longer recover the MCP session.
var ErrMCPOAuthSessionExpired = oauth.ErrSessionExpired

// MCPRefresher refreshes an MCP OAuth token without opening an interactive
// login flow.
type MCPRefresher = oauth.Refresher

// MCPOAuthRefreshOutcome is the degraded-mode outcome of RefreshMCPOAuth.
type MCPOAuthRefreshOutcome = oauth.RefreshOutcome

const (
	MCPOAuthRefreshOutcomeRefreshed              MCPOAuthRefreshOutcome = oauth.RefreshOutcomeRefreshed
	MCPOAuthRefreshOutcomeStillValid             MCPOAuthRefreshOutcome = oauth.RefreshOutcomeStillValid
	MCPOAuthRefreshOutcomeTokenCleared           MCPOAuthRefreshOutcome = oauth.RefreshOutcomeTokenCleared
	MCPOAuthRefreshOutcomeNoninteractiveRequired MCPOAuthRefreshOutcome = oauth.RefreshOutcomeNoninteractiveRequired
	MCPOAuthRefreshOutcomeRefresherUnavailable   MCPOAuthRefreshOutcome = oauth.RefreshOutcomeRefresherUnavailable
)

// MCPOAuthRefreshResult reports the recovery decision without exposing token
// material or requiring callers to inspect transport-specific errors.
type MCPOAuthRefreshResult = oauth.RefreshResult

// RefreshMCPOAuth refreshes an expired OAuth token for server using refresher.
// It is pure over the supplied in-memory store and never starts an interactive
// authorization flow.
func RefreshMCPOAuth(ctx context.Context, store *MCPOAuthStore, server string, refresher MCPRefresher, now time.Time) (MCPOAuthRefreshResult, error) {
	return oauth.Refresh(ctx, store, server, refresher, now)
}
