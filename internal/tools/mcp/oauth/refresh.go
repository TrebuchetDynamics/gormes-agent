package oauth

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrSessionExpired marks a 401-equivalent refresh failure that means the
// stored OAuth token can no longer recover the MCP session.
var ErrSessionExpired = errors.New("mcp oauth: session expired")

// Refresher refreshes an MCP OAuth token without opening an interactive login
// flow.
type Refresher interface {
	Refresh(ctx context.Context, refreshToken string) (newTokens Token, err error)
}

// RefreshOutcome is the degraded-mode outcome of Refresh.
type RefreshOutcome string

const (
	RefreshOutcomeRefreshed              RefreshOutcome = "refreshed"
	RefreshOutcomeStillValid             RefreshOutcome = "still_valid"
	RefreshOutcomeTokenCleared           RefreshOutcome = "token_cleared"
	RefreshOutcomeNoninteractiveRequired RefreshOutcome = "noninteractive_required"
	RefreshOutcomeRefresherUnavailable   RefreshOutcome = "refresher_unavailable"
)

// RefreshResult reports the recovery decision without exposing token material.
type RefreshResult struct {
	Server  string
	Outcome RefreshOutcome
}

// Refresh refreshes an expired OAuth token for server using refresher.
func Refresh(ctx context.Context, store *Store, server string, refresher Refresher, now time.Time) (RefreshResult, error) {
	result := RefreshResult{Server: server}

	tok, ok := store.Get(server)
	if !ok {
		result.Outcome = RefreshOutcomeNoninteractiveRequired
		return result, ErrNoninteractiveRequired
	}
	if tok.ExpiresAt.IsZero() || now.Before(tok.ExpiresAt) {
		result.Outcome = RefreshOutcomeStillValid
		return result, nil
	}
	if tok.RefreshToken == "" {
		result.Outcome = RefreshOutcomeNoninteractiveRequired
		return result, ErrNoninteractiveRequired
	}
	if refresher == nil {
		result.Outcome = RefreshOutcomeRefresherUnavailable
		return result, nil
	}

	newToken, err := refresher.Refresh(ctx, tok.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrSessionExpired) {
			store.Clear(server)
			result.Outcome = RefreshOutcomeTokenCleared
			return result, nil
		}
		result.Outcome = RefreshOutcomeRefresherUnavailable
		return result, fmt.Errorf("mcp oauth refresh %q: %w", server, err)
	}
	if err := store.Set(server, newToken); err != nil {
		return result, err
	}
	result.Outcome = RefreshOutcomeRefreshed
	return result, nil
}
