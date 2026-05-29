package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

// TestGormesProfileInfo_JSONEmitsStructuredManifest guards an API
// consistency papercut: every other `gormes profile <subcmd>` accepts
// `--json` (`show`, `set`, `list`) but the freshly-added `info`
// subcommand was shipped without it, breaking fleet automation that
// expects a uniform `gormes profile <verb> --json` interface.
//
// Contract: `profile info <name> --json` emits a parseable document
// with `build` provenance plus `name`, `root` (redacted), and an
// optional `distribution` block when the profile carries a manifest.
func TestGormesProfileInfo_JSONEmitsStructuredManifest(t *testing.T) {
	const root = "/home/operator-secret/.config/gormes/profiles/work"
	fake := &profileCommandFakeSeams{
		knownProfiles: []string{"main", "work"},
		resolveProfileRoot: func(name string) (string, error) {
			return root, nil
		},
		distributionByRoot: map[string]cli.ProfileDistributionManifest{
			root: {
				Name:        "work-distribution",
				Version:     "1.0.0",
				Description: "Work distribution",
				Author:      "ops",
			},
		},
	}
	stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "info", "work", "--json")
	if err != nil {
		t.Fatalf("profile info --json: %v\nstdout=%s stderr=%s", err, stdout, stderr)
	}

	// Operator-secret raw path must never appear (same redaction promise
	// as `profile show --json`).
	if strings.Contains(stdout+stderr, "/home/operator-secret") {
		t.Fatalf("profile info --json leaked raw path; stdout=%s stderr=%s", stdout, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Name         string `json:"name"`
		Root         string `json:"root"`
		Distribution *struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
			Author      string `json:"author"`
		} `json:"distribution,omitempty"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version || got.Build.GitCommit == "" {
		t.Errorf("build provenance missing/wrong: %+v", got.Build)
	}
	if got.Name != "work" {
		t.Errorf("got.Name = %q, want work", got.Name)
	}
	if !strings.Contains(got.Root, "...") {
		t.Errorf("got.Root = %q, want a redacted form (`...` marker)", got.Root)
	}
	if got.Distribution == nil {
		t.Fatal("distribution block missing in JSON for a profile that has a manifest")
	}
	if got.Distribution.Name != "work-distribution" {
		t.Errorf("got.Distribution.Name = %q, want work-distribution", got.Distribution.Name)
	}
	if got.Distribution.Version != "1.0.0" {
		t.Errorf("got.Distribution.Version = %q, want 1.0.0", got.Distribution.Version)
	}
}

// TestGormesProfileInfo_JSONOmitsDistributionWhenAbsent: when no
// distribution.yaml exists for the profile, `--json` must still emit a
// parseable document, with `distribution` either nil/omitted. Mirrors
// `profile show --json`'s `omitempty` shape.
func TestGormesProfileInfo_JSONOmitsDistributionWhenAbsent(t *testing.T) {
	fake := &profileCommandFakeSeams{
		knownProfiles: []string{"main"},
		// distributionByRoot left nil → ReadDistributionManifest returns
		// hasManifest=false.
	}
	stdout, _, err := runProfileTestCommand(t, fake.defaults(), "info", "main", "--json")
	if err != nil {
		t.Fatalf("profile info default --json (no manifest): %v", err)
	}
	var got map[string]any
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if _, present := got["distribution"]; present && got["distribution"] != nil {
		t.Errorf("distribution should be omitted/nil when no manifest; got %v", got["distribution"])
	}
}
