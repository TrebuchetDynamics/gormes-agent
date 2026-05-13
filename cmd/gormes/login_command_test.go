package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func executeRootCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := executeRootCommand(cmd, args...)
	return stdout.String(), stderr.String(), err
}

func TestGormesLoginCodexProviderDelegatesToOAuthAdd(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := authCodexOAuthLogin
	authCodexOAuthLogin = func(_ context.Context, req codexOAuthLoginRequest) (config.CodexOAuthTokens, error) {
		if req.Label != "" {
			t.Fatalf("login request label = %q, want empty legacy login label", req.Label)
		}
		return config.CodexOAuthTokens{
			AccountID:    "codex-login-1",
			Label:        "codex-login",
			AccessToken:  "plain-login-access",
			RefreshToken: "plain-login-refresh",
			BaseURL:      "https://chatgpt.com/backend-api/codex",
			Source:       config.CodexOAuthSourceDeviceCode,
		}, nil
	}
	t.Cleanup(func() { authCodexOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "login", "--provider", "openai-codex", "--no-browser")
	if err != nil {
		t.Fatalf("login --provider openai-codex: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"plain-login-access", "plain-login-refresh"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("login leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
	if !strings.Contains(stdout, "auth_oauth_saved provider=openai-codex") || !strings.Contains(stdout, "redacted=true") {
		t.Fatalf("stdout = %q, want Codex OAuth save evidence", stdout)
	}

	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	entries := pool.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want one Codex credential", entries)
	}
	if entries[0].ID != "codex-login-1" || entries[0].AccessToken != "plain-login-access" || entries[0].RefreshToken != "plain-login-refresh" {
		t.Fatalf("stored Codex credential = %#v", entries[0])
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Hermes.Provider != config.CodexOAuthProvider {
		t.Fatalf("cfg.Hermes.Provider = %q, want %q", cfg.Hermes.Provider, config.CodexOAuthProvider)
	}
}

func TestGormesLoginIsRegisteredAsCobraCommand(t *testing.T) {
	cmd := newRootCommandWithRuntime(rootRuntime{})
	found, _, err := cmd.Find([]string{"login", "--provider", "openai-codex"})
	if err != nil {
		t.Fatalf("Find login: %v", err)
	}
	if found == nil || found.Name() != "login" {
		t.Fatalf("found command = %#v, want top-level login", found)
	}
}

func TestGormesLoginRequiresExplicitProviderWithoutSideEffects(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := authCodexOAuthLogin
	authCodexOAuthLogin = func(context.Context, codexOAuthLoginRequest) (config.CodexOAuthTokens, error) {
		return config.CodexOAuthTokens{}, errors.New("device code should not run without --provider")
	}
	t.Cleanup(func() { authCodexOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "login")
	if err == nil {
		t.Fatalf("bare login error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	for _, want := range []string{"auth_login_provider_required", "openai-codex", "gormes auth add openai-codex --type oauth"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("bare login output missing %q:\nstdout=%s\nstderr=%s\nerr=%v", want, stdout, stderr, err)
		}
	}
	assertCredentialCount(t, config.CodexOAuthProvider, 0)
}

func TestGormesLoginRejectsUnsupportedProviderWithoutLeakingValue(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "login", "--provider", "plain-secret-provider")
	if err == nil {
		t.Fatalf("unsupported-provider login error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	for _, want := range []string{"auth_login_provider_unsupported", "allowed=nous|openai-codex"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("unsupported-provider output missing %q:\nstdout=%s\nstderr=%s\nerr=%v", want, stdout, stderr, err)
		}
	}
	if strings.Contains(combined, "plain-secret-provider") {
		t.Fatalf("unsupported-provider output leaked provider arg:\n%s", combined)
	}
	assertCredentialCount(t, config.CodexOAuthProvider, 0)
}
