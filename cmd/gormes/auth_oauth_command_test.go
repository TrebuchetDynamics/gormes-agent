package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

func TestAuthAddCodexAlwaysRunsDeviceCode(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	shadowHome := filepath.Join(t.TempDir(), "shadow-codex")
	if err := os.MkdirAll(shadowHome, 0o700); err != nil {
		t.Fatalf("mkdir shadow codex home: %v", err)
	}
	shadowPath := filepath.Join(shadowHome, "auth.json")
	if err := os.WriteFile(shadowPath, []byte(`{"tokens":{"access_token":"plain-shadow-access","refresh_token":"plain-shadow-refresh"}}`), 0o600); err != nil {
		t.Fatalf("write shadow codex auth: %v", err)
	}
	t.Setenv("CODEX_HOME", shadowHome)

	calls := 0
	prev := authCodexOAuthLogin
	authCodexOAuthLogin = func(_ context.Context, req codexOAuthLoginRequest) (config.CodexOAuthTokens, error) {
		calls++
		return config.CodexOAuthTokens{
			AccountID:    "codex-device-1",
			Label:        "codex-device",
			AccessToken:  "plain-device-access",
			RefreshToken: "plain-device-refresh",
			BaseURL:      "https://chatgpt.com/backend-api/codex",
			Source:       config.CodexOAuthSourceDeviceCode,
		}, nil
	}
	t.Cleanup(func() { authCodexOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "add", "openai-codex", "--type", "oauth")
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if calls != 1 {
		t.Fatalf("device-code login calls = %d, want 1", calls)
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	entries := pool.Entries()
	if len(entries) != 1 || entries[0].AccessToken != "plain-device-access" || entries[0].Source != config.CodexOAuthSourceDeviceCode {
		t.Fatalf("stored entries = %#v, want device-code credential only", entries)
	}
	for _, leak := range []string{"plain-shadow-access", "plain-shadow-refresh", shadowPath} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("auth add leaked or imported shadow value %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
}

func TestAuthAddCodexEnvelopeMessage(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := authCodexOAuthLogin
	authCodexOAuthLogin = func(context.Context, codexOAuthLoginRequest) (config.CodexOAuthTokens, error) {
		return config.CodexOAuthTokens{AccessToken: "plain-device-access", RefreshToken: "plain-device-refresh"}, nil
	}
	t.Cleanup(func() { authCodexOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "add", "openai-codex", "--type", "oauth")
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Hermes will keep working independently with its own session") {
		t.Fatalf("stdout = %q, want Codex/Hermes isolation envelope", stdout)
	}
}

func TestAuthAddCodexFailureLeavesPoolUntouched(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	before := []config.PooledCredential{{ID: "existing-codex", Label: "existing", AuthType: config.CredentialAuthOAuth, Source: config.CodexOAuthSourceDeviceCode, AccessToken: "plain-existing-access", RefreshToken: "plain-existing-refresh"}}
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider}, before); err != nil {
		t.Fatalf("seed pool: %v", err)
	}
	prev := authCodexOAuthLogin
	authCodexOAuthLogin = func(context.Context, codexOAuthLoginRequest) (config.CodexOAuthTokens, error) {
		return config.CodexOAuthTokens{}, errors.New("provider body contains access_token plain-device-access")
	}
	t.Cleanup(func() { authCodexOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "add", "openai-codex", "--type", "oauth")
	if err == nil || !strings.Contains(err.Error(), "codex_device_code_failed") {
		t.Fatalf("Execute err = %v, want codex_device_code_failed\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(err.Error()+stdout+stderr, "plain-device-access") {
		t.Fatalf("failure output leaked token-shaped body: err=%v stdout=%s stderr=%s", err, stdout, stderr)
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	entries := pool.Entries()
	if len(entries) != 1 || entries[0].ID != before[0].ID || entries[0].AccessToken != before[0].AccessToken {
		t.Fatalf("entries after failed login = %#v, want original pool untouched", entries)
	}
}

func TestAuthAddCodexClearsDeviceCodeSuppression(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	authPath := filepath.Join(os.Getenv("GORMES_HOME"), "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o700); err != nil {
		t.Fatalf("mkdir auth store: %v", err)
	}
	if err := os.WriteFile(authPath, []byte(`{"version":1,"suppressed_sources":{"openai-codex":["device_code","device-code"]}}`), 0o600); err != nil {
		t.Fatalf("write auth store: %v", err)
	}
	prev := authCodexOAuthLogin
	authCodexOAuthLogin = func(context.Context, codexOAuthLoginRequest) (config.CodexOAuthTokens, error) {
		return config.CodexOAuthTokens{AccessToken: "plain-device-access", RefreshToken: "plain-device-refresh"}, nil
	}
	t.Cleanup(func() { authCodexOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "add", "openai-codex", "--type", "oauth")
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	data, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("read auth store: %v", err)
	}
	var authStore map[string]any
	if err := json.Unmarshal(data, &authStore); err != nil {
		t.Fatalf("unmarshal auth store: %v", err)
	}
	if suppressed, ok := authStore["suppressed_sources"]; ok {
		t.Fatalf("auth store still has Codex device suppression: %#v\n%s", suppressed, data)
	}
}

func TestAuthAddCodexEmergencyImportFlagRequiresExplicitInvocation(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	importPath := filepath.Join(t.TempDir(), "codex-auth.json")
	freshToken := codexTestJWT(t, time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC))
	writeCodexCLIAuthFixture(t, importPath, freshToken, "plain-import-refresh")

	prev := authCodexOAuthLogin
	authCodexOAuthLogin = func(context.Context, codexOAuthLoginRequest) (config.CodexOAuthTokens, error) {
		return config.CodexOAuthTokens{}, errors.New("device flow should not run for explicit emergency import")
	}
	t.Cleanup(func() { authCodexOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "add", "openai-codex", "--type", "oauth", "--emergency-import-from-codex-cli", importPath)
	if err != nil {
		t.Fatalf("Execute explicit import: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "CODEx CLI / VS Code refresh-token race") && !strings.Contains(stdout, "Codex CLI / VS Code refresh-token race") {
		t.Fatalf("stdout = %q, want screen-filling race warning", stdout)
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	entries := pool.Entries()
	if len(entries) != 1 || entries[0].Source != config.CodexOAuthSourceCodexCLIImport || entries[0].AccessToken != freshToken {
		t.Fatalf("entries = %#v, want explicit codex-cli import", entries)
	}

	setupOneshotFlagTestEnv(t)
	expiredPath := filepath.Join(t.TempDir(), "expired-codex-auth.json")
	writeCodexCLIAuthFixture(t, expiredPath, codexTestJWT(t, time.Unix(1, 0).UTC()), "plain-expired-refresh")
	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err = executeOneshotFlagCommand(cmd, "auth", "add", "openai-codex", "--type", "oauth", "--emergency-import-from-codex-cli", expiredPath)
	if err == nil || !strings.Contains(err.Error(), "codex_emergency_import_jwt_expired") {
		t.Fatalf("Execute expired import err = %v, want codex_emergency_import_jwt_expired\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
}

func writeCodexCLIAuthFixture(t *testing.T, path, accessToken, refreshToken string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir auth fixture: %v", err)
	}
	payload := map[string]any{"tokens": map[string]string{"access_token": accessToken, "refresh_token": refreshToken}}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal auth fixture: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write auth fixture: %v", err)
	}
}

func codexTestJWT(t *testing.T, exp time.Time) string {
	t.Helper()
	payload, err := json.Marshal(map[string]int64{"exp": exp.Unix()})
	if err != nil {
		t.Fatalf("marshal jwt payload: %v", err)
	}
	return fmt.Sprintf("header.%s.signature", base64.RawURLEncoding.EncodeToString(payload))
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

func TestGormesAuthAddGoogleGeminiOAuthStoresGooglePKCECredential(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := authGoogleGeminiOAuthLogin
	authGoogleGeminiOAuthLogin = func(_ context.Context, req googleGeminiOAuthLoginRequest) (googleGeminiOAuthTokens, error) {
		if req.Label != "" {
			t.Fatalf("login request label = %q, want empty so email can supply label", req.Label)
		}
		return googleGeminiOAuthTokens{
			Email:        "gemini-user@example.test",
			AccessToken:  "google-access-secret",
			RefreshToken: "google-refresh-secret",
			BaseURL:      "cloudcode-pa://google",
			ExpiresAtMS:  1_900_000_000_000,
			Source:       "google_pkce",
		}, nil
	}
	t.Cleanup(func() { authGoogleGeminiOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"auth", "add", "google-gemini-cli",
		"--type", "oauth",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"google-access-secret", "google-refresh-secret"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("auth add google leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
	if !strings.Contains(stdout, "auth_oauth_saved") || !strings.Contains(stdout, "provider=google-gemini-cli") || !strings.Contains(stdout, "source=google_pkce") || !strings.Contains(stdout, "redacted=true") {
		t.Fatalf("stdout = %q, want redacted google auth_oauth_saved evidence", stdout)
	}

	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: "google-gemini-cli"})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	entries := pool.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want one Google Gemini OAuth credential", entries)
	}
	entry := entries[0]
	if entry.ID != "google-gemini-cli-manual-1" || entry.Label != "gemini-user@example.test" || entry.AuthType != config.CredentialAuthOAuth || entry.Source != "manual:google_pkce" {
		t.Fatalf("entry metadata = %#v", entry)
	}
	if entry.AccessToken != "google-access-secret" || entry.RefreshToken != "google-refresh-secret" {
		t.Fatalf("entry tokens not persisted for runtime use: %#v", entry)
	}
	if entry.BaseURL != "cloudcode-pa://google" || entry.InferenceBaseURL != "cloudcode-pa://google" || entry.ExpiresAtMS != 1_900_000_000_000 {
		t.Fatalf("entry runtime metadata = %#v", entry)
	}
}

func TestGormesAuthAddQwenOAuthImportsCLIcredential(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := authQwenOAuthImport
	authQwenOAuthImport = func(_ context.Context, req qwenOAuthImportRequest) (qwenOAuthTokens, error) {
		if req.Label != "qwen-team" {
			t.Fatalf("import request label = %q, want qwen-team", req.Label)
		}
		return qwenOAuthTokens{
			AccountID:    "qwen-cli-1",
			Label:        "qwen-cli-profile",
			AccessToken:  "qwen-access-secret",
			RefreshToken: "qwen-refresh-secret",
			BaseURL:      "https://portal.qwen.example.test/v1",
			ExpiresAtMS:  1_900_000_000_000,
			Source:       "qwen_cli",
		}, nil
	}
	t.Cleanup(func() { authQwenOAuthImport = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"auth", "add", "qwen-oauth",
		"--type", "oauth",
		"--label", "qwen-team",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"qwen-access-secret", "qwen-refresh-secret"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("auth add qwen leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
	if !strings.Contains(stdout, "auth_oauth_saved") || !strings.Contains(stdout, "provider=qwen-oauth") || !strings.Contains(stdout, "source=qwen_cli") || !strings.Contains(stdout, "redacted=true") {
		t.Fatalf("stdout = %q, want redacted qwen auth_oauth_saved evidence", stdout)
	}

	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: "qwen-oauth"})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	entries := pool.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want one Qwen OAuth credential", entries)
	}
	entry := entries[0]
	if entry.ID != "qwen-cli-1" || entry.Label != "qwen-team" || entry.AuthType != config.CredentialAuthOAuth || entry.Source != "manual:qwen_cli" {
		t.Fatalf("entry metadata = %#v", entry)
	}
	if entry.AccessToken != "qwen-access-secret" || entry.RefreshToken != "qwen-refresh-secret" {
		t.Fatalf("entry tokens not persisted for runtime use: %#v", entry)
	}
	if entry.BaseURL != "https://portal.qwen.example.test/v1" || entry.InferenceBaseURL != "https://portal.qwen.example.test/v1" || entry.ExpiresAtMS != 1_900_000_000_000 {
		t.Fatalf("entry runtime metadata = %#v", entry)
	}
}

func TestGormesAuthAddQwenOAuthMissingAndExpiredAreRedacted(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := authQwenOAuthImport
	t.Cleanup(func() { authQwenOAuthImport = prev })

	authQwenOAuthImport = func(context.Context, qwenOAuthImportRequest) (qwenOAuthTokens, error) {
		return qwenOAuthTokens{}, errQwenCLIAuthMissing
	}
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "add", "qwen-oauth", "--type", "oauth")
	if err == nil || !strings.Contains(err.Error(), "qwen_cli_auth_missing") {
		t.Fatalf("missing err = %v stdout=%s stderr=%s, want qwen_cli_auth_missing", err, stdout, stderr)
	}

	authQwenOAuthImport = func(context.Context, qwenOAuthImportRequest) (qwenOAuthTokens, error) {
		return qwenOAuthTokens{}, errQwenCLIRefreshFailed
	}
	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err = executeOneshotFlagCommand(cmd, "auth", "add", "qwen-oauth", "--type", "oauth")
	if err == nil || !strings.Contains(err.Error(), "qwen_cli_refresh_failed") {
		t.Fatalf("expired err = %v stdout=%s stderr=%s, want qwen_cli_refresh_failed", err, stdout, stderr)
	}
	if strings.Contains(err.Error()+stdout+stderr, "access_token") || strings.Contains(err.Error()+stdout+stderr, "refresh_token") {
		t.Fatalf("qwen failure leaked token field names: err=%v stdout=%s stderr=%s", err, stdout, stderr)
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
