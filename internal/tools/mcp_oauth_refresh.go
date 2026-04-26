package tools

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrMCPOAuthSessionExpired marks a 401-equivalent refresh failure that means
// the stored OAuth token can no longer recover the MCP session.
var ErrMCPOAuthSessionExpired = errors.New("mcp oauth: session expired")

// MCPRefresher refreshes an MCP OAuth token without opening an interactive
// login flow.
type MCPRefresher interface {
	Refresh(ctx context.Context, refreshToken string) (newTokens MCPOAuthToken, err error)
}

// MCPOAuthRefreshOutcome is the degraded-mode outcome of RefreshMCPOAuth.
type MCPOAuthRefreshOutcome string

const (
	MCPOAuthRefreshOutcomeRefreshed              MCPOAuthRefreshOutcome = "refreshed"
	MCPOAuthRefreshOutcomeStillValid             MCPOAuthRefreshOutcome = "still_valid"
	MCPOAuthRefreshOutcomeTokenCleared           MCPOAuthRefreshOutcome = "token_cleared"
	MCPOAuthRefreshOutcomeNoninteractiveRequired MCPOAuthRefreshOutcome = "noninteractive_required"
	MCPOAuthRefreshOutcomeRefresherUnavailable   MCPOAuthRefreshOutcome = "refresher_unavailable"
)

// MCPOAuthRefreshResult reports the recovery decision without exposing token
// material or requiring callers to inspect transport-specific errors.
type MCPOAuthRefreshResult struct {
	Server  string
	Outcome MCPOAuthRefreshOutcome
}

// RefreshMCPOAuth refreshes an expired OAuth token for server using refresher.
// It is pure over the supplied in-memory store and never starts an interactive
// authorization flow.
func RefreshMCPOAuth(ctx context.Context, store *MCPOAuthStore, server string, refresher MCPRefresher, now time.Time) (MCPOAuthRefreshResult, error) {
	result := MCPOAuthRefreshResult{Server: server}

	tok, ok := store.Get(server)
	if !ok {
		result.Outcome = MCPOAuthRefreshOutcomeNoninteractiveRequired
		return result, ErrMCPOAuthNoninteractiveRequired
	}
	if tok.ExpiresAt.IsZero() || now.Before(tok.ExpiresAt) {
		result.Outcome = MCPOAuthRefreshOutcomeStillValid
		return result, nil
	}
	if tok.RefreshToken == "" {
		result.Outcome = MCPOAuthRefreshOutcomeNoninteractiveRequired
		return result, ErrMCPOAuthNoninteractiveRequired
	}
	if refresher == nil {
		result.Outcome = MCPOAuthRefreshOutcomeRefresherUnavailable
		return result, nil
	}

	newToken, err := refresher.Refresh(ctx, tok.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrMCPOAuthSessionExpired) {
			store.Clear(server)
			result.Outcome = MCPOAuthRefreshOutcomeTokenCleared
			return result, nil
		}
		result.Outcome = MCPOAuthRefreshOutcomeRefresherUnavailable
		return result, fmt.Errorf("mcp oauth refresh %q: %w", server, err)
	}
	if err := store.Set(server, newToken); err != nil {
		return result, err
	}
	result.Outcome = MCPOAuthRefreshOutcomeRefreshed
	return result, nil
}
