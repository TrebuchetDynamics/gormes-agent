package hermes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newFakeNousPortal(t *testing.T) *httptest.Server {
	t.Helper()
	deviceCode := "test-device-code-123"
	userCode := "ABCD-EFGH"
	accessToken := "test-access-token-abc"
	refreshToken := "test-refresh-token-xyz"
	agentKey := "test-agent-key-456"

	mux := http.NewServeMux()

	mux.HandleFunc("/api/oauth/device/code", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"device_code":               deviceCode,
			"user_code":                 userCode,
			"verification_uri_complete": "https://portal.nousresearch.com/activate?code=" + userCode,
			"expires_in":                600,
			"interval":                  5,
		})
	})

	mux.HandleFunc("/api/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.ParseForm()
		grantType := r.FormValue("grant_type")
		switch grantType {
		case "urn:ietf:params:oauth:grant-type:device_code":
			json.NewEncoder(w).Encode(map[string]any{
				"access_token":  accessToken,
				"refresh_token": refreshToken,
				"token_type":    "Bearer",
				"expires_in":    3600,
				"scope":         "openid profile",
			})
		case "refresh_token":
			if r.Header.Get("x-nous-refresh-token") != refreshToken {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{
					"error":             "invalid_grant",
					"error_description": "Refresh token reuse detected",
				})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"access_token":  accessToken + "-refreshed",
				"refresh_token": refreshToken + "-rotated",
				"token_type":    "Bearer",
				"expires_in":    3600,
			})
		default:
			http.Error(w, "unsupported grant type", http.StatusBadRequest)
		}
	})

	mux.HandleFunc("/api/oauth/agent-key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || len(auth) < 8 {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error":             "invalid_token",
				"error_description": "Access token is invalid or expired",
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"api_key":    agentKey,
			"key_id":     "key-nous-001",
			"expires_at": "2026-05-09T12:00:00Z",
			"expires_in": 86400,
			"key_reused": false,
		})
	})

	return httptest.NewServer(mux)
}

func newFakeNousPortalWithErrors(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/oauth/device/code", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"error": "temporarily_unavailable"})
	})

	mux.HandleFunc("/api/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		grantType := r.FormValue("grant_type")
		switch grantType {
		case "urn:ietf:params:oauth:grant-type:device_code":
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error":             "expired_token",
				"error_description": "The device code has expired",
			})
		case "refresh_token":
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error":             "invalid_grant",
				"error_description": "Refresh token reuse detected",
			})
		default:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": "unsupported_grant_type", "error_description": "unsupported grant type"})
		}
	})

	mux.HandleFunc("/api/oauth/agent-key", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_token",
			"error_description": "Access token is invalid or expired",
		})
	})

	return httptest.NewServer(mux)
}

// TestNousOAuthDeviceCodeLogin_Success proves the full device-code flow.
func TestNousOAuthDeviceCodeLogin_Success(t *testing.T) {
	srv := newFakeNousPortal(t)
	defer srv.Close()

	creds, err := NousOAuthDeviceCodeLogin(context.Background(), NousOAuthLoginOptions{
		PortalBaseURL:    srv.URL,
		InferenceBaseURL: "https://inference.nousresearch.com",
		ClientID:         "test-client",
		Scope:            "openid profile",
		HTTPClient:       srv.Client(),
		OpenBrowser:      func(url string) error { return nil },
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if creds.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if creds.AgentKey == "" {
		t.Error("expected agent key after force_mint")
	}
	if creds.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want Bearer", creds.TokenType)
	}
}

// TestNousOAuthDeviceCodeLogin_ErrorExpiredCode proves error classification.
func TestNousOAuthDeviceCodeLogin_ErrorExpiredCode(t *testing.T) {
	srv := newFakeNousPortalWithErrors(t)
	defer srv.Close()

	_, err := NousOAuthDeviceCodeLogin(context.Background(), NousOAuthLoginOptions{
		PortalBaseURL:    srv.URL,
		InferenceBaseURL: "https://inference.nousresearch.com",
		ClientID:         "test-client",
		Scope:            "openid profile",
		HTTPClient:       srv.Client(),
		OpenBrowser:      func(url string) error { return nil },
	})
	if err == nil {
		t.Fatal("expected error for expired device code")
	}
	authErr, ok := err.(*NousAuthError)
	if !ok {
		t.Fatalf("expected *NousAuthError, got %T: %v", err, err)
	}
	if authErr.Code != "device_code_expired" {
		t.Errorf("code = %q, want device_code_expired", authErr.Code)
	}
	if !authErr.ReloginRequired {
		t.Error("expected relogin_required=true for expired device code")
	}
}

// TestNousRefreshAccessToken_Success proves refresh via X-Nous-Refresh-Token header.
func TestNousRefreshAccessToken_Success(t *testing.T) {
	srv := newFakeNousPortal(t)
	defer srv.Close()

	creds, err := RefreshNousAccessToken(context.Background(), NousOAuthRefreshOptions{
		PortalBaseURL: srv.URL,
		ClientID:      "test-client",
		RefreshToken:  "test-refresh-token-xyz",
		HTTPClient:    srv.Client(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AccessToken != "test-access-token-abc-refreshed" {
		t.Errorf("access_token = %q, want test-access-token-abc-refreshed", creds.AccessToken)
	}
	if creds.RefreshToken != "test-refresh-token-xyz-rotated" {
		t.Errorf("refresh_token = %q, want test-refresh-token-xyz-rotated", creds.RefreshToken)
	}
}

// TestNousRefreshAccessToken_ErrorReuseDetected proves token-theft detection.
func TestNousRefreshAccessToken_ErrorReuseDetected(t *testing.T) {
	srv := newFakeNousPortalWithErrors(t)
	defer srv.Close()

	_, err := RefreshNousAccessToken(context.Background(), NousOAuthRefreshOptions{
		PortalBaseURL: srv.URL,
		ClientID:      "test-client",
		RefreshToken:  "wrong-token",
		HTTPClient:    srv.Client(),
	})
	if err == nil {
		t.Fatal("expected error for invalid refresh token")
	}
	authErr, ok := err.(*NousAuthError)
	if !ok {
		t.Fatalf("expected *NousAuthError, got %T: %v", err, err)
	}
	if authErr.Code != "refresh_token_revoked" {
		t.Errorf("code = %q, want refresh_token_revoked", authErr.Code)
	}
	if !authErr.ReloginRequired {
		t.Error("expected relogin_required=true for revoked refresh token")
	}
}

// TestNousMintAgentKey_Success proves agent key minting via Bearer access token.
func TestNousMintAgentKey_Success(t *testing.T) {
	srv := newFakeNousPortal(t)
	defer srv.Close()

	creds, err := MintNousAgentKey(context.Background(), NousOAuthMintOptions{
		PortalBaseURL:    srv.URL,
		AccessToken:      "test-access-token-abc",
		MinKeyTTLSeconds: 300,
		HTTPClient:       srv.Client(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AgentKey != "test-agent-key-456" {
		t.Errorf("agent_key = %q, want test-agent-key-456", creds.AgentKey)
	}
	if creds.AgentKeyID != "key-nous-001" {
		t.Errorf("agent_key_id = %q, want key-nous-001", creds.AgentKeyID)
	}
	if creds.AgentKeyExpiresAt == "" {
		t.Error("expected non-empty agent_key_expires_at")
	}
}

// TestNousMintAgentKey_ErrorInvalidToken proves error for invalid access token.
func TestNousMintAgentKey_ErrorInvalidToken(t *testing.T) {
	srv := newFakeNousPortalWithErrors(t)
	defer srv.Close()

	_, err := MintNousAgentKey(context.Background(), NousOAuthMintOptions{
		PortalBaseURL:    srv.URL,
		AccessToken:      "expired-token",
		MinKeyTTLSeconds: 300,
		HTTPClient:       srv.Client(),
	})
	if err == nil {
		t.Fatal("expected error for invalid access token")
	}
	authErr, ok := err.(*NousAuthError)
	if !ok {
		t.Fatalf("expected *NousAuthError, got %T: %v", err, err)
	}
	if authErr.Code != "agent_key_minting_failed" {
		t.Errorf("code = %q, want agent_key_minting_failed", authErr.Code)
	}
}

// TestResolveNousRuntimeCredentials_CacheReturnsCachedKey proves
// cached agent key is returned when still valid.
func TestResolveNousRuntimeCredentials_CacheReturnsCachedKey(t *testing.T) {
	srv := newFakeNousPortal(t)
	defer srv.Close()

	creds := NousOAuthCredentials{
		PortalBaseURL:     srv.URL,
		InferenceBaseURL:  "https://inference.nousresearch.com",
		AccessToken:       "cached-access-token",
		RefreshToken:      "cached-refresh-token",
		AgentKey:          "cached-agent-key",
		AgentKeyID:        "cached-key-id",
		AgentKeyExpiresAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		ExpiresAt:         time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		TokenType:         "Bearer",
	}

	rt, err := resolveNousRuntimeFromCreds(context.Background(), creds, NousOAuthRuntimeOptions{
		MinKeyTTLSeconds: 300,
		HTTPClient:       srv.Client(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.APIKey != "cached-agent-key" {
		t.Errorf("api_key = %q, want cached-agent-key", rt.APIKey)
	}
	if rt.Source != "cache" {
		t.Errorf("source = %q, want cache", rt.Source)
	}
}

// TestResolveNousRuntimeCredentials_MintsNearExpiry proves a fresh
// agent key is minted when cached key is near expiry.
func TestResolveNousRuntimeCredentials_MintsNearExpiry(t *testing.T) {
	srv := newFakeNousPortal(t)
	defer srv.Close()

	creds := NousOAuthCredentials{
		PortalBaseURL:     srv.URL,
		InferenceBaseURL:  "https://inference.nousresearch.com",
		AccessToken:       "test-access-token-abc",
		RefreshToken:      "test-refresh-token-xyz",
		AgentKey:          "stale-key",
		AgentKeyID:        "stale-key-id",
		AgentKeyExpiresAt: time.Now().Add(2 * time.Minute).Format(time.RFC3339),
		ExpiresAt:         time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		TokenType:         "Bearer",
	}

	rt, err := resolveNousRuntimeFromCreds(context.Background(), creds, NousOAuthRuntimeOptions{
		MinKeyTTLSeconds: 300,
		HTTPClient:       srv.Client(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.APIKey != "test-agent-key-456" {
		t.Errorf("api_key = %q, want test-agent-key-456 (fresh mint)", rt.APIKey)
	}
	if rt.Source != "portal" {
		t.Errorf("source = %q, want portal", rt.Source)
	}
}

// TestNousAuthError_ImplementsError proves NousAuthError satisfies error interface.
func TestNousAuthError_ImplementsError(t *testing.T) {
	err := &NousAuthError{
		Message:         "test error",
		Code:            "test_code",
		ReloginRequired: true,
	}
	if err.Error() == "" {
		t.Error("Error() returned empty string")
	}
	if !strings.Contains(err.Error(), "test error") {
		t.Errorf("Error() = %q, want containing 'test error'", err.Error())
	}
}
