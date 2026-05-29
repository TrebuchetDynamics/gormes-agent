package llm

import "testing"

func TestCodexOAuthRefreshFailureRequiresReloginForStaleTokens(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{
			name: "unauthorized",
			err:  &HTTPError{Status: 401, Body: `{"error":{"message":"refresh token expired"}}`},
			code: CodexOAuthRefreshReloginRequired,
		},
		{
			name: "forbidden",
			err:  &HTTPError{Status: 403, Body: `{"error":{"message":"access denied"}}`},
			code: CodexOAuthRefreshReloginRequired,
		},
		{
			name: "invalid grant",
			err:  &HTTPError{Status: 400, Body: `{"error":"invalid_grant","error_description":"expired refresh token"}`},
			code: "invalid_grant",
		},
		{
			name: "refresh token reused",
			err:  &HTTPError{Status: 400, Body: `{"error":{"code":"refresh_token_reused","message":"token already used"}}`},
			code: "refresh_token_reused",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyCodexOAuthRefreshFailure(tc.err)
			if !got.ReloginRequired {
				t.Fatalf("ReloginRequired = false, want true: %#v", got)
			}
			if got.Retryable {
				t.Fatalf("Retryable = true, want stale token failures to stop retry: %#v", got)
			}
			if got.Code != tc.code {
				t.Fatalf("Code = %q, want %q", got.Code, tc.code)
			}
			if got.Provider != "openai-codex" {
				t.Fatalf("Provider = %q", got.Provider)
			}
		})
	}
}

func TestCodexOAuthRefreshFailureClassifiesTransientFailuresAsRetryable(t *testing.T) {
	got := ClassifyCodexOAuthRefreshFailure(&HTTPError{Status: 503, Body: `{"error":{"message":"try later"}}`})
	if got.ReloginRequired {
		t.Fatalf("ReloginRequired = true, want false: %#v", got)
	}
	if !got.Retryable {
		t.Fatalf("Retryable = false, want true: %#v", got)
	}
	if got.Code != CodexOAuthRefreshRetryable {
		t.Fatalf("Code = %q, want %q", got.Code, CodexOAuthRefreshRetryable)
	}
}
