package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUninstallDryRunDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	// Create some Gormes artifacts
	os.MkdirAll(home, 0o755)
	os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]"), 0o644)
	os.WriteFile(filepath.Join(home, "auth.json"), []byte(`{}`), 0o644)
	os.WriteFile(filepath.Join(home, "gateway_state.json"), []byte(`{}`), 0o644)

	groups := collectArtifacts(home)

	total := 0
	for _, g := range groups {
		total += len(g.Paths)
	}
	if total < 3 {
		t.Fatalf("expected at least 3 artifact paths, got %d", total)
	}

	// Verify groups are present
	names := make(map[string]bool)
	for _, g := range groups {
		if len(g.Paths) > 0 {
			names[g.Name] = true
		}
	}
	if !names["config"] || !names["credentials"] || !names["gateway-state"] {
		t.Fatalf("expected config, credentials, gateway-state groups: %v", names)
	}

	// Verify paths are sorted
	for _, g := range groups {
		for i := 1; i < len(g.Paths); i++ {
			if g.Paths[i] < g.Paths[i-1] {
				t.Errorf("%s paths not sorted: %s < %s", g.Name, g.Paths[i], g.Paths[i-1])
			}
		}
	}
}

func TestUninstallKeepConfigExcludesGroup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]"), 0o644)
	os.WriteFile(filepath.Join(home, "auth.json"), []byte(`{}`), 0o644)

	groups := collectArtifacts(home)
	groups = removeGroup(groups, "config")

	for _, g := range groups {
		if g.Name == "config" {
			t.Fatal("config group should be excluded")
		}
	}
}

// TestUninstallCommand_DryRunRoutesThroughCobraWriters proves the
// command writes its dry-run output via cmd.OutOrStdout() so end-to-end
// tests can capture stdout instead of forking through helpers like
// printDryRunTo. Without this, runUninstall's fmt.Printf goes to
// os.Stdout directly and bypasses cobra's writer plumbing — meaning
// the test fixture in this file had to copy-paste a parallel
// `printDryRunTo` helper to verify formatting. The refactor matches
// the same testability pattern doctor adopted (commit 39ff073db).
func TestUninstallCommand_DryRunRoutesThroughCobraWriters(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newUninstallCommand()
	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("uninstall: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	got := stdout.String()
	if got == "" {
		t.Fatalf("uninstall must write to cmd.OutOrStdout(); captured stdout is empty (output likely went to os.Stdout)")
	}
	for _, want := range []string{"uninstall dry-run:", "[config]", "config.toml"} {
		if !strings.Contains(got, want) {
			t.Fatalf("uninstall stdout missing %q:\n%s", want, got)
		}
	}
}

func TestUninstallDryRunOutputIsGroupedAndStable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]"), 0o644)

	// First run — exercise the real printDryRun, not a copy-paste helper.
	groups1 := collectArtifacts(home)
	var sb1 strings.Builder
	if err := printDryRun(&sb1, groups1); err != nil {
		t.Fatalf("printDryRun: %v", err)
	}

	// Second run should produce identical output.
	groups2 := collectArtifacts(home)
	var sb2 strings.Builder
	if err := printDryRun(&sb2, groups2); err != nil {
		t.Fatalf("printDryRun: %v", err)
	}

	if sb1.String() != sb2.String() {
		t.Errorf("dry-run output not byte-stable between runs")
	}
}
