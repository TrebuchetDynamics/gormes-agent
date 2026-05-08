package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestVersionCommand_ConstructorReturnsIndependentInstances proves
// that newVersionCommand() returns a fresh cobra.Command each call
// with isolated flag state. Without a constructor, the package-level
// versionCmd var would let an earlier `--json` flag set on one
// instance leak into a later non-JSON invocation — same anti-pattern
// that broke `doctorCmd` before its constructor refactor.
func TestVersionCommand_ConstructorReturnsIndependentInstances(t *testing.T) {
	a := newVersionCommand()
	b := newVersionCommand()
	if a == b {
		t.Fatal("newVersionCommand must return distinct instances; got the same pointer")
	}
	// Set --json on `a` then run `b` with default args; `b` must
	// emit the human format, not JSON.
	a.SetArgs([]string{"--json"})
	_ = a.Execute()

	var stdoutB strings.Builder
	b.SetOut(&stdoutB)
	b.SetArgs(nil)
	_ = b.Execute()
	if !strings.HasPrefix(strings.TrimSpace(stdoutB.String()), "gormes ") {
		t.Fatalf("instance B (no --json) must emit human format; got %q", stdoutB.String())
	}
}

// TestVersionCommand_HumanFormat is the regression baseline for the
// existing default `gormes version` output. Refactoring to add --json
// must not change the human-readable line.
func TestVersionCommand_HumanFormat(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	root := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeRootCommandForTest(root, "version")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	want := "gormes " + Version
	if strings.TrimSpace(stdout) != want {
		t.Fatalf("default version output = %q, want %q", strings.TrimSpace(stdout), want)
	}
}

// TestVersionCommand_JSONIncludesGitCommitField proves --json carries
// a git_commit field. Defaults to "unknown" in dev/source builds; CI
// release builds are expected to inject the build-time SHA via ldflags
// (-X main.GitCommit=<sha>) in a follow-up CI slice. Fleet automation
// verifying binaries against a specific commit reads this field.
func TestVersionCommand_JSONIncludesGitCommitField(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	root := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeRootCommandForTest(root, "version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var got struct {
		GitCommit string `json:"git_commit"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.GitCommit == "" {
		t.Fatalf("git_commit must be non-empty (`unknown` in dev builds is acceptable); got %q", stdout)
	}
}

// TestVersionCommand_JSONIncludesSemverAndDateAlias proves --json emits
// both the canonical semver and the Hermes-style vYYYY.M.D date alias.
// Fleet automation that tracks Gormes deployments across operators
// needs the alias to compare against Hermes upstream baselines (whose
// own version IS the date).
func TestVersionCommand_JSONIncludesSemverAndDateAlias(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	root := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeRootCommandForTest(root, "version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var got struct {
		Version    string `json:"version"`
		DateAlias  string `json:"date_alias"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Version != Version {
		t.Fatalf("got.Version = %q, want %q (matching the package-level Version)", got.Version, Version)
	}
	// Hermes-style date alias must follow vYYYY.M.D shape and be
	// non-empty so fleet automation can rely on it.
	if !strings.HasPrefix(got.DateAlias, "v") || strings.Count(got.DateAlias, ".") != 2 {
		t.Fatalf("got.DateAlias = %q, want format vYYYY.M.D (e.g. `v2026.5.7`)", got.DateAlias)
	}
}
