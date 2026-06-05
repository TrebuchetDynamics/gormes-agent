package gormescli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runAgentTestCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newAgentCommandForTest()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestAgentResetCommandCreatesTemplatesInTarget(t *testing.T) {
	target := t.TempDir()

	stdout, stderr, err := runAgentTestCommand(t, "reset", "--target", target)
	if err != nil {
		t.Fatalf("agent reset: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"create SOUL.md",
		"create AGENTS.md",
		"create memory/USER.md",
		"create memory/MEMORY.md",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if _, err := os.Stat(filepath.Join(target, "SOUL.md")); err != nil {
		t.Fatalf("SOUL.md not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "memory", "USER.md")); err != nil {
		t.Fatalf("memory/USER.md not created: %v", err)
	}
}

func TestAgentResetCommandDryRunLeavesTargetEmpty(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "gormes-context")

	stdout, stderr, err := runAgentTestCommand(t, "reset", "--target", target, "--dry-run")
	if err != nil {
		t.Fatalf("agent reset dry-run: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "would_create SOUL.md") || !strings.Contains(stdout, "would_create memory/USER.md") {
		t.Fatalf("dry-run stdout missing would_create actions:\n%s", stdout)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("dry-run created target or returned unexpected error: %v", err)
	}
}

// TestAgentResetCommand_JSONEmitsStructuredOutcome proves
// `gormes agent reset --json` returns a parseable
// `{build, target, dry_run, files: [{path, action}]}` document so
// fleet automation seeding agent context across many machines can
// audit which template files were created/overwritten/skipped per
// machine. Same convention as the rest of the `--json` arc.
func TestAgentResetCommand_JSONEmitsStructuredOutcome(t *testing.T) {
	target := t.TempDir()

	stdout, stderr, err := runAgentTestCommand(t, "reset", "--target", target, "--json")
	if err != nil {
		t.Fatalf("agent reset --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Target string `json:"target"`
		DryRun bool   `json:"dry_run"`
		Files  []struct {
			Path   string `json:"path"`
			Action string `json:"action"`
		} `json:"files"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("agent reset --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Target != target {
		t.Errorf("target = %q, want %q", got.Target, target)
	}
	if got.DryRun {
		t.Errorf("dry_run must be false in apply mode")
	}
	if len(got.Files) < 4 {
		t.Errorf("files len = %d, want >=4 (SOUL.md, AGENTS.md, memory/USER.md, memory/MEMORY.md)", len(got.Files))
	}
	wantPaths := map[string]bool{"SOUL.md": false, "AGENTS.md": false, "memory/USER.md": false, "memory/MEMORY.md": false}
	for _, f := range got.Files {
		if _, ok := wantPaths[f.Path]; ok {
			wantPaths[f.Path] = true
			if f.Action != "create" {
				t.Errorf("%s action = %q, want %q", f.Path, f.Action, "create")
			}
		}
	}
	for path, saw := range wantPaths {
		if !saw {
			t.Errorf("files JSON missing %s; got %+v", path, got.Files)
		}
	}
}

// TestAgentResetCommand_DryRunJSONReportsWouldActions proves
// `agent reset --dry-run --json` emits actions like `would_create`
// and `dry_run: true`, leaving the target untouched on disk.
func TestAgentResetCommand_DryRunJSONReportsWouldActions(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "gormes-context")

	stdout, stderr, err := runAgentTestCommand(t, "reset", "--target", target, "--dry-run", "--json")
	if err != nil {
		t.Fatalf("agent reset --dry-run --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		DryRun bool `json:"dry_run"`
		Files  []struct {
			Path   string `json:"path"`
			Action string `json:"action"`
		} `json:"files"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("agent reset --dry-run --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if !got.DryRun {
		t.Errorf("dry_run must be true in dry-run mode")
	}
	for _, f := range got.Files {
		if !strings.HasPrefix(f.Action, "would_") {
			t.Errorf("dry-run actions must be `would_*`; got %q for %s", f.Action, f.Path)
		}
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("dry-run JSON must not create target dir; stat err=%v", err)
	}
}

func TestRootCommandIncludesAgentCommand(t *testing.T) {
	root := newAgentRootCommandForTest()
	cmd, _, err := root.Find([]string{"agent", "reset"})
	if err != nil {
		t.Fatalf("find agent reset: %v", err)
	}
	if cmd == nil || cmd.Use != "reset" {
		t.Fatalf("root command did not expose agent reset: %#v", cmd)
	}
}
