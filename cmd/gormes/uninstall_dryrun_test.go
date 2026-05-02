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

func TestUninstallDryRunOutputIsGroupedAndStable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]"), 0o644)

	// First run
	groups1 := collectArtifacts(home)
	var sb1 strings.Builder
	printDryRunTo(&sb1, groups1)

	// Second run should produce identical output
	groups2 := collectArtifacts(home)
	var sb2 strings.Builder
	printDryRunTo(&sb2, groups2)

	if sb1.String() != sb2.String() {
		t.Errorf("dry-run output not byte-stable between runs")
	}
}

func printDryRunTo(sb *strings.Builder, groups []artifactGroup) {
	total := 0
	for _, g := range groups {
		total += len(g.Paths)
	}
	for _, g := range groups {
		sb.WriteString("[" + g.Name + "]\n")
		for _, p := range g.Paths {
			sb.WriteString("  " + p + "\n")
		}
	}
}
