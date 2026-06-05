package llm

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/oauth"
)

// NousOAuthCredentials is the local type for OAuth state, mirroring
// config.NousOAuthCredentials to avoid an import cycle.
type NousOAuthCredentials = oauth.NousOAuthCredentials

// CredentialPoolOptions is a local stub for config.CredentialPoolOptions.
type CredentialPoolOptions = oauth.CredentialPoolOptions

// NousOAuthLoginOptions controls the device-code login flow.
type NousOAuthLoginOptions = oauth.NousOAuthLoginOptions

// NousOAuthRefreshOptions controls refresh-token rotation.
type NousOAuthRefreshOptions = oauth.NousOAuthRefreshOptions

// NousOAuthMintOptions controls short-lived agent-key minting.
type NousOAuthMintOptions = oauth.NousOAuthMintOptions

// NousOAuthRuntimeOptions controls runtime credential resolution.
type NousOAuthRuntimeOptions = oauth.NousOAuthRuntimeOptions

// NousRuntimeCredentials is the resolved inference credential ready
// for provider use.
type NousRuntimeCredentials = oauth.NousRuntimeCredentials

// NousAuthError is a classified OAuth error with actionable operator guidance.
type NousAuthError = oauth.NousAuthError

// NousOAuthDeviceCodeLogin runs the full Hermes-compatible Nous device-code
// OAuth flow: request device code, open browser for user approval, poll for
// token, then immediately mint a short-lived agent key. Returns full
// credentials suitable for SaveNousOAuthCredentials.
func NousOAuthDeviceCodeLogin(ctx context.Context, opts NousOAuthLoginOptions) (NousOAuthCredentials, error) {
	return oauth.NousOAuthDeviceCodeLogin(ctx, opts)
}

func RefreshNousAccessToken(ctx context.Context, opts NousOAuthRefreshOptions) (NousOAuthCredentials, error) {
	return oauth.RefreshNousAccessToken(ctx, opts)
}

func MintNousAgentKey(ctx context.Context, opts NousOAuthMintOptions) (NousOAuthCredentials, error) {
	return oauth.MintNousAgentKey(ctx, opts)
}

func ResolveNousRuntimeCredentials(ctx context.Context, creds NousOAuthCredentials, opts NousOAuthRuntimeOptions) (NousRuntimeCredentials, error) {
	return oauth.ResolveNousRuntimeCredentials(ctx, creds, opts)
}

func resolveNousRuntimeFromCreds(ctx context.Context, creds NousOAuthCredentials, opts NousOAuthRuntimeOptions) (NousRuntimeCredentials, error) {
	return oauth.ResolveNousRuntimeCredentials(ctx, creds, opts)
}
