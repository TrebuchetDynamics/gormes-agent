package main

import (
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestGormesAuthAddAPIKeyPersistsManualEntry(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"auth", "add", "openrouter",
		"--type", "api-key",
		"--label", "personal",
		"--api-key", "sk-test-secret",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "sk-test-secret") {
		t.Fatalf("auth add leaked API key:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(stdout, "auth_api_key_saved") {
		t.Fatalf("stdout = %q, want auth_api_key_saved evidence", stdout)
	}

	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: "openrouter"})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	entries := pool.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want one credential", entries)
	}
	entry := entries[0]
	if entry.Label != "personal" || entry.AuthType != config.CredentialAuthAPIKey || entry.Source != "manual" {
		t.Fatalf("entry metadata = %#v", entry)
	}
	if entry.AccessToken != "sk-test-secret" {
		t.Fatalf("stored token = %q, want test secret in auth store", entry.AccessToken)
	}
	if entry.BaseURL != "https://openrouter.ai/api/v1" || entry.InferenceBaseURL != "https://openrouter.ai/api/v1" {
		t.Fatalf("entry base URLs = base %q inference %q", entry.BaseURL, entry.InferenceBaseURL)
	}
}

func TestGormesAuthListRedactsSecrets(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	resetAt := time.Now().Add(time.Hour).Unix()
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: "openrouter"}, []config.PooledCredential{
		{
			ID:               "openrouter-manual-1",
			Label:            "personal",
			AuthType:         config.CredentialAuthAPIKey,
			Source:           "manual",
			AccessToken:      "plain-existing-token",
			RefreshToken:     "plain-refresh-token",
			LastStatus:       config.CredentialStatusExhausted,
			LastErrorReason:  "rate_limited",
			LastErrorResetAt: resetAt,
		},
	}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "list", "openrouter")
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"plain-existing-token", "plain-refresh-token"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("auth list leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
	for _, want := range []string{"openrouter", "personal", "api_key", "manual", "rate_limited", "redacted=true"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestGormesAuthRemoveByIndexIDOrLabel(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "openrouter", []config.PooledCredential{
		{ID: "cred-a", Label: "alpha", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-token-a"},
		{ID: "cred-b", Label: "beta", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-token-b"},
		{ID: "cred-c", Label: "gamma", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-token-c"},
	})

	for _, target := range []string{"2", "cred-a", "gamma"} {
		cmd := newRootCommandWithRuntime(rootRuntime{})
		stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "remove", "openrouter", target)
		if err != nil {
			t.Fatalf("remove %s: %v\nstdout=%s\nstderr=%s", target, err, stdout, stderr)
		}
		if strings.Contains(stdout+stderr, "plain-token-") {
			t.Fatalf("remove %s leaked secret:\nstdout=%s\nstderr=%s", target, stdout, stderr)
		}
		if !strings.Contains(stdout, "auth_credential_removed") {
			t.Fatalf("remove %s stdout = %q, want removal evidence", target, stdout)
		}
	}

	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: "openrouter"})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	if got := pool.Entries(); len(got) != 0 {
		t.Fatalf("entries after removals = %#v, want empty", got)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "remove", "openrouter", "missing")
	if err == nil || !strings.Contains(err.Error(), "credential_not_found") {
		t.Fatalf("remove missing err = %v, stdout=%s stderr=%s, want credential_not_found", err, stdout, stderr)
	}
}

func TestGormesAuthResetClearsExhaustion(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "openrouter", []config.PooledCredential{
		{ID: "cred-a", Label: "alpha", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-token-a", LastStatus: config.CredentialStatusExhausted, LastErrorCode: 429, LastErrorReason: "rate_limited", LastErrorMessage: "provider said retry", LastErrorResetAt: time.Now().Add(time.Hour).Unix()},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "reset", "openrouter")
	if err != nil {
		t.Fatalf("reset: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "auth_status_reset") {
		t.Fatalf("stdout = %q, want auth_status_reset evidence", stdout)
	}

	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: "openrouter"})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	entry := pool.Entries()[0]
	if entry.LastStatus != config.CredentialStatusOK || entry.LastErrorCode != 0 || entry.LastErrorReason != "" || entry.LastErrorMessage != "" || entry.LastErrorResetAt != 0 {
		t.Fatalf("entry status after reset = %#v", entry)
	}
}

func TestGormesAuthStatusAndLogout(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "openrouter", []config.PooledCredential{
		{ID: "cred-a", Label: "alpha", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-token-a"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "status", "openrouter")
	if err != nil {
		t.Fatalf("status: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "openrouter: logged in") || strings.Contains(stdout+stderr, "plain-token-a") {
		t.Fatalf("status output = stdout:%q stderr:%q", stdout, stderr)
	}

	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err = executeOneshotFlagCommand(cmd, "auth", "logout", "openrouter")
	if err != nil {
		t.Fatalf("logout: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "auth_logged_out") || strings.Contains(stdout+stderr, "plain-token-a") {
		t.Fatalf("logout output = stdout:%q stderr:%q", stdout, stderr)
	}

	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err = executeOneshotFlagCommand(cmd, "auth", "status", "openrouter")
	if err != nil {
		t.Fatalf("status after logout: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "openrouter: logged out") {
		t.Fatalf("status after logout stdout = %q", stdout)
	}
}

func TestGormesAuthBareReadoutListsCredentialPools(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "openrouter", []config.PooledCredential{
		{ID: "cred-a", Label: "alpha", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-token-a"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth")
	if err != nil {
		t.Fatalf("auth: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "openrouter") || !strings.Contains(stdout, "credentials=1") {
		t.Fatalf("bare auth stdout = %q, want provider pool readout", stdout)
	}
	if strings.Contains(stdout+stderr, "plain-token-a") {
		t.Fatalf("bare auth leaked secret:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestGormesLoginPrintsRemovedCommandGuidance(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "login", "--provider", "openai-codex")
	if err != nil {
		t.Fatalf("login compatibility shim returned error: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"command has been removed", "gormes auth", "gormes model", "gormes setup"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestGormesAuthBareReadout(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "openrouter", []config.PooledCredential{
		{ID: "openrouter-manual-1", Label: "personal", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "sk-openrouter-secret"},
	})
	seedAuthCommandCredentials(t, config.CodexOAuthProvider, []config.PooledCredential{
		{ID: "codex-device-1", Label: "codex", AuthType: config.CredentialAuthOAuth, Source: config.CodexOAuthSourceDeviceCode, AccessToken: "codex-access-secret", RefreshToken: "codex-refresh-secret"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth")
	if err != nil {
		t.Fatalf("auth: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"sk-openrouter-secret", "codex-access-secret", "codex-refresh-secret"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("bare auth leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
	for _, want := range []string{"openrouter (1 credentials)", "openai-codex (1 credentials)", "bedrock_identity status=not_checked", "redacted=true"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("bare auth stdout missing %q:\n%s", want, stdout)
		}
	}
}

func seedAuthCommandCredentials(t *testing.T, provider string, entries []config.PooledCredential) {
	t.Helper()
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: provider}, entries); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}
}
