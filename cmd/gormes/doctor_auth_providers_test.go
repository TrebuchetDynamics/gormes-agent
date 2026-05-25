package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestDoctorCommandRendersAuthProvidersSection(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	if err := os.MkdirAll(config.GormesHome(), 0o755); err != nil {
		t.Fatalf("create GORMES_HOME: %v", err)
	}
	writeOneshotFlagConfig(t, []byte(`
[hermes]
provider = "openai-codex"
endpoint = "https://chatgpt.com/backend-api/codex"
model = "gpt-5.5"
`))
	if _, err := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{}).SaveTokens(config.CodexOAuthTokens{
		AccountID:    "codex-account",
		Label:        "Codex Account",
		AccessToken:  "plain-codex-access",
		RefreshToken: "plain-codex-refresh",
	}); err != nil {
		t.Fatalf("save codex tokens: %v", err)
	}
	if _, err := config.SaveNousOAuthCredentials(config.CredentialPoolOptions{}, config.NousOAuthCredentials{
		Label:            "Nous Account",
		InferenceBaseURL: "https://inference-api.nousresearch.com/v1",
		AccessToken:      "plain-nous-access",
		RefreshToken:     "plain-nous-refresh",
	}); err != nil {
		t.Fatalf("save nous tokens: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommand(), "doctor", "--offline")
	if err != nil {
		t.Fatalf("doctor --offline: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	out := stdout + stderr
	for _, want := range []string{
		"◆ Auth Providers",
		"✓ provider auth ready",
		"OpenAI Codex",
		"openai-codex",
		"logged in",
		"Nous Portal",
		"nous",
		"Custom endpoint",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, out)
		}
	}
	for _, leak := range []string{
		"plain-codex-access",
		"plain-codex-refresh",
		"plain-nous-access",
		"plain-nous-refresh",
		filepath.Join(config.GormesHome(), "auth.json"),
		"~/.hermes",
		"hermes auth",
	} {
		if strings.Contains(out, leak) {
			t.Fatalf("doctor Auth Providers leaked %q:\n%s", leak, out)
		}
	}
}
