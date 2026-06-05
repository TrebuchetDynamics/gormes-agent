package cli

import (
	"context"
	"io"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/providerstatus"
)

const (
	AuthStatusLoggedIn  = providerstatus.AuthStatusLoggedIn
	AuthStatusLoggedOut = providerstatus.AuthStatusLoggedOut
	AuthStatusError     = providerstatus.AuthStatusError
)

type AuthStatusOptions = providerstatus.AuthStatusOptions
type ProviderAuthStatus = providerstatus.ProviderAuthStatus

func RenderAuthStatus(ctx context.Context, out io.Writer, providerInput string, opts AuthStatusOptions) (ProviderAuthStatus, error) {
	return providerstatus.RenderAuthStatus(ctx, out, providerInput, opts)
}

func ResolveAuthStatus(ctx context.Context, providerInput string, opts AuthStatusOptions) (ProviderAuthStatus, error) {
	return providerstatus.ResolveAuthStatus(ctx, providerInput, opts)
}
