package oauth

import "testing"

func TestClassifyRefreshFailureRequiresReloginForStaleTokens(t *testing.T) {
	cases := []struct {
		name         string
		input        RefreshFailureInput
		wantCode     string
		wantProvider string
	}{
		{
			name: "codex unauthorized",
			input: refreshInputForTest("openai-codex", CodexRefreshOK, CodexRefreshFailed, CodexRefreshRetryable, CodexRefreshReloginRequired,
				401, "refresh token expired", "", `{"error":{"message":"refresh token expired"}}`, CodexReloginCodes()),
			wantCode:     CodexRefreshReloginRequired,
			wantProvider: "openai-codex",
		},
		{
			name: "anthropic invalid grant",
			input: refreshInputForTest("anthropic", AnthropicRefreshOK, AnthropicRefreshFailed, AnthropicRefreshRetryable, AnthropicRefreshReloginRequired,
				400, "expired refresh token", "invalid_grant", `{"error":"invalid_grant","error_description":"expired refresh token"}`, AnthropicReloginCodes()),
			wantCode:     "invalid_grant",
			wantProvider: "anthropic",
		},
		{
			name: "nested provider code",
			input: refreshInputForTest("openai-codex", CodexRefreshOK, CodexRefreshFailed, CodexRefreshRetryable, CodexRefreshReloginRequired,
				400, "token already used", "refresh_token_reused", `{"error":{"code":"refresh_token_reused","message":"token already used"}}`, CodexReloginCodes()),
			wantCode:     "refresh_token_reused",
			wantProvider: "openai-codex",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyRefreshFailure(tc.input)
			if !got.ReloginRequired {
				t.Fatalf("ReloginRequired = false, want true: %#v", got)
			}
			if got.Retryable {
				t.Fatalf("Retryable = true, want stale token failures to stop retry: %#v", got)
			}
			if got.Code != tc.wantCode {
				t.Fatalf("Code = %q, want %q", got.Code, tc.wantCode)
			}
			if got.Provider != tc.wantProvider {
				t.Fatalf("Provider = %q, want %q", got.Provider, tc.wantProvider)
			}
		})
	}
}

func TestClassifyRefreshFailureClassifiesTransientFailuresAsRetryable(t *testing.T) {
	got := ClassifyRefreshFailure(refreshInputForTest("anthropic", AnthropicRefreshOK, AnthropicRefreshFailed, AnthropicRefreshRetryable, AnthropicRefreshReloginRequired,
		503, "try later", "", `{"error":{"message":"try later"}}`, AnthropicReloginCodes()))
	if got.ReloginRequired {
		t.Fatalf("ReloginRequired = true, want false: %#v", got)
	}
	if !got.Retryable {
		t.Fatalf("Retryable = false, want true: %#v", got)
	}
	if got.Code != AnthropicRefreshRetryable {
		t.Fatalf("Code = %q, want %q", got.Code, AnthropicRefreshRetryable)
	}
}

func refreshInputForTest(provider, okCode, failedCode, retryableCode, reloginCode string, status int, message, providerCode, body string, relogin map[string]struct{}) RefreshFailureInput {
	return RefreshFailureInput{
		Provider:             provider,
		OKCode:               okCode,
		FailedCode:           failedCode,
		RetryableCode:        retryableCode,
		ReloginRequiredCode:  reloginCode,
		HTTP:                 true,
		HTTPStatus:           status,
		Message:              message,
		ProviderCode:         providerCode,
		Body:                 body,
		RequiresReloginCodes: relogin,
	}
}
