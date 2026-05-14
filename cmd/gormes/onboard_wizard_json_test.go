package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestOnboardWizard_JSONEmitsStructuredPlan pins the regression
// observed during a fresh-install probe sweep:
// `gormes onboard --wizard --json` silently ignored the `--wizard`
// flag and emitted the same `--json` status snapshot as plain
// `gormes onboard --json`. Operators driving an interactive
// onboarding from JSON (e.g. a fleet provisioning script that
// renders the same step-ladder the human surface prints) had no way
// to ingest the wizard plan structure — they had to scrape the
// numbered text rows.
//
// Contract: `--wizard --json` must emit a parseable
// `{build, mode, steps: [{id, title, status, detail, next_command,
// skip_warning}, ...]}` document. The plain `--json` snapshot path
// stays unchanged for callers that don't ask for the wizard plan.
func TestOnboardWizard_JSONEmitsStructuredPlan(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "onboard", "--wizard", "--json", "--non-interactive")
	if err != nil {
		t.Fatalf("onboard --wizard --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Mode  string `json:"mode"`
		Steps []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Status      string `json:"status"`
			Detail      string `json:"detail"`
			NextCommand string `json:"next_command"`
			SkipWarning string `json:"skip_warning"`
		} `json:"steps"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Build.GitCommit == "" {
		t.Errorf("build.git_commit must be populated")
	}
	if got.Mode == "" {
		t.Errorf("mode must be populated; got empty")
	}
	if len(got.Steps) == 0 {
		t.Fatalf("steps must be non-empty in wizard mode; got %d", len(got.Steps))
	}
	// Spot-check that the same steps the text surface prints land in
	// JSON: a fresh install must have a Provider step in the missing
	// status, mirroring the text ladder.
	var sawProvider bool
	for _, s := range got.Steps {
		if s.Title == "Provider" {
			sawProvider = true
			if s.Status == "" || s.NextCommand == "" || s.SkipWarning == "" {
				t.Errorf("provider step is missing fields: %+v", s)
			}
		}
	}
	if !sawProvider {
		t.Fatalf("wizard plan must include the Provider step; got %+v", got.Steps)
	}
}

// TestOnboard_JSONWithoutWizardFlagStaysSnapshotShape pins the
// regression fence: callers that just want a status snapshot (no
// wizard) must continue to see `{home, config_path, provider_configured,
// agents, ...}` — the existing onboardStatusReportJSON shape.
// Adding the wizard JSON path must not double up the surface.
func TestOnboard_JSONWithoutWizardFlagStaysSnapshotShape(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, _, err := executeRootCommandForTest(cmd, "onboard", "--json")
	if err != nil {
		t.Fatalf("onboard --json: %v", err)
	}
	if !strings.Contains(stdout, `"home"`) || !strings.Contains(stdout, `"provider_configured"`) {
		t.Fatalf("plain --json must keep status-snapshot fields; got:\n%s", stdout)
	}
	if strings.Contains(stdout, `"steps"`) {
		t.Fatalf("plain --json must NOT emit wizard `steps`; got:\n%s", stdout)
	}
}

func TestOnboardJSONIncludesFirstRunReadiness(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "onboard", "--json")
	if err != nil {
		t.Fatalf("onboard --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		Home               string `json:"home"`
		ProviderConfigured bool   `json:"provider_configured"`
		FirstRun           struct {
			Ready       bool     `json:"ready"`
			Target      string   `json:"target"`
			NextCommand string   `json:"next_command"`
			Missing     []string `json:"missing"`
		} `json:"first_run"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Home == "" {
		t.Fatalf("existing snapshot field home must remain populated; stdout=%s", stdout)
	}
	if got.FirstRun.Ready {
		t.Fatalf("first_run.ready = true, want false for fresh home; stdout=%s", stdout)
	}
	if got.FirstRun.Target != "terminal" {
		t.Fatalf("first_run.target = %q, want terminal", got.FirstRun.Target)
	}
	if got.FirstRun.NextCommand != "gormes setup --quick --target terminal" {
		t.Fatalf("first_run.next_command = %q", got.FirstRun.NextCommand)
	}
	wantMissing := []string{"provider", "auth"}
	if !reflect.DeepEqual(got.FirstRun.Missing, wantMissing) {
		t.Fatalf("first_run.missing = %v, want %v", got.FirstRun.Missing, wantMissing)
	}
	if strings.Contains(stdout, "api_key") || strings.Contains(stdout, "token") {
		t.Fatalf("onboard --json must not leak secret material or secret field names:\n%s", stdout)
	}
}

func TestOnboardJSONIncludesFirstRunReadinessWithSecretRefAuth(t *testing.T) {
	secret := "sk-onboard-secretref"
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GORMES_PROVIDER_SECRET", secret)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "https://provider.example/v1"
model = "fixture-model"

[hermes.api_key_ref]
source = "env"
id = "GORMES_PROVIDER_SECRET"
`))

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "onboard", "--json")
	if err != nil {
		t.Fatalf("onboard --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		AuthConfigured bool `json:"auth_configured"`
		FirstRun       struct {
			Ready   bool     `json:"ready"`
			Target  string   `json:"target"`
			Missing []string `json:"missing"`
		} `json:"first_run"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if !got.AuthConfigured {
		t.Fatalf("auth_configured = false, want true for resolvable API key SecretRef")
	}
	if got.FirstRun.Target != "terminal" || !got.FirstRun.Ready || len(got.FirstRun.Missing) != 0 {
		t.Fatalf("first_run = %+v, want ready terminal with no missing steps", got.FirstRun)
	}
	if strings.Contains(stdout+stderr, secret) {
		t.Fatalf("onboard --json leaked resolved SecretRef value:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestOnboard_AcceptsOfflineSmokeFlag(t *testing.T) {
	for _, args := range [][]string{
		{"onboard", "--offline", "--json"},
		{"--offline", "onboard", "--json"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			t.Setenv("GORMES_HOME", t.TempDir())
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())

			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeRootCommandForTest(cmd, args...)
			if err != nil {
				t.Fatalf("%v: %v\nstdout=%s\nstderr=%s", args, err, stdout, stderr)
			}

			var got struct {
				Home string `json:"home"`
			}
			if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
				t.Fatalf("stdout must be valid onboard JSON: %v\nstdout=%s", jsonErr, stdout)
			}
			if got.Home == "" {
				t.Fatalf("onboard JSON home must be populated; stdout=%s", stdout)
			}
		})
	}
}
