package gormescli

import (
	"context"

	appauthcodex "github.com/TrebuchetDynamics/gormes-agent/internal/app/authcodex"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const (
	CodexOAuthIssuer   = appauthcodex.OAuthIssuer
	CodexOAuthTokenURL = appauthcodex.OAuthTokenURL
	CodexOAuthClientID = appauthcodex.OAuthClientID
	CodexOAuthBaseURL  = appauthcodex.OAuthBaseURL
)

type CodexOAuthLoginRequest = appauthcodex.LoginRequest

type CodexDeviceCode = appauthcodex.DeviceCode

type CodexAuthorizationCode = appauthcodex.AuthorizationCode

type CodexTokenResponse = appauthcodex.TokenResponse

func RunCodexDeviceCodeLogin(ctx context.Context, req CodexOAuthLoginRequest) (config.CodexOAuthTokens, error) {
	return appauthcodex.RunDeviceCodeLogin(ctx, req)
}

func SanitizeAuthCommandError(input string) string { return appauthcodex.SanitizeCommandError(input) }
