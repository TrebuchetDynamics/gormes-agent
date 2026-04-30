package main

import (
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

func TestAuthStatusCommandRejectsUnknownProvider(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "status", "not-a-provider")
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("err = %v, stdout=%s stderr=%s, want unknown provider", err, stdout, stderr)
	}
}
