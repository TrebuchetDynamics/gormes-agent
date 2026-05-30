package oauth

import (
	"encoding/json"
	"net/http"
	"strings"
)

const (
	AnthropicRefreshOK              = "anthropic_refresh_ok"
	AnthropicRefreshRetryable       = "anthropic_refresh_retryable"
	AnthropicRefreshFailed          = "anthropic_refresh_failed"
	AnthropicRefreshReloginRequired = "anthropic_refresh_relogin_required"

	CodexRefreshOK              = "codex_refresh_ok"
	CodexRefreshRetryable       = "codex_refresh_retryable"
	CodexRefreshFailed          = "codex_refresh_failed"
	CodexRefreshReloginRequired = "codex_refresh_relogin_required"
)

type RefreshStatus struct {
	Provider        string `json:"provider"`
	Code            string `json:"code"`
	Status          int    `json:"status,omitempty"`
	Message         string `json:"message,omitempty"`
	ReloginRequired bool   `json:"relogin_required"`
	Retryable       bool   `json:"retryable"`
	Redacted        bool   `json:"redacted"`
}

type RefreshFailureInput struct {
	Provider             string
	OKCode               string
	FailedCode           string
	RetryableCode        string
	ReloginRequiredCode  string
	HTTP                 bool
	HTTPStatus           int
	Message              string
	ProviderCode         string
	Body                 string
	RequiresReloginCodes map[string]struct{}
}

func ClassifyRefreshFailure(input RefreshFailureInput) RefreshStatus {
	status := RefreshStatus{
		Provider: input.Provider,
		Code:     input.OKCode,
		Redacted: true,
	}
	if input.Message == "" && !input.HTTP {
		return status
	}
	status.Code = input.FailedCode
	status.Message = input.Message
	if !input.HTTP {
		return status
	}
	status.Status = input.HTTPStatus
	providerCode := strings.TrimSpace(input.ProviderCode)
	if oauthCode, oauthMessage := OAuthErrorBody(input.Body); oauthCode != "" {
		providerCode = oauthCode
		if oauthMessage != "" {
			status.Message = oauthMessage
		}
	}
	if codeRequiresRelogin(providerCode, input.RequiresReloginCodes) {
		status.Code = providerCode
		status.ReloginRequired = true
		status.Retryable = false
		return status
	}
	if input.HTTPStatus == http.StatusUnauthorized || input.HTTPStatus == http.StatusForbidden {
		status.Code = input.ReloginRequiredCode
		status.ReloginRequired = true
		status.Retryable = false
		return status
	}
	if input.HTTPStatus == http.StatusTooManyRequests || input.HTTPStatus >= 500 {
		status.Code = input.RetryableCode
		status.Retryable = true
		return status
	}
	if providerCode != "" {
		status.Code = providerCode
	}
	return status
}

func CodexReloginCodes() map[string]struct{} {
	return refreshReloginCodes()
}

func AnthropicReloginCodes() map[string]struct{} {
	return refreshReloginCodes()
}

func refreshReloginCodes() map[string]struct{} {
	return map[string]struct{}{
		"invalid_grant":        {},
		"invalid_token":        {},
		"invalid_request":      {},
		"refresh_token_reused": {},
	}
}

func codeRequiresRelogin(code string, reloginCodes map[string]struct{}) bool {
	_, ok := reloginCodes[strings.ToLower(strings.TrimSpace(code))]
	return ok
}

func OAuthErrorBody(body string) (code, message string) {
	var decoded map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(body)), &decoded) != nil {
		return "", ""
	}
	if errValue, ok := decoded["error"].(string); ok {
		code = strings.TrimSpace(errValue)
	}
	if desc, ok := decoded["error_description"].(string); ok {
		message = strings.TrimSpace(desc)
	}
	if msg, ok := decoded["message"].(string); ok && message == "" {
		message = strings.TrimSpace(msg)
	}
	return code, message
}
