package gormescli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func testVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		DateAlias: "v2026.6.5",
		GitCommit: "test-git",
		GitDirty:  "false",
		BuildDate: "unknown",
	}
}

// TestVersionCommand_ConstructorReturnsIndependentInstances proves
// that NewVersionCommand returns a fresh cobra.Command each call
// with isolated flag state. Without a constructor, a package-level
// command var could let an earlier --json flag set on one instance
// leak into a later non-JSON invocation.
func TestVersionCommand_ConstructorReturnsIndependentInstances(t *testing.T) {
	a := NewVersionCommand(testVersionInfo())
	b := NewVersionCommand(testVersionInfo())
	if a == b {
		t.Fatal("NewVersionCommand must return distinct instances; got the same pointer")
	}

	var stdoutA strings.Builder
	a.SetOut(&stdoutA)
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

func TestRootCommand_VersionFlagPrintsVersion(t *testing.T) {
	for _, flag := range []string{"--version", "-V"} {
		t.Run(flag, func(t *testing.T) {
			root := NewRootCommand(RootOptions{Version: Version}, stubRootFactories())
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

func TestVersionCommand_HumanFormat(t *testing.T) {
	stdout := executeVersionRootCommandForTest(t, testVersionInfo())
	want := "gormes " + Version
	if strings.TrimSpace(stdout) != want {
		t.Fatalf("default version output = %q, want %q", strings.TrimSpace(stdout), want)
	}
}

func TestVersionCommand_HumanFormatMarksDirtyBuild(t *testing.T) {
	info := testVersionInfo()
	info.GitDirty = "true"
	stdout := executeVersionRootCommandForTest(t, info)
	got := strings.TrimSpace(stdout)
	want := "gormes " + Version + " (dirty)"
	if got != want {
		t.Fatalf("dirty version output = %q, want %q", got, want)
	}
}

func TestVersionCommand_JSONIncludesPlatformFields(t *testing.T) {
	stdout := executeVersionCommandJSONForTest(t)
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

func TestVersionCommand_JSONIncludesGoVersionField(t *testing.T) {
	stdout := executeVersionCommandJSONForTest(t)
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

func TestVersionCommand_JSONIncludesGitDirtyField(t *testing.T) {
	stdout := executeVersionCommandJSONForTest(t)
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

func TestVersionCommand_JSONIncludesBuildDateField(t *testing.T) {
	info := testVersionInfo()
	info.BuildDate = "2026-05-09T12:34:56Z"
	stdout := executeVersionRootCommandForTest(t, info, "--json")
	var got struct {
		BuildDate string `json:"build_date"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.BuildDate != info.BuildDate {
		t.Fatalf("build_date = %q, want %q", got.BuildDate, info.BuildDate)
	}
}

func TestVersionCommand_JSONIncludesGitCommitField(t *testing.T) {
	stdout := executeVersionCommandJSONForTest(t)
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

func TestVersionCommand_JSONIncludesSemverAndDateAlias(t *testing.T) {
	stdout := executeVersionCommandJSONForTest(t)
	var got struct {
		Version   string `json:"version"`
		DateAlias string `json:"date_alias"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Version != Version {
		t.Fatalf("got.Version = %q, want %q (matching the injected version)", got.Version, Version)
	}
	if !strings.HasPrefix(got.DateAlias, "v") || strings.Count(got.DateAlias, ".") != 2 {
		t.Fatalf("got.DateAlias = %q, want format vYYYY.M.D (e.g. `v2026.5.7`)", got.DateAlias)
	}
}

func executeVersionCommandJSONForTest(t *testing.T) string {
	t.Helper()
	return executeVersionRootCommandForTest(t, testVersionInfo(), "--json")
}

func executeVersionRootCommandForTest(t *testing.T, info VersionInfo, args ...string) string {
	t.Helper()
	root := newRootCommandWithFactoriesForTest(map[string]func() *cobra.Command{
		"version": func() *cobra.Command { return NewVersionCommand(info) },
	})
	stdout, stderr, err := executeRootCommandForTest(root, append([]string{"version"}, args...)...)
	if err != nil {
		t.Fatalf("version %s: %v\nstdout=%s\nstderr=%s", strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout
}

func versionMapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
