package gormescli

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestOnboardWizard_JSONEmitsStructuredPlan pins the internal onboarding
// seam's wizard JSON shape. Setup integrations that render the same
// step-ladder the human surface prints need the wizard plan structure
// instead of scraping numbered text rows.
//
// Contract: `--wizard --json` must emit a parseable
// `{build, mode, steps: [{id, title, status, detail, next_command,
// skip_warning}, ...]}` document. The plain `--json` snapshot path
// stays unchanged for callers that don't ask for the wizard plan.
func TestOnboardWizard_JSONEmitsStructuredPlan(t *testing.T) {
	setupOnboardWizardTestEnv(t)

	stdout, stderr, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{}, "--wizard", "--json", "--non-interactive")
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
// agents, ...}` — the existing internal onboardStatusReportJSON shape.
// Adding the wizard JSON path must not double up the surface.
func TestOnboard_JSONWithoutWizardFlagStaysSnapshotShape(t *testing.T) {
	setupOnboardWizardTestEnv(t)

	stdout, _, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{}, "--json")
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
	setupOnboardWizardTestEnv(t)

	stdout, stderr, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{}, "--json")
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
	setupOnboardWizardTestEnv(t)
	t.Setenv("GORMES_PROVIDER_SECRET", secret)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "https://provider.example/v1"
model = "fixture-model"

[hermes.api_key_ref]
source = "env"
id = "GORMES_PROVIDER_SECRET"
`))

	stdout, stderr, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{}, "--json")
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
