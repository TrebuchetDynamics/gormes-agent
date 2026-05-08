package main

import (
	"encoding/json"
	"strings"
	"testing"
)

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
