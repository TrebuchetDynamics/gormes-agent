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

// TestRootCommand_VersionFlagPrintsVersion proves `gormes --version`
// (and `-V` shorthand) print the binary's version line, matching the
// near-universal CLI convention. Without this, operators running
// `gormes --version` get an "unknown flag" error instead of the
// version they're checking for. The command-form `gormes version`
// already works; the flag-form is just a convention bridge.
func TestRootCommand_VersionFlagPrintsVersion(t *testing.T) {
	for _, flag := range []string{"--version", "-V"} {
		t.Run(flag, func(t *testing.T) {
			setupOneshotFlagTestEnv(t)
			root := newRootCommandWithRuntime(rootRuntime{})
			stdout, _, err := executeRootCommandForTest(root, flag)
			if err != nil {
				t.Fatalf("%s: %v\nstdout=%s", flag, err, stdout)
			}
			if !strings.Contains(stdout, Version) {
				t.Fatalf("%s output must mention version %q; got %q", flag, Version, stdout)
			}
		})
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

// TestVersionCommand_HumanFormatMarksDirtyBuild proves the human
// `gormes version` line appends a `(dirty)` marker when the binary was
// built from a source tree with uncommitted changes. Operators in a
// terminal need an at-a-glance signal that they're not running a clean
// release — a dirty marker matches `git describe --dirty` convention
// and avoids forcing operators to run `version --json` to see the bit.
func TestVersionCommand_HumanFormatMarksDirtyBuild(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := GitDirty
	GitDirty = "true"
	t.Cleanup(func() { GitDirty = prev })

	root := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeRootCommandForTest(root, "version")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	got := strings.TrimSpace(stdout)
	want := "gormes " + Version + " (dirty)"
	if got != want {
		t.Fatalf("dirty version output = %q, want %q", got, want)
	}
}

// TestVersionCommand_JSONIncludesPlatformFields proves --json
// surfaces the binary's target os/arch (sourced from runtime.GOOS /
// runtime.GOARCH). Fleet inventory matching gormes deployments
// against the 6-platform release matrix uses this — same
// {linux,darwin,windows} × {amd64,arm64} grid as the release
// workflow.
func TestVersionCommand_JSONIncludesPlatformFields(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	root := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeRootCommandForTest(root, "version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var got struct {
		OS   string `json:"os"`
		Arch string `json:"arch"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.OS == "" {
		t.Fatalf("`os` must be non-empty (matches runtime.GOOS); got %q", stdout)
	}
	if got.Arch == "" {
		t.Fatalf("`arch` must be non-empty (matches runtime.GOARCH); got %q", stdout)
	}
}

// TestVersionCommand_JSONIncludesGoVersionField proves --json
// surfaces the Go runtime version the binary was compiled with. Fleet
// auditors checking for stale toolchain versions (e.g. CVEs in
// older Go releases) need this programmatically. Sourced from
// runtime.Version() so no ldflags injection is required.
func TestVersionCommand_JSONIncludesGoVersionField(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	root := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeRootCommandForTest(root, "version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var got struct {
		GoVersion string `json:"go_version"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if !strings.HasPrefix(got.GoVersion, "go1.") {
		t.Fatalf("go_version must look like `go1.X.Y` (matches runtime.Version() format); got %q", got.GoVersion)
	}
}

// TestVersionCommand_JSONIncludesGitDirtyField proves --json carries
// a git_dirty boolean. Defaults to false in dev/source builds; CI is
// expected to inject the actual dirty-tree flag via ldflags so fleet
// auditors can distinguish reproducible release builds from
// hand-tweaked or unreleased binaries.
func TestVersionCommand_JSONIncludesGitDirtyField(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	root := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeRootCommandForTest(root, "version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var raw map[string]any
	if jsonErr := json.Unmarshal([]byte(stdout), &raw); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	val, ok := raw["git_dirty"]
	if !ok {
		t.Fatalf("JSON must include `git_dirty` boolean; got keys=%v", versionMapKeys(raw))
	}
	if _, ok := val.(bool); !ok {
		t.Fatalf("`git_dirty` must be a bool; got %T %v", val, val)
	}
}

// TestVersionCommand_JSONIncludesBuildDateField proves --json carries
// binary build-date provenance. Release CI injects this via ldflags, while
// source builds fall back to Go VCS build-info time or "unknown".
func TestVersionCommand_JSONIncludesBuildDateField(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := BuildDate
	BuildDate = "2026-05-09T12:34:56Z"
	t.Cleanup(func() { BuildDate = prev })

	root := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeRootCommandForTest(root, "version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var got struct {
		BuildDate string `json:"build_date"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.BuildDate != BuildDate {
		t.Fatalf("build_date = %q, want %q", got.BuildDate, BuildDate)
	}
}

func versionMapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
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
		Version   string `json:"version"`
		DateAlias string `json:"date_alias"`
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
