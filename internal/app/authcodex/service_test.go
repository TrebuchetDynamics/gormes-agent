package authcodex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizeCommandErrorRedactsOAuthSecrets(t *testing.T) {
	for _, input := range []string{
		"access_token=abc",
		"refresh_token=abc",
		"Authorization: Bearer abc",
		"client_secret=abc",
	} {
		if got := SanitizeCommandError(input); got != "[redacted]" {
			t.Fatalf("SanitizeCommandError(%q) = %q", input, got)
		}
	}
}

func TestSanitizeCommandErrorTruncatesLongErrors(t *testing.T) {
	input := ""
	for len(input) <= 200 {
		input += "x"
	}
	if got := SanitizeCommandError(input); len(got) != 160 {
		t.Fatalf("len(SanitizeCommandError(long)) = %d", len(got))
	}
}

func TestRequestDeviceCodeRetriesOn429(t *testing.T) {
	// Mirrors Hermes fix(auth): retry Codex device-code login on 429 with
	// clear rate-limit message (cbfa018ae). Two 429 responses then a 200
	// must succeed; all 429s must surface a clear rate_limited error.
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"user_code":"XXXX-YYYY","device_auth_id":"test-device-id","interval":1}`))
	}))
	defer srv.Close()

	device, err := RequestDeviceCode(context.Background(), srv.Client(), srv.URL, "test-client")
	if err != nil {
		t.Fatalf("RequestDeviceCode after 2 x 429 then 200: unexpected error: %v", err)
	}
	if device.UserCode != "XXXX-YYYY" {
		t.Fatalf("UserCode = %q, want XXXX-YYYY", device.UserCode)
	}
	if calls != 3 {
		t.Fatalf("server calls = %d, want 3 (2 x 429 + 1 success)", calls)
	}
}

func TestRequestDeviceCodeRateLimitExhaustedReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := RequestDeviceCode(context.Background(), srv.Client(), srv.URL, "test-client")
	if err == nil {
		t.Fatal("expected error after exhausting 429 retries, got nil")
	}
	if !strings.Contains(err.Error(), "rate_limited") {
		t.Fatalf("error = %q, want it to contain 'rate_limited'", err.Error())
	}
}
