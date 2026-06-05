package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

func TestDoctorCommandRendersSkillsHubSectionOffline(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	hub := filepath.Join(config.GormesHome(), "skills", ".hub")
	if err := os.MkdirAll(hub, 0o755); err != nil {
		t.Fatalf("mkdir skills hub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hub, "lock.json"), []byte(`{"installed":{"skill-a":{},"skill-b":{}}}`), 0o644); err != nil {
		t.Fatalf("write lock.json: %v", err)
	}

	called := false
	prevRunner := doctorGitHubAuthRunner
	doctorGitHubAuthRunner = func(context.Context) doctor.GitHubAuthStatusResult {
		called = true
		t.Fatalf("doctor --offline must not run gh auth status for Skills Hub")
		return doctor.GitHubAuthStatusResult{}
	}
	t.Cleanup(func() { doctorGitHubAuthRunner = prevRunner })

	stdout, stderr, err := executeRootCommandForTest(newRootCommand(), "doctor", "--offline")
	if err != nil {
		t.Fatalf("doctor --offline: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if called {
		t.Fatalf("gh auth runner was called in offline mode")
	}
	out := stdout + stderr
	for _, want := range []string{
		"◆ Skills Hub",
		"✓ Skills Hub ready",
		".hub",
		"2 hub-installed skill(s)",
		"skipped (--offline",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{"~/.hermes", "hermes skills"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("Skills Hub output leaked forbidden text %q:\n%s", forbidden, out)
		}
	}
}
