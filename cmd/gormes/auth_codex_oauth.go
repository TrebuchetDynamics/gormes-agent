package main

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

const (
	codexOAuthIssuer   = gormescli.CodexOAuthIssuer
	codexOAuthTokenURL = gormescli.CodexOAuthTokenURL
	codexOAuthClientID = gormescli.CodexOAuthClientID
	codexOAuthBaseURL  = gormescli.CodexOAuthBaseURL
)

var authCodexOAuthLogin = runCodexDeviceCodeLogin

type codexOAuthLoginRequest = gormescli.CodexOAuthLoginRequest

type codexDeviceCode = gormescli.CodexDeviceCode

type codexAuthorizationCode = gormescli.CodexAuthorizationCode

type codexTokenResponse = gormescli.CodexTokenResponse

func runCodexDeviceCodeLogin(ctx context.Context, req codexOAuthLoginRequest) (config.CodexOAuthTokens, error) {
	return gormescli.RunCodexDeviceCodeLogin(ctx, req)
}

func sanitizeAuthCommandError(input string) string { return gormescli.SanitizeAuthCommandError(input) }
