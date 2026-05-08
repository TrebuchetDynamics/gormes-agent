package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestAuthStatusCommandUsesProviderSpecificReadModel(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, config.CodexOAuthProvider, []config.PooledCredential{
		{ID: "codex-device-1", Label: "codex", AuthType: config.CredentialAuthOAuth, Source: config.CodexOAuthSourceDeviceCode, AccessToken: "plain-codex-access", RefreshToken: "plain-codex-refresh"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "status", config.CodexOAuthProvider)
	if err != nil {
		t.Fatalf("auth status: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"plain-codex-access", "plain-codex-refresh"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("auth status leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
	for _, want := range []string{"auth_status provider=openai-codex", "status=logged_in", "auth_type=oauth_external", "codex-device-1", "redacted=true"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

// TestAuthStatusCommand_JSONEmitsRedactedReport proves
// `gormes auth status <provider> --json` returns a parseable
// auth-readiness document with the same redacted fields the human
// surface emits. CI/cron consumers monitoring fleet credential health
// can ingest this without scraping the human row.
func TestAuthStatusCommand_JSONEmitsRedactedReport(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, config.CodexOAuthProvider, []config.PooledCredential{
		{ID: "codex-device-1", Label: "codex", AuthType: config.CredentialAuthOAuth, Source: config.CodexOAuthSourceDeviceCode, AccessToken: "plain-codex-access", RefreshToken: "plain-codex-refresh"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "status", config.CodexOAuthProvider, "--json")
	if err != nil {
		t.Fatalf("auth status --json: %v\nstdout=%s stderr=%s", err, stdout, stderr)
	}
	// Secrets must remain redacted in JSON mode.
	for _, leak := range []string{"plain-codex-access", "plain-codex-refresh"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("auth status --json leaked %q", leak)
		}
	}

	var got struct {
		Provider      string `json:"provider"`
		Status        string `json:"status"`
		AuthType      string `json:"auth_type"`
		Authenticated bool   `json:"authenticated"`
		Redacted      bool   `json:"redacted"`
		Credentials   []struct {
			ID              string `json:"id"`
			Label           string `json:"label"`
			AuthType        string `json:"auth_type"`
			SecretsRedacted bool   `json:"secrets_redacted"`
		} `json:"credentials"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Provider != config.CodexOAuthProvider {
		t.Fatalf("got.Provider = %q, want %q", got.Provider, config.CodexOAuthProvider)
	}
	if got.Status != "logged_in" || !got.Authenticated || !got.Redacted {
		t.Fatalf("got.Status/Authenticated/Redacted = %q/%t/%t, want logged_in/true/true", got.Status, got.Authenticated, got.Redacted)
	}
	if len(got.Credentials) != 1 || got.Credentials[0].ID != "codex-device-1" || !got.Credentials[0].SecretsRedacted {
		t.Fatalf("credentials shape unexpected: %+v", got.Credentials)
	}
	// JSON mode must not interleave the human `auth_status provider=`
	// row, which would make stdout unparseable.
	if strings.Contains(stdout, "auth_status provider=") {
		t.Fatalf("--json must not emit the human row; got:\n%s", stdout)
	}
}

// TestAuthStatusCommand_JSONIncludesBuildProvenance proves
// `gormes auth status <provider> --json` carries the running binary's
// build version + SHA. Same contract as update/doctor/status/restore —
// captured credential-health snapshots stay attributable to a specific
// binary.
func TestAuthStatusCommand_JSONIncludesBuildProvenance(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, config.CodexOAuthProvider, []config.PooledCredential{
		{ID: "codex-device-1", Label: "codex", AuthType: config.CredentialAuthOAuth, Source: config.CodexOAuthSourceDeviceCode, AccessToken: "plain-codex-access", RefreshToken: "plain-codex-refresh"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "status", config.CodexOAuthProvider, "--json")
	if err != nil {
		t.Fatalf("auth status --json: %v\nstdout=%s stderr=%s", err, stdout, stderr)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version {
		t.Fatalf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Build.GitCommit == "" {
		t.Fatalf("got.build.git_commit must be non-empty")
	}
}

func TestAuthStatusCommandRejectsUnknownProvider(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "status", "not-a-provider")
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("err = %v, stdout=%s stderr=%s, want unknown provider", err, stdout, stderr)
	}
}
