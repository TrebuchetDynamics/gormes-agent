package llm

import (
	"errors"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/oauth"
)

const (
	AnthropicOAuthRefreshOK              = oauth.AnthropicRefreshOK
	AnthropicOAuthRefreshRetryable       = oauth.AnthropicRefreshRetryable
	AnthropicOAuthRefreshFailed          = oauth.AnthropicRefreshFailed
	AnthropicOAuthRefreshReloginRequired = oauth.AnthropicRefreshReloginRequired
)

type AnthropicOAuthRefreshStatus = oauth.RefreshStatus

func ClassifyAnthropicOAuthRefreshFailure(err error) AnthropicOAuthRefreshStatus {
	input := oauth.RefreshFailureInput{
		Provider:             "anthropic",
		OKCode:               AnthropicOAuthRefreshOK,
		FailedCode:           AnthropicOAuthRefreshFailed,
		RetryableCode:        AnthropicOAuthRefreshRetryable,
		ReloginRequiredCode:  AnthropicOAuthRefreshReloginRequired,
		RequiresReloginCodes: oauth.AnthropicReloginCodes(),
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

func anthropicOAuthRefreshCodeRequiresRelogin(code string) bool {
	_, ok := oauth.AnthropicReloginCodes()[strings.ToLower(strings.TrimSpace(code))]
	return ok
}
