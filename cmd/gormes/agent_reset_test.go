package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runAgentTestCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newAgentCommand()
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

func TestRootCommandIncludesAgentCommand(t *testing.T) {
	root := newRootCommandWithRuntime(rootRuntime{})
	cmd, _, err := root.Find([]string{"agent", "reset"})
	if err != nil {
		t.Fatalf("find agent reset: %v", err)
	}
	if cmd == nil || cmd.Use != "reset" {
		t.Fatalf("root command did not expose agent reset: %#v", cmd)
	}
}
