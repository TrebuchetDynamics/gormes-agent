package doctor

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDoctorGitHubAuthTokenEnvPasses(t *testing.T) {
	called := false
	got := CheckGitHubAuth(context.Background(), GitHubAuthOptions{
		Env: map[string]string{"GITHUB_TOKEN": "ghp_secret_token"},
		RunGHAuthStatus: func(context.Context) GitHubAuthStatusResult {
			called = true
			return GitHubAuthStatusResult{ExitCode: 1}
		},
	})

	if called {
		t.Fatalf("gh runner was called despite GITHUB_TOKEN")
	}
	if got.Status != StatusPass {
		t.Fatalf("Status = %v, want PASS: %+v", got.Status, got)
	}
	out := got.Format()
	if !strings.Contains(out, "github_token_env") {
		t.Fatalf("output missing env evidence:\n%s", out)
	}
	if strings.Contains(out, "ghp_secret_token") {
		t.Fatalf("output leaked token:\n%s", out)
	}
}

func TestDoctorGitHubAuthFallsBackToGHCLI(t *testing.T) {
	got := CheckGitHubAuth(context.Background(), GitHubAuthOptions{
		Env: map[string]string{},
		RunGHAuthStatus: func(context.Context) GitHubAuthStatusResult {
			return GitHubAuthStatusResult{ExitCode: 0}
		},
	})

	if got.Status != StatusPass {
		t.Fatalf("Status = %v, want PASS: %+v", got.Status, got)
	}
	out := got.Format()
	for _, want := range []string{"GitHub auth", "gh CLI", "github_cli_authenticated"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestDoctorGitHubAuthGHCLIUnauthenticatedWarns(t *testing.T) {
	tests := []struct {
		name string
		run  GitHubAuthStatusRunner
		want string
	}{
		{
			name: "missing",
			run: func(context.Context) GitHubAuthStatusResult {
				return GitHubAuthStatusResult{Err: ErrGitHubCLIMissing}
			},
			want: "github_cli_missing",
		},
		{
			name: "nonzero",
			run: func(context.Context) GitHubAuthStatusResult {
				return GitHubAuthStatusResult{ExitCode: 1}
			},
			want: "github_cli_unauthenticated",
		},
		{
			name: "timeout",
			run: func(context.Context) GitHubAuthStatusResult {
				return GitHubAuthStatusResult{TimedOut: true, Err: context.DeadlineExceeded}
			},
			want: "github_cli_timeout",
		},
		{
			name: "command failure",
			run: func(context.Context) GitHubAuthStatusResult {
				return GitHubAuthStatusResult{Err: errors.New("raw secret stderr should not leak")}
			},
			want: "github_cli_status_failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckGitHubAuth(context.Background(), GitHubAuthOptions{
				Env:             map[string]string{},
				RunGHAuthStatus: tc.run,
			})
			if got.Status != StatusWarn {
				t.Fatalf("Status = %v, want WARN: %+v", got.Status, got)
			}
			out := got.Format()
			if !strings.Contains(out, tc.want) {
				t.Fatalf("output missing evidence %q:\n%s", tc.want, out)
			}
			if strings.Contains(out, "raw secret stderr") {
				t.Fatalf("output leaked command failure details:\n%s", out)
			}
		})
	}
}
