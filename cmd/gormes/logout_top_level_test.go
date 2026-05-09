package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// TestLogoutTopLevelJSONEmitsLifecycleReport proves
// `gormes logout --provider <p> --json` emits the same
// `{build, action, provider, redacted}` shape as `gormes auth logout
// <p> --json` so fleet automation rotating provider auth across
// machines can use either spelling without rewriting their JSON
// parsers. Build provenance leads — same convention as the rest of
// the `--json` arc. Raw tokens MUST never appear.
func TestLogoutTopLevelJSONEmitsLifecycleReport(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "nous", []config.PooledCredential{
		{ID: "nous-cred-1", Label: "primary", AuthType: config.CredentialAuthOAuth, Source: "manual", AccessToken: "plain-token-nous"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "logout", "--provider", "nous", "--json")
	if err != nil {
		t.Fatalf("logout --json: %v\nstderr=%s", err, stderr)
	}
	if strings.Contains(stdout+stderr, "plain-token-nous") {
		t.Fatalf("logout --json LEAKED token:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action   string `json:"action"`
		Provider string `json:"provider"`
		Redacted bool   `json:"redacted"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("logout --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "logged_out" {
		t.Errorf("action = %q, want logged_out", got.Action)
	}
	if got.Provider != "nous" {
		t.Errorf("provider = %q, want nous", got.Provider)
	}
	if !got.Redacted {
		t.Errorf("redacted = false, want true")
	}
}

func TestLogoutTopLevelAcceptsAllowedProviders(t *testing.T) {
	for _, provider := range []string{"nous", config.CodexOAuthProvider, "spotify"} {
		t.Run(provider, func(t *testing.T) {
			setupOneshotFlagTestEnv(t)
			seedAuthCommandCredentials(t, provider, []config.PooledCredential{
				{ID: provider + "-cred-1", Label: "primary", AuthType: config.CredentialAuthOAuth, Source: "manual", AccessToken: "plain-token-" + provider},
			})
			seedAuthCommandCredentials(t, "openrouter", []config.PooledCredential{
				{ID: "openrouter-cred-1", Label: "other", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-token-openrouter"},
			})

			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeOneshotFlagCommand(cmd, "logout", "--provider", provider)
			if err != nil {
				t.Fatalf("logout %s: %v\nstdout=%s\nstderr=%s", provider, err, stdout, stderr)
			}
			if !strings.Contains(stdout, "auth_logged_out provider="+provider) {
				t.Fatalf("stdout = %q, want scoped auth_logged_out evidence", stdout)
			}
			if strings.Contains(stdout+stderr, "plain-token-") {
				t.Fatalf("logout leaked a secret:\nstdout=%s\nstderr=%s", stdout, stderr)
			}
			assertCredentialCount(t, provider, 0)
			assertCredentialCount(t, "openrouter", 1)
		})
	}
}

func TestLogoutTopLevelRejectsOtherProviders(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "anthropic", []config.PooledCredential{
		{ID: "anthropic-cred-1", Label: "anthropic", AuthType: config.CredentialAuthOAuth, Source: "manual", AccessToken: "plain-token-anthropic"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "logout", "--provider", "anthropic")
	if err == nil {
		t.Fatalf("logout anthropic error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 2 {
		t.Fatalf("exit code = %d, want 2 (err=%v)", code, err)
	}
	if !strings.Contains(err.Error(), "auth_logout_provider_unsupported") || !strings.Contains(err.Error(), "nous|openai-codex|spotify") {
		t.Fatalf("err = %v, want unsupported provider evidence with allow-list", err)
	}
	assertCredentialCount(t, "anthropic", 1)
}

func TestLogoutTopLevelDoesNotShadowAuthLogout(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "anthropic", []config.PooledCredential{
		{ID: "anthropic-cred-1", Label: "anthropic", AuthType: config.CredentialAuthOAuth, Source: "manual", AccessToken: "plain-token-anthropic"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "logout", "anthropic")
	if err != nil {
		t.Fatalf("auth logout anthropic: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "auth_logged_out provider=anthropic") {
		t.Fatalf("stdout = %q, want auth logout to retain full provider command", stdout)
	}
	assertCredentialCount(t, "anthropic", 0)
}

func TestLogoutTopLevelRedactsSecrets(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, config.CodexOAuthProvider, []config.PooledCredential{
		{ID: "codex-cred-1", Label: "codex", AuthType: config.CredentialAuthOAuth, Source: "manual", AccessToken: "plain-codex-access", RefreshToken: "plain-codex-refresh"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "logout", "--provider", config.CodexOAuthProvider)
	if err != nil {
		t.Fatalf("logout codex: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"plain-codex-access", "plain-codex-refresh", "credential_pool"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("logout leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
}

func TestLogoutTopLevelIdempotentMissingState(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "nous", []config.PooledCredential{
		{ID: "nous-cred-1", Label: "nous", AuthType: config.CredentialAuthOAuth, Source: "manual", AccessToken: "plain-token-nous"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "logout", "--provider", "nous")
	if err != nil {
		t.Fatalf("first logout: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err = executeOneshotFlagCommand(cmd, "logout", "--provider", "nous")
	if err != nil {
		t.Fatalf("second logout should be idempotent: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "auth_state_absent provider=nous") {
		t.Fatalf("second logout stdout = %q, want auth_state_absent", stdout)
	}
}

func assertCredentialCount(t *testing.T, provider string, want int) {
	t.Helper()
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		t.Fatalf("LoadCredentialPool(%s): %v", provider, err)
	}
	if got := len(pool.Entries()); got != want {
		t.Fatalf("credential count for %s = %d, want %d", provider, got, want)
	}
}
