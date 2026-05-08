package main

import (
	"encoding/json"
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
		"--api-key", "plain-openrouter-token",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "plain-openrouter-token") {
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
	if entry.AccessToken != "plain-openrouter-token" {
		t.Fatalf("stored token = %q, want test secret in auth store", entry.AccessToken)
	}
	if entry.BaseURL != "https://openrouter.ai/api/v1" || entry.InferenceBaseURL != "https://openrouter.ai/api/v1" {
		t.Fatalf("entry base URLs = base %q inference %q", entry.BaseURL, entry.InferenceBaseURL)
	}
}

func TestGormesAuthAddBedrockRefusesCredentialPoolMutation(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"auth", "add", "bedrock",
		"--type", "api-key",
		"--label", "aws",
		"--api-key", "plain-bedrock-token",
		"--inference-url", "https://bedrock-runtime.example.invalid",
	)
	combined := stdout + stderr
	if err == nil || !strings.Contains(err.Error(), "bedrock_use_aws_sdk_chain") {
		t.Fatalf("auth add bedrock err = %v, stdout=%s stderr=%s, want bedrock_use_aws_sdk_chain", err, stdout, stderr)
	}
	if strings.Contains(combined+err.Error(), "plain-bedrock-token") || strings.Contains(combined+err.Error(), "https://bedrock-runtime.example.invalid") {
		t.Fatalf("auth add bedrock leaked credential detail:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
	if !strings.Contains(err.Error(), "AWS credential chain") {
		t.Fatalf("auth add bedrock err = %v, want AWS credential chain guidance", err)
	}

	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: "bedrock"})
	if err != nil {
		t.Fatalf("LoadCredentialPool: %v", err)
	}
	if entries := pool.Entries(); len(entries) != 0 {
		t.Fatalf("bedrock credential pool entries = %#v, want none", entries)
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

// TestGormesAuthListJSONEmitsRedactedCredentialPool proves
// `gormes auth list <provider> --json` returns a parseable
// `{build, provider, credentials: [...]}` document with the same
// redacted fields the human row already emits. CI/cron consumers
// monitoring fleet credential health (which provider is exhausted,
// which is in cooldown, which has no entries) need a structured shape
// — scraping the human "(2 credentials)" header is fragile across
// formatting changes.
func TestGormesAuthListJSONEmitsRedactedCredentialPool(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: "openrouter"}, []config.PooledCredential{
		{
			ID:              "openrouter-manual-1",
			Label:           "personal",
			AuthType:        config.CredentialAuthAPIKey,
			Source:          "manual",
			AccessToken:     "plain-list-json-secret",
			RefreshToken:    "plain-list-json-refresh",
			LastStatus:      config.CredentialStatusExhausted,
			LastErrorReason: "rate_limited",
		},
	}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "list", "openrouter", "--json")
	if err != nil {
		t.Fatalf("auth list --json: %v\nstdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"plain-list-json-secret", "plain-list-json-refresh"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("auth list --json leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Provider    string `json:"provider"`
		Redacted    bool   `json:"redacted"`
		Credentials []struct {
			ID              string `json:"id"`
			Label           string `json:"label"`
			AuthType        string `json:"auth_type"`
			Source          string `json:"source"`
			Status          string `json:"status"`
			Reason          string `json:"reason"`
			SecretsRedacted bool   `json:"secrets_redacted"`
		} `json:"credentials"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version || got.Build.GitCommit == "" {
		t.Fatalf("build provenance missing/wrong: %+v", got.Build)
	}
	if got.Provider != "openrouter" {
		t.Fatalf("got.Provider = %q, want openrouter", got.Provider)
	}
	if !got.Redacted {
		t.Fatalf("got.Redacted = false, want true")
	}
	if len(got.Credentials) != 1 {
		t.Fatalf("got %d credentials, want 1", len(got.Credentials))
	}
	c := got.Credentials[0]
	if c.ID != "openrouter-manual-1" || c.Label != "personal" || c.AuthType != "api_key" {
		t.Fatalf("credential shape unexpected: %+v", c)
	}
	if c.Status != "exhausted" || c.Reason != "rate_limited" {
		t.Fatalf("status/reason = %q/%q, want exhausted/rate_limited", c.Status, c.Reason)
	}
	if !c.SecretsRedacted {
		t.Fatalf("secrets_redacted must be true; got %+v", c)
	}
	// JSON mode must not interleave the human row, which would make stdout
	// unparseable.
	if strings.Contains(stdout, "openrouter (1 credentials)") {
		t.Fatalf("--json must not emit the human row; got:\n%s", stdout)
	}
}

// TestGormesAuthListJSONEmptyPoolEmitsEmptyArray proves the JSON
// surface stays parseable when no credentials exist for a provider —
// consumers see `{"credentials": []}`, not the free-form
// `credential_pool_empty provider=X` message.
func TestGormesAuthListJSONEmptyPoolEmitsEmptyArray(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeOneshotFlagCommand(cmd, "auth", "list", "openrouter", "--json")
	if err != nil {
		t.Fatalf("auth list --json on empty pool: %v", err)
	}
	var got struct {
		Provider    string `json:"provider"`
		Credentials []any  `json:"credentials"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("empty-pool stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Provider != "openrouter" {
		t.Fatalf("got.Provider = %q, want openrouter", got.Provider)
	}
	if got.Credentials == nil {
		t.Fatalf("credentials must be `[]`, not omitted/null; got %q", stdout)
	}
	if len(got.Credentials) != 0 {
		t.Fatalf("empty pool credentials must have len=0; got %d", len(got.Credentials))
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
	if !strings.Contains(stdout, "provider=openrouter status=logged_in") || strings.Contains(stdout+stderr, "plain-token-a") {
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
	if !strings.Contains(stdout, "provider=openrouter status=logged_out") {
		t.Fatalf("status after logout stdout = %q", stdout)
	}
}

// TestGormesAuthAddJSONEmitsStructuredOutcome proves
// `gormes auth add <provider> --type api-key --api-key X --json`
// returns `{build, action, provider, id, label, redacted}` so fleet
// credential-provisioning automation can record the assigned id +
// label per machine without scraping `auth_api_key_saved` prose.
// The raw API key MUST never appear in stdout.
func TestGormesAuthAddJSONEmitsStructuredOutcome(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "add", "openrouter",
		"--type", "api-key",
		"--label", "primary",
		"--api-key", "sk-or-must-not-leak-Z9",
		"--json",
	)
	if err != nil {
		t.Fatalf("auth add --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "sk-or-must-not-leak-Z9") {
		t.Fatalf("auth add --json LEAKED the raw API key:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action   string `json:"action"`
		Provider string `json:"provider"`
		ID       string `json:"id"`
		Label    string `json:"label"`
		Redacted bool   `json:"redacted"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("auth add --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "added" {
		t.Errorf("action = %q, want %q", got.Action, "added")
	}
	if got.Provider != "openrouter" {
		t.Errorf("provider = %q, want openrouter", got.Provider)
	}
	if got.ID == "" {
		t.Errorf("id must be populated (auto-assigned credential id)")
	}
	if got.Label != "primary" {
		t.Errorf("label = %q, want primary", got.Label)
	}
	if !got.Redacted {
		t.Errorf("redacted must always be true")
	}
}

// TestGormesAuthRemoveJSONEmitsStructuredOutcome proves
// `gormes auth remove <provider> <target> --json` returns
// `{build, action, provider, removed: {id, label}, redacted}` so
// fleet credential-rotation automation can audit which credential
// was actually removed without scraping `auth_credential_removed`
// `key=value` lines. The raw access token MUST never appear.
func TestGormesAuthRemoveJSONEmitsStructuredOutcome(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "openrouter", []config.PooledCredential{
		{ID: "cred-a", Label: "alpha", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-token-X"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "remove", "openrouter", "cred-a", "--json")
	if err != nil {
		t.Fatalf("auth remove --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "plain-token-X") {
		t.Fatalf("auth remove --json LEAKED secret:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action   string `json:"action"`
		Provider string `json:"provider"`
		Removed  struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"removed"`
		Redacted bool `json:"redacted"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("auth remove --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "removed" {
		t.Errorf("action = %q, want %q", got.Action, "removed")
	}
	if got.Provider != "openrouter" {
		t.Errorf("provider = %q, want openrouter", got.Provider)
	}
	if got.Removed.ID != "cred-a" || got.Removed.Label != "alpha" {
		t.Errorf("removed = %+v, want {ID: cred-a, Label: alpha}", got.Removed)
	}
	if !got.Redacted {
		t.Errorf("redacted must be true (we never returned the raw token)")
	}
}

// TestGormesAuthResetJSONEmitsStructuredOutcome proves the reset
// command emits `{build, action, provider, count, redacted}` for
// fleet automation clearing exhaustion across machines.
func TestGormesAuthResetJSONEmitsStructuredOutcome(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "openrouter", []config.PooledCredential{
		{ID: "cred-a", Label: "alpha", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-A", LastStatus: config.CredentialStatusExhausted},
		{ID: "cred-b", Label: "beta", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-B", LastStatus: config.CredentialStatusOK},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "reset", "openrouter", "--json")
	if err != nil {
		t.Fatalf("auth reset --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "plain-") {
		t.Fatalf("auth reset --json LEAKED secret:\nstdout=%s", stdout)
	}
	var got struct {
		Action   string `json:"action"`
		Provider string `json:"provider"`
		Count    int    `json:"count"`
		Redacted bool   `json:"redacted"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("auth reset --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Action != "reset" {
		t.Errorf("action = %q, want %q", got.Action, "reset")
	}
	if got.Count != 2 {
		t.Errorf("count = %d, want 2", got.Count)
	}
}

// TestGormesAuthLogoutJSONEmitsStructuredOutcome proves logout
// emits `{build, action: "logged_out"|"absent", provider, redacted}`.
// The "absent" path runs when the credential pool is already empty —
// fleet scripts iterating across hosts need a stable parseable signal.
func TestGormesAuthLogoutJSONEmitsStructuredOutcome(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "openrouter", []config.PooledCredential{
		{ID: "cred-a", Label: "alpha", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-X"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "logout", "openrouter", "--json")
	if err != nil {
		t.Fatalf("auth logout --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "plain-X") {
		t.Fatalf("auth logout --json LEAKED secret:\nstdout=%s", stdout)
	}
	var got struct {
		Action   string `json:"action"`
		Provider string `json:"provider"`
		Redacted bool   `json:"redacted"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("auth logout --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Action != "logged_out" {
		t.Errorf("action = %q, want %q", got.Action, "logged_out")
	}

	// Second logout — pool now empty; action must be "absent".
	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err = executeOneshotFlagCommand(cmd, "auth", "logout", "openrouter", "--json")
	if err != nil {
		t.Fatalf("auth logout --json (already empty): %v\nstdout=%s", err, stdout)
	}
	var got2 struct {
		Action string `json:"action"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got2); jsonErr != nil {
		t.Fatalf("must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got2.Action != "absent" {
		t.Errorf("action = %q, want %q", got2.Action, "absent")
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
	stdout, stderr, err := executeRootCommandForTest(cmd, "login", "--provider", "openai-codex")
	if err == nil {
		t.Fatalf("login removed-command typo path returned nil error: stdout=%s stderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	for _, want := range []string{"unknown command", "did you mean", "gormes auth add <provider> --type oauth"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("combined output missing %q:\nstdout=%s\nstderr=%s\nerr=%v", want, stdout, stderr, err)
		}
	}
	if strings.Contains(combined, "openai-codex") {
		t.Fatalf("login suggestion leaked provider argument:\n%s", combined)
	}
}

func TestGormesAuthBareReadout(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "openrouter", []config.PooledCredential{
		{ID: "openrouter-manual-1", Label: "personal", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-openrouter-token"},
	})
	seedAuthCommandCredentials(t, config.CodexOAuthProvider, []config.PooledCredential{
		{ID: "codex-device-1", Label: "codex", AuthType: config.CredentialAuthOAuth, Source: config.CodexOAuthSourceDeviceCode, AccessToken: "plain-codex-access", RefreshToken: "plain-codex-refresh"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth")
	if err != nil {
		t.Fatalf("auth: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"plain-openrouter-token", "plain-codex-access", "plain-codex-refresh"} {
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
