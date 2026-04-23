package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLoginOAuthPersistsTokenAndClientState(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	tokenAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			t.Fatalf("path = %s, want /token", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm(): %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "authorization_code" {
			t.Fatalf("grant_type = %q, want authorization_code", got)
		}
		if got := r.Form.Get("code"); got != "oauth-code-123" {
			t.Fatalf("code = %q, want oauth-code-123", got)
		}
		if got := r.Form.Get("client_id"); got != "client-123" {
			t.Fatalf("client_id = %q, want client-123", got)
		}
		if got := r.Form.Get("client_secret"); got != "secret-456" {
			t.Fatalf("client_secret = %q, want secret-456", got)
		}
		if got := r.Form.Get("code_verifier"); got == "" {
			t.Fatal("code_verifier = empty, want PKCE verifier")
		}
		if got := r.Form.Get("redirect_uri"); got == "" {
			t.Fatal("redirect_uri = empty, want callback URI")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"access-123","refresh_token":"refresh-456","token_type":"Bearer","expires_in":900}`)
	}))
	defer tokenAPI.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	authURLCh := make(chan string, 1)
	resultCh := make(chan OAuthLoginResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := LoginOAuth(ctx, OAuthLoginOptions{
			ServerName:            "protected-api",
			AuthorizationEndpoint: "https://accounts.example.test/oauth2/authorize",
			TokenEndpoint:         tokenAPI.URL + "/token",
			ClientID:              "client-123",
			ClientSecret:          "secret-456",
			Scope:                 "read write",
			NotifyURL: func(target string) {
				authURLCh <- target
			},
		})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	var authURL string
	select {
	case authURL = <-authURLCh:
	case err := <-errCh:
		t.Fatalf("LoginOAuth() early error = %v", err)
	case <-ctx.Done():
		t.Fatalf("timed out waiting for auth URL: %v", ctx.Err())
	}

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("url.Parse(authURL): %v", err)
	}
	query := parsed.Query()
	if got := query.Get("response_type"); got != "code" {
		t.Fatalf("response_type = %q, want code", got)
	}
	if got := query.Get("client_id"); got != "client-123" {
		t.Fatalf("client_id = %q, want client-123", got)
	}
	if got := query.Get("scope"); got != "read write" {
		t.Fatalf("scope = %q, want read write", got)
	}
	if got := query.Get("code_challenge_method"); got != "S256" {
		t.Fatalf("code_challenge_method = %q, want S256", got)
	}
	if got := query.Get("code_challenge"); got == "" {
		t.Fatal("code_challenge = empty, want PKCE challenge")
	}

	redirectURI := query.Get("redirect_uri")
	state := query.Get("state")
	if redirectURI == "" || state == "" {
		t.Fatalf("auth URL missing redirect/state: %s", authURL)
	}
	callbackURL := redirectURI + "?code=oauth-code-123&state=" + url.QueryEscape(state)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("http.Get(callbackURL): %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("callback status = %d, want 200", resp.StatusCode)
	}

	var result OAuthLoginResult
	select {
	case result = <-resultCh:
	case err := <-errCh:
		t.Fatalf("LoginOAuth() error = %v", err)
	case <-ctx.Done():
		t.Fatalf("timed out waiting for login result: %v", ctx.Err())
	}

	tokenPath := filepath.Join(dataHome, "gormes", "mcp-tokens", "protected-api.json")
	if result.Path != tokenPath {
		t.Fatalf("result.Path = %q, want %q", result.Path, tokenPath)
	}

	rawToken, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("ReadFile(tokenPath): %v", err)
	}
	var storedToken map[string]any
	if err := json.Unmarshal(rawToken, &storedToken); err != nil {
		t.Fatalf("json.Unmarshal(token): %v", err)
	}
	if got := storedToken["access_token"]; got != "access-123" {
		t.Fatalf("stored access_token = %#v, want access-123", got)
	}
	if got := storedToken["refresh_token"]; got != "refresh-456" {
		t.Fatalf("stored refresh_token = %#v, want refresh-456", got)
	}

	clientPath := filepath.Join(dataHome, "gormes", "mcp-tokens", "protected-api.client.json")
	rawClient, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("ReadFile(clientPath): %v", err)
	}
	var storedClient map[string]any
	if err := json.Unmarshal(rawClient, &storedClient); err != nil {
		t.Fatalf("json.Unmarshal(client): %v", err)
	}
	if got := storedClient["client_id"]; got != "client-123" {
		t.Fatalf("stored client_id = %#v, want client-123", got)
	}
	if got := storedClient["token_endpoint"]; got != tokenAPI.URL+"/token" {
		t.Fatalf("stored token_endpoint = %#v, want %q", got, tokenAPI.URL+"/token")
	}
}

func TestLoadOAuthAccessTokenRefreshesExpiredToken(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	now := time.Date(2026, 4, 23, 18, 0, 0, 0, time.UTC)
	prevNow := oauthNow
	oauthNow = func() time.Time { return now }
	t.Cleanup(func() { oauthNow = prevNow })

	tokenAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			t.Fatalf("path = %s, want /token", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm(): %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "refresh_token" {
			t.Fatalf("grant_type = %q, want refresh_token", got)
		}
		if got := r.Form.Get("refresh_token"); got != "refresh-456" {
			t.Fatalf("refresh_token = %q, want refresh-456", got)
		}
		if got := r.Form.Get("client_id"); got != "client-123" {
			t.Fatalf("client_id = %q, want client-123", got)
		}
		if got := r.Form.Get("client_secret"); got != "secret-456" {
			t.Fatalf("client_secret = %q, want secret-456", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"refreshed-access","expires_in":1200}`)
	}))
	defer tokenAPI.Close()

	store := NewOAuthTokenStore("protected-api")
	if err := os.MkdirAll(filepath.Dir(store.TokenPath()), 0o755); err != nil {
		t.Fatalf("MkdirAll(token dir): %v", err)
	}
	expiredAt := now.Add(-2 * time.Minute).UnixMilli()
	if err := os.WriteFile(store.TokenPath(), []byte(`{
  "access_token": "expired-access",
  "refresh_token": "refresh-456",
  "expires_at": `+strconv.FormatInt(expiredAt, 10)+`
}`), 0o600); err != nil {
		t.Fatalf("WriteFile(token): %v", err)
	}
	if err := os.WriteFile(store.ClientPath(), []byte(`{
  "client_id": "client-123",
  "client_secret": "secret-456",
  "token_endpoint": "`+tokenAPI.URL+`/token"
}`), 0o600); err != nil {
		t.Fatalf("WriteFile(client): %v", err)
	}

	token, err := LoadOAuthAccessToken(context.Background(), "protected-api")
	if err != nil {
		t.Fatalf("LoadOAuthAccessToken(): %v", err)
	}
	if token != "refreshed-access" {
		t.Fatalf("token = %q, want refreshed-access", token)
	}

	raw, err := os.ReadFile(store.TokenPath())
	if err != nil {
		t.Fatalf("ReadFile(token): %v", err)
	}
	if !strings.Contains(string(raw), "refreshed-access") {
		t.Fatalf("stored token file = %s, want refreshed-access", string(raw))
	}
}
