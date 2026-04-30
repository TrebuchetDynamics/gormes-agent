package hermes

import (
	"errors"
	"net/http"
	"strings"
)

const (
	AnthropicOAuthRefreshOK              = "anthropic_refresh_ok"
	AnthropicOAuthRefreshRetryable       = "anthropic_refresh_retryable"
	AnthropicOAuthRefreshFailed          = "anthropic_refresh_failed"
	AnthropicOAuthRefreshReloginRequired = "anthropic_refresh_relogin_required"
)

type AnthropicOAuthRefreshStatus struct {
	Provider        string `json:"provider"`
	Code            string `json:"code"`
	Status          int    `json:"status,omitempty"`
	Message         string `json:"message,omitempty"`
	ReloginRequired bool   `json:"relogin_required"`
	Retryable       bool   `json:"retryable"`
	Redacted        bool   `json:"redacted"`
}

func ClassifyAnthropicOAuthRefreshFailure(err error) AnthropicOAuthRefreshStatus {
	status := AnthropicOAuthRefreshStatus{
		Provider: "anthropic",
		Code:     AnthropicOAuthRefreshOK,
		Redacted: true,
	}
	if err == nil {
		return status
	}
	status.Code = AnthropicOAuthRefreshFailed
	status.Message = err.Error()

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		return status
	}
	status.Status = httpErr.Status
	message, _, providerCode := providerHTTPErrorText(httpErr)
	if message != "" {
		status.Message = message
	}
	if oauthCode, oauthMessage := codexOAuthErrorBody(httpErr.Body); oauthCode != "" {
		providerCode = oauthCode
		if oauthMessage != "" {
			status.Message = oauthMessage
		}
	}
	providerCode = strings.TrimSpace(providerCode)
	if anthropicOAuthRefreshCodeRequiresRelogin(providerCode) {
		status.Code = providerCode
		status.ReloginRequired = true
		status.Retryable = false
		return status
	}
	if httpErr.Status == http.StatusUnauthorized || httpErr.Status == http.StatusForbidden {
		status.Code = AnthropicOAuthRefreshReloginRequired
		status.ReloginRequired = true
		status.Retryable = false
		return status
	}
	if httpErr.Status == http.StatusTooManyRequests || httpErr.Status >= 500 {
		status.Code = AnthropicOAuthRefreshRetryable
		status.Retryable = true
		return status
	}
	if providerCode != "" {
		status.Code = providerCode
	}
	return status
}

func anthropicOAuthRefreshCodeRequiresRelogin(code string) bool {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "invalid_grant", "invalid_token", "invalid_request", "refresh_token_reused":
		return true
	default:
		return false
	}
}
