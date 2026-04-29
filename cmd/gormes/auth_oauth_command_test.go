package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestGormesAuthAddAnthropicOAuthStoresHermesPKCECredential(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := authAnthropicOAuthLogin
	authAnthropicOAuthLogin = func(_ context.Context, req anthropicOAuthLoginRequest) (anthropicOAuthTokens, error) {
		if req.Label != "" {
			t.Fatalf("login request label = %q, want empty so token claim can supply label", req.Label)
		}
		return anthropicOAuthTokens{
			AccountID:    "anthropic-user-1",
			Label:        "anthropic-claim-label",
			AccessToken:  "header.eyJlbWFpbCI6ImFudGhyb3BpYy10ZWFtQGV4YW1wbGUudGVzdCJ9.signature",
			RefreshToken: "anthropic-refresh-plain",
			BaseURL:      "https://api.anthropic.com",
			ExpiresAtMS:  1_900_000_000_000,
			Source:       "hermes_pkce",
		}, nil
	}
	t.Cleanup(func() { authAnthropicOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"auth", "add", "anthropic",
		"--type", "oauth",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"anthropic-refresh-plain", "header.eyJlbWFpbCI6ImFudGhyb3BpYy10ZWFtQGV4YW1wbGUudGVzdCJ9.signature"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("auth add anthropic leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
	if !strings.Contains(stdout, "auth_oauth_saved") || !strings.Contains(stdout, "anthropic") || !strings.Contains(stdout, "redacted=true") {
		t.Fatalf("stdout = %q, want redacted auth_oauth_saved evidence", stdout)
	}

	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: config.AnthropicProvider})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	entries := pool.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want one Anthropic OAuth credential", entries)
	}
	entry := entries[0]
	if entry.ID != "anthropic-user-1" || entry.Label != "anthropic-claim-label" || entry.AuthType != config.CredentialAuthOAuth {
		t.Fatalf("entry metadata = %#v", entry)
	}
	if entry.Source != "manual:hermes_pkce" || entry.AccessToken == "" || entry.RefreshToken != "anthropic-refresh-plain" {
		t.Fatalf("entry source/tokens = %#v", entry)
	}
	if entry.BaseURL != "https://api.anthropic.com" || entry.InferenceBaseURL != "https://api.anthropic.com" || entry.ExpiresAtMS != 1_900_000_000_000 {
		t.Fatalf("entry runtime metadata = %#v", entry)
	}
}

func TestGormesAuthAddNousOAuthMirrorsProviderAndPool(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := authNousOAuthLogin
	authNousOAuthLogin = func(_ context.Context, req nousOAuthLoginRequest) (nousOAuthTokens, error) {
		if req.Label != "nous-team" {
			t.Fatalf("login request label = %q, want nous-team", req.Label)
		}
		if req.PortalBaseURL != "https://portal.example.test" || req.InferenceBaseURL != "https://inference.example.test/v1" {
			t.Fatalf("login request URLs = portal %q inference %q", req.PortalBaseURL, req.InferenceBaseURL)
		}
		return nousOAuthTokens{
			PortalBaseURL:      "https://portal.example.test",
			InferenceBaseURL:   "https://inference.example.test/v1",
			ClientID:           "hermes-cli",
			Scope:              "inference:mint_agent_key",
			TokenType:          "Bearer",
			AccessToken:        "nous-access-secret",
			RefreshToken:       "nous-refresh-secret",
			ExpiresAt:          "2026-03-23T11:00:00Z",
			AgentKey:           "nous-agent-key-secret",
			AgentKeyID:         "agent-key-id",
			AgentKeyExpiresAt:  "2026-03-23T10:30:00Z",
			AgentKeyObtainedAt: "2026-03-23T10:00:10Z",
		}, nil
	}
	t.Cleanup(func() { authNousOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"auth", "add", "nous",
		"--type", "oauth",
		"--label", "nous-team",
		"--portal-url", "https://portal.example.test",
		"--inference-url", "https://inference.example.test/v1",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"nous-access-secret", "nous-refresh-secret", "nous-agent-key-secret"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("auth add nous leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
	if !strings.Contains(stdout, "auth_oauth_saved") || !strings.Contains(stdout, "provider=nous") || !strings.Contains(stdout, "redacted=true") {
		t.Fatalf("stdout = %q, want redacted auth_oauth_saved evidence", stdout)
	}

	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: "nous"})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	entries := pool.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want one canonical Nous device-code credential", entries)
	}
	entry := entries[0]
	if entry.ID != "nous-device-code" || entry.Label != "nous-team" || entry.AuthType != config.CredentialAuthOAuth || entry.Source != "device_code" {
		t.Fatalf("entry metadata = %#v", entry)
	}
	if entry.AccessToken != "nous-access-secret" || entry.RefreshToken != "nous-refresh-secret" || entry.AgentKey != "nous-agent-key-secret" {
		t.Fatalf("entry tokens not persisted for runtime use: %#v", entry)
	}
	if entry.BaseURL != "https://inference.example.test/v1" || entry.InferenceBaseURL != "https://inference.example.test/v1" || entry.PortalBaseURL != "https://portal.example.test" {
		t.Fatalf("entry URLs = %#v", entry)
	}

	var raw struct {
		Providers map[string]struct {
			Label            string `json:"label"`
			AccessToken      string `json:"access_token"`
			RefreshToken     string `json:"refresh_token"`
			AgentKey         string `json:"agent_key"`
			PortalBaseURL    string `json:"portal_base_url"`
			InferenceBaseURL string `json:"inference_base_url"`
		} `json:"providers"`
	}
	if err := readAuthJSONForTest(&raw); err != nil {
		t.Fatalf("read auth.json: %v", err)
	}
	providerState, ok := raw.Providers["nous"]
	if !ok {
		t.Fatalf("providers.nous missing from auth.json: %#v", raw.Providers)
	}
	if providerState.Label != "nous-team" || providerState.AccessToken != "nous-access-secret" || providerState.RefreshToken != "nous-refresh-secret" || providerState.AgentKey != "nous-agent-key-secret" {
		t.Fatalf("providers.nous token state = %#v", providerState)
	}
	if providerState.PortalBaseURL != "https://portal.example.test" || providerState.InferenceBaseURL != "https://inference.example.test/v1" {
		t.Fatalf("providers.nous URLs = %#v", providerState)
	}
}

func TestGormesAuthOAuthErrorsAreAtomic(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seed := []config.PooledCredential{{
		ID:           "existing-anthropic",
		Label:        "existing",
		AuthType:     config.CredentialAuthOAuth,
		Source:       "manual:hermes_pkce",
		AccessToken:  "plain-existing-token",
		RefreshToken: "plain-existing-refresh",
	}}
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: config.AnthropicProvider}, seed); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}
	before, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: config.AnthropicProvider})
	if err != nil {
		t.Fatalf("LoadCredentialPool before: %v", err)
	}
	prev := authAnthropicOAuthLogin
	authAnthropicOAuthLogin = func(context.Context, anthropicOAuthLoginRequest) (anthropicOAuthTokens, error) {
		return anthropicOAuthTokens{}, errAuthFixture("provider raw token access_token=secret should redact")
	}
	t.Cleanup(func() { authAnthropicOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"auth", "add", "anthropic",
		"--type", "oauth",
	)
	if err == nil {
		t.Fatalf("Execute error = nil, want redacted anthropic_oauth_failed evidence\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(stdout+stderr, "secret") {
		t.Fatalf("failure leaked secret-shaped text: err=%v stdout=%s stderr=%s", err, stdout, stderr)
	}
	after, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: config.AnthropicProvider})
	if err != nil {
		t.Fatalf("LoadCredentialPool after: %v", err)
	}
	if got, want := after.Entries(), before.Entries(); len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("credential pool mutated on failed OAuth login\ngot=%#v\nwant=%#v", got, want)
	}
}

type errAuthFixture string

func (e errAuthFixture) Error() string { return string(e) }

func readAuthJSONForTest(target any) error {
	data, err := os.ReadFile(filepath.Join(config.GormesHome(), "auth.json"))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
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
