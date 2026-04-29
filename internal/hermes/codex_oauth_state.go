package hermes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

const (
	CodexOAuthRefreshOK              = "codex_refresh_ok"
	CodexOAuthRefreshRetryable       = "codex_refresh_retryable"
	CodexOAuthRefreshFailed          = "codex_refresh_failed"
	CodexOAuthRefreshReloginRequired = "codex_refresh_relogin_required"
)

type CodexOAuthRefreshStatus struct {
	Provider        string `json:"provider"`
	Code            string `json:"code"`
	Status          int    `json:"status,omitempty"`
	Message         string `json:"message,omitempty"`
	ReloginRequired bool   `json:"relogin_required"`
	Retryable       bool   `json:"retryable"`
	Redacted        bool   `json:"redacted"`
}

func ClassifyCodexOAuthRefreshFailure(err error) CodexOAuthRefreshStatus {
	status := CodexOAuthRefreshStatus{
		Provider: "openai-codex",
		Code:     CodexOAuthRefreshOK,
		Redacted: true,
	}
	if err == nil {
		return status
	}
	status.Code = CodexOAuthRefreshFailed
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
	if codexOAuthRefreshCodeRequiresRelogin(providerCode) {
		status.Code = providerCode
		status.ReloginRequired = true
		status.Retryable = false
		return status
	}
	if httpErr.Status == http.StatusUnauthorized || httpErr.Status == http.StatusForbidden {
		status.Code = CodexOAuthRefreshReloginRequired
		status.ReloginRequired = true
		status.Retryable = false
		return status
	}
	if httpErr.Status == http.StatusTooManyRequests || httpErr.Status >= 500 {
		status.Code = CodexOAuthRefreshRetryable
		status.Retryable = true
		return status
	}
	if providerCode != "" {
		status.Code = providerCode
	}
	return status
}

func codexOAuthRefreshCodeRequiresRelogin(code string) bool {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "invalid_grant", "invalid_token", "invalid_request", "refresh_token_reused":
		return true
	default:
		return false
	}
}

func codexOAuthErrorBody(body string) (code, message string) {
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
