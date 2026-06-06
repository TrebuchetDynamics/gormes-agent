package oauth

import (
	"context"
	"time"

	refreshpkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/oauth/refresh"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/oauth/tokens"
)

// State labels the operator-visible state of an MCP OAuth token slot without
// leaking secret material.
type State = tokens.State

const (
	StateAbsent                 = tokens.StateAbsent
	StateValid                  = tokens.StateValid
	StateExpired                = tokens.StateExpired
	StateNoninteractiveRequired = tokens.StateNoninteractiveRequired
)

// ErrNoninteractiveRequired is returned when a token cannot be recovered
// without user interaction.
var ErrNoninteractiveRequired = tokens.ErrNoninteractiveRequired

// Token is the in-memory credential record for a single MCP server.
type Token = tokens.Token

// Status is the redacted, operator-visible read of one server's OAuth state.
type Status = tokens.Status

// Store is a pure in-memory state store for MCP OAuth tokens.
type Store = tokens.Store

// NewStore returns an empty store in interactive mode.
func NewStore() *Store {
	return tokens.NewStore()
}

// ErrSessionExpired marks a 401-equivalent refresh failure that means the
// stored OAuth token can no longer recover the MCP session.
var ErrSessionExpired = refreshpkg.ErrSessionExpired

// ErrInvalidRefreshToken marks a refresh response that does not contain a
// usable access token.
var ErrInvalidRefreshToken = refreshpkg.ErrInvalidRefreshToken

// Refresher refreshes an MCP OAuth token without opening an interactive login
// flow.
type Refresher = refreshpkg.Refresher

// RefreshOutcome is the degraded-mode outcome of Refresh.
type RefreshOutcome = refreshpkg.RefreshOutcome

const (
	RefreshOutcomeRefreshed              = refreshpkg.RefreshOutcomeRefreshed
	RefreshOutcomeStillValid             = refreshpkg.RefreshOutcomeStillValid
	RefreshOutcomeTokenCleared           = refreshpkg.RefreshOutcomeTokenCleared
	RefreshOutcomeNoninteractiveRequired = refreshpkg.RefreshOutcomeNoninteractiveRequired
	RefreshOutcomeRefresherUnavailable   = refreshpkg.RefreshOutcomeRefresherUnavailable
)

// RefreshResult reports the recovery decision without exposing token material.
type RefreshResult = refreshpkg.RefreshResult

// Refresh refreshes an expired OAuth token for server using refresher.
func Refresh(ctx context.Context, store *Store, server string, refresher Refresher, now time.Time) (RefreshResult, error) {
	return refreshpkg.Refresh(ctx, store, server, refresher, now)
}
