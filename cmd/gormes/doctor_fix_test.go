package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/doctor"
)

// `gormes doctor --fix` auto-remediates the config-version class: an
// on-disk config missing the `_config_version` stamp is migrated to the
// current schema via the real config.MigrateConfigFile seam, and the run
// reports the fix in Gormes-owned wording. `--offline` keeps the provider
// health network check SKIPped, and the config migrate is a pure local
// file op (no network remediation under --offline).
func TestDoctorFixMigratesUnstampedConfigAndReportsGormesWording(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "https://example.test/v1"
model = "configured-model"
api_key = "sk-doctor-fix-test"
`))

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--fix"})
	_ = cmd.Execute()

	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "Auto-fix results:") {
		t.Fatalf("doctor --fix must print an auto-fix results block:\n%s", out)
	}
	if !strings.Contains(out, "✓ Fixed: config schema") {
		t.Fatalf("doctor --fix must report the config-version remediation as Fixed:\n%s", out)
	}
	if !strings.Contains(out, "skipped (--offline)") {
		t.Fatalf("doctor --offline --fix must still SKIP the provider health network check:\n%s", out)
	}
	for _, forbidden := range []string{"hermes doctor", "hermes setup", "~/.hermes", "/.hermes"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("doctor --fix leaked Hermes-owned wording %q (must be Gormes-owned):\n%s", forbidden, out)
		}
	}

	body, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read migrated config: %v", err)
	}
	if !strings.Contains(string(body), "_config_version") {
		t.Fatalf("doctor --fix must have written the _config_version stamp to config.toml, got:\n%s", string(body))
	}
}

// Running `gormes doctor --fix` a second time is idempotent: a config
// already at the current schema is reported as already-current, not
// re-"Fixed" (computed from MigrateConfigFile's real NoOp result).
func TestDoctorFixIsIdempotentOnAlreadyCurrentConfig(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
_config_version = 1

[hermes]
endpoint = "https://example.test/v1"
model = "configured-model"
api_key = "sk-doctor-fix-test"
`))

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--fix"})
	_ = cmd.Execute()

	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "config schema: already current") {
		t.Fatalf("already-current config must be reported as already current, not re-Fixed:\n%s", out)
	}
	if strings.Contains(out, "✓ Fixed: config schema") {
		t.Fatalf("idempotent run must not claim a config schema fix:\n%s", out)
	}
}

// The human doctor report must end with the computed issues summary
// (parity with Hermes doctor.py run_doctor end-of-report), in Gormes wording.
func TestDoctorReporterHumanFinalizePrintsIssuesSummary(t *testing.T) {
	var buf bytes.Buffer
	r := &doctorReporter{w: &buf}
	r.Add(doctor.CheckResult{Name: "build identity", Status: doctor.StatusPass, Summary: "version=0.2.12"})
	r.Add(doctor.CheckResult{Name: "config schema", Status: doctor.StatusFail, Summary: "config schema version is behind; run migrate"})
	r.Add(doctor.CheckResult{Name: "Gateway Slack", Status: doctor.StatusWarn, Summary: "disabled"})
	if err := r.Finalize(); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Found 2 issue(s) to address:") {
		t.Fatalf("human report missing computed issues summary:\n%s", out)
	}
	if !strings.Contains(out, "`gormes doctor --fix`") {
		t.Fatalf("human report missing gormes doctor --fix tip (config schema is fixable):\n%s", out)
	}
	for _, forbidden := range []string{"hermes doctor", "hermes setup", "~/.hermes"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("human report leaked Hermes-owned wording %q:\n%s", forbidden, out)
		}
	}
}

func TestDoctorReporterHumanFinalizeCleanWhenNoIssues(t *testing.T) {
	var buf bytes.Buffer
	r := &doctorReporter{w: &buf}
	r.Add(doctor.CheckResult{Name: "build identity", Status: doctor.StatusPass, Summary: "version=0.2.12"})
	r.Add(doctor.CheckResult{Name: "provider health", Status: doctor.StatusSkip, Summary: "skipped (--offline)"})
	if err := r.Finalize(); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "Found") || strings.Contains(out, "gormes doctor --fix") {
		t.Fatalf("clean run must not print a Found-N list or --fix tip:\n%s", out)
	}
	if !strings.Contains(out, "No issues") {
		t.Fatalf("clean run must print a no-issues line:\n%s", out)
	}
}

// JSON mode is unchanged by this slice (row scope: human end-of-report only).
func TestDoctorReporterJSONModeUnchanged(t *testing.T) {
	var buf bytes.Buffer
	r := &doctorReporter{w: &buf, asJSON: true}
	r.Add(doctor.CheckResult{Name: "Gateway Slack", Status: doctor.StatusWarn, Summary: "disabled"})
	if err := r.Finalize(); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"checks"`) {
		t.Fatalf("JSON mode must still emit the checks document:\n%s", out)
	}
	if strings.Contains(out, "Found 1 issue(s) to address:") {
		t.Fatalf("JSON mode must not gain the human summary block:\n%s", out)
	}
}
