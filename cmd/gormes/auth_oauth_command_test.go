package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestGormesAuthAddCodexOAuthStoresDeviceCodeCredential(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := authCodexOAuthLogin
	authCodexOAuthLogin = func(_ context.Context, req codexOAuthLoginRequest) (config.CodexOAuthTokens, error) {
		if req.Label != "codex-team" {
			t.Fatalf("login request label = %q, want codex-team", req.Label)
		}
		return config.CodexOAuthTokens{
			AccountID:    "codex-device-1",
			Label:        "codex-team",
			AccessToken:  "codex-access-secret",
			RefreshToken: "codex-refresh-secret",
			BaseURL:      "https://chatgpt.com/backend-api/codex",
			Source:       config.CodexOAuthSourceDeviceCode,
		}, nil
	}
	t.Cleanup(func() { authCodexOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"auth", "add", "openai-codex",
		"--type", "oauth",
		"--label", "codex-team",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"codex-access-secret", "codex-refresh-secret"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("auth add codex leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
	if !strings.Contains(stdout, "auth_oauth_saved") || !strings.Contains(stdout, "openai-codex") {
		t.Fatalf("stdout = %q, want auth_oauth_saved evidence", stdout)
	}

	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	entries := pool.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want one Codex OAuth credential", entries)
	}
	entry := entries[0]
	if entry.ID != "codex-device-1" || entry.Label != "codex-team" || entry.AuthType != config.CredentialAuthOAuth {
		t.Fatalf("entry metadata = %#v", entry)
	}
	if entry.AccessToken != "codex-access-secret" || entry.RefreshToken != "codex-refresh-secret" {
		t.Fatalf("stored token fields not persisted for runtime use: %#v", entry)
	}
	if entry.BaseURL != "https://chatgpt.com/backend-api/codex" || entry.InferenceBaseURL != "https://chatgpt.com/backend-api/codex" {
		t.Fatalf("entry base URLs = base %q inference %q", entry.BaseURL, entry.InferenceBaseURL)
	}
}

func TestRunCodexDeviceCodeLoginUsesHermesDeviceFlow(t *testing.T) {
	var sawUserCode bool
	var sawPoll bool
	var sawExchange bool
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/accounts/deviceauth/usercode":
			sawUserCode = true
			if r.Method != http.MethodPost {
				t.Fatalf("usercode method = %s", r.Method)
			}
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode usercode body: %v", err)
			}
			if body["client_id"] != codexOAuthClientID {
				t.Fatalf("client_id = %q", body["client_id"])
			}
			_, _ = w.Write([]byte(`{"user_code":"CODE-123","device_auth_id":"device-1","interval":3}`))
		case "/api/accounts/deviceauth/token":
			sawPoll = true
			_, _ = w.Write([]byte(`{"authorization_code":"auth-code-1","code_verifier":"verifier-1"}`))
		case "/oauth/token":
			sawExchange = true
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm: %v", err)
			}
			if got := r.Form.Get("grant_type"); got != "authorization_code" {
				t.Fatalf("grant_type = %q", got)
			}
			if got := r.Form.Get("redirect_uri"); got != server.URL+"/deviceauth/callback" {
				t.Fatalf("redirect_uri = %q", got)
			}
			_, _ = w.Write([]byte(`{"access_token":"access-from-device","refresh_token":"refresh-from-device"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out bytes.Buffer
	tokens, err := runCodexDeviceCodeLogin(context.Background(), codexOAuthLoginRequest{
		Label:    "device-label",
		Out:      &out,
		Client:   server.Client(),
		Issuer:   server.URL,
		TokenURL: server.URL + "/oauth/token",
		Sleep:    func(time.Duration) {},
		Now:      func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("runCodexDeviceCodeLogin: %v\nout=%s", err, out.String())
	}
	if !sawUserCode || !sawPoll || !sawExchange {
		t.Fatalf("saw usercode=%t poll=%t exchange=%t", sawUserCode, sawPoll, sawExchange)
	}
	if tokens.Label != "device-label" || tokens.AccessToken != "access-from-device" || tokens.RefreshToken != "refresh-from-device" {
		t.Fatalf("tokens = %#v", tokens)
	}
	if tokens.Source != config.CodexOAuthSourceDeviceCode || tokens.BaseURL != "https://chatgpt.com/backend-api/codex" {
		t.Fatalf("token source/base = %#v", tokens)
	}
	if strings.Contains(out.String(), "access-from-device") || strings.Contains(out.String(), "refresh-from-device") {
		t.Fatalf("device login output leaked tokens:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "CODE-123") {
		t.Fatalf("device login output missing user code:\n%s", out.String())
	}
}
