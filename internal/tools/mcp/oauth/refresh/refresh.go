package refresh

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/oauth/tokens"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/redaction"
)

// ErrSessionExpired marks a 401-equivalent refresh failure that means the
// stored OAuth token can no longer recover the MCP session.
var ErrSessionExpired = errors.New("mcp oauth: session expired")

// ErrInvalidRefreshToken marks a refresh response that does not contain a
// usable access token.
var ErrInvalidRefreshToken = errors.New("mcp oauth: refreshed token missing access token")

// Refresher refreshes an MCP OAuth token without opening an interactive login
// flow.
type Refresher interface {
	Refresh(ctx context.Context, refreshToken string) (newTokens tokens.Token, err error)
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

func mergeRefreshedToken(previous, refreshed tokens.Token) tokens.Token {
	refreshToken := strings.TrimSpace(refreshed.RefreshToken)
	if refreshToken == "" {
		refreshed.RefreshToken = previous.RefreshToken
	} else {
		refreshed.RefreshToken = refreshToken
	}
	return refreshed
}

type redactedRefreshError struct {
	err error
}

func (e redactedRefreshError) Error() string {
	return redaction.String(e.err.Error())
}

func (e redactedRefreshError) Unwrap() error {
	return e.err
}

func safeRefreshError(server string, err error) error {
	return fmt.Errorf("mcp oauth refresh %q: %w", server, redactedRefreshError{err: err})
}

// Refresh refreshes an expired OAuth token for server using refresher.
func Refresh(ctx context.Context, store *tokens.Store, server string, refresher Refresher, now time.Time) (RefreshResult, error) {
	result := RefreshResult{Server: server}

	tok, ok := store.Get(server)
	if !ok {
		result.Outcome = RefreshOutcomeNoninteractiveRequired
		return result, tokens.ErrNoninteractiveRequired
	}
	if strings.TrimSpace(tok.AccessToken) != "" && (tok.ExpiresAt.IsZero() || now.Before(tok.ExpiresAt)) {
		result.Outcome = RefreshOutcomeStillValid
		return result, nil
	}
	refreshToken := strings.TrimSpace(tok.RefreshToken)
	if refreshToken == "" {
		result.Outcome = RefreshOutcomeNoninteractiveRequired
		return result, tokens.ErrNoninteractiveRequired
	}
	tok.RefreshToken = refreshToken
	if refresher == nil {
		result.Outcome = RefreshOutcomeRefresherUnavailable
		return result, nil
	}

	newToken, err := refresher.Refresh(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, ErrSessionExpired) {
			store.Clear(server)
			result.Outcome = RefreshOutcomeTokenCleared
			return result, nil
		}
		result.Outcome = RefreshOutcomeRefresherUnavailable
		return result, safeRefreshError(server, err)
	}
	mergedToken := mergeRefreshedToken(tok, newToken)
	if strings.TrimSpace(mergedToken.AccessToken) == "" {
		result.Outcome = RefreshOutcomeRefresherUnavailable
		return result, safeRefreshError(server, ErrInvalidRefreshToken)
	}
	if err := store.Set(server, mergedToken); err != nil {
		return result, err
	}
	result.Outcome = RefreshOutcomeRefreshed
	return result, nil
}
