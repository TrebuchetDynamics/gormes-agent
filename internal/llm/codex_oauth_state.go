package llm

import (
	"errors"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/oauth"
)

const (
	CodexOAuthRefreshOK              = oauth.CodexRefreshOK
	CodexOAuthRefreshRetryable       = oauth.CodexRefreshRetryable
	CodexOAuthRefreshFailed          = oauth.CodexRefreshFailed
	CodexOAuthRefreshReloginRequired = oauth.CodexRefreshReloginRequired
)

type CodexOAuthRefreshStatus = oauth.RefreshStatus

func ClassifyCodexOAuthRefreshFailure(err error) CodexOAuthRefreshStatus {
	input := oauth.RefreshFailureInput{
		Provider:             "openai-codex",
		OKCode:               CodexOAuthRefreshOK,
		FailedCode:           CodexOAuthRefreshFailed,
		RetryableCode:        CodexOAuthRefreshRetryable,
		ReloginRequiredCode:  CodexOAuthRefreshReloginRequired,
		RequiresReloginCodes: oauth.CodexReloginCodes(),
	}
	if err == nil {
		return oauth.ClassifyRefreshFailure(input)
	}
	input.Message = err.Error()
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		input.HTTP = true
		input.HTTPStatus = httpErr.Status
		message, _, providerCode := providerHTTPErrorText(httpErr)
		if message != "" {
			input.Message = message
		}
		input.ProviderCode = providerCode
		input.Body = httpErr.Body
	}
	return oauth.ClassifyRefreshFailure(input)
}

func codexOAuthRefreshCodeRequiresRelogin(code string) bool {
	_, ok := oauth.CodexReloginCodes()[strings.ToLower(strings.TrimSpace(code))]
	return ok
}

func codexOAuthErrorBody(body string) (code, message string) {
	return oauth.OAuthErrorBody(body)
}
