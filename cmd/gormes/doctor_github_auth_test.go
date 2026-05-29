package main

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

func TestDoctorCommandRendersGitHubAuthFallback(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	writeOneshotFlagConfig(t, []byte(`
[hermes]
provider = "openai-codex"
model = "gpt-5.1-codex"
`))
	if _, err := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{}).SaveTokens(config.CodexOAuthTokens{
		AccessToken:  "codex-access",
		RefreshToken: "codex-refresh",
	}); err != nil {
		t.Fatalf("save codex auth: %v", err)
	}

	orig := doctorGitHubAuthRunner
	doctorGitHubAuthRunner = func(context.Context) doctor.GitHubAuthStatusResult {
		return doctor.GitHubAuthStatusResult{ExitCode: 0}
	}
	t.Cleanup(func() { doctorGitHubAuthRunner = orig })

	stdout, err := captureDoctorStdout(t, func() error {
		cmd := newRootCommand()
		cmd.SetArgs([]string{"doctor"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s", err, stdout)
	}

	for _, want := range []string{"✓ GitHub auth —", "gh CLI", "github_cli_authenticated"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, leak := range []string{"GITHUB_TOKEN=", "GH_TOKEN=", "raw secret stderr"} {
		if strings.Contains(stdout, leak) {
			t.Fatalf("doctor output leaked %q:\n%s", leak, stdout)
		}
	}
}
