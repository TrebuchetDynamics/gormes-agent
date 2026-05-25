package doctor

import (
	"strings"
	"testing"
)

func TestDoctorIssuesSummaryComputesActionableCountAndGormesWording(t *testing.T) {
	results := []CheckResult{
		{Name: "build identity", Status: StatusWarn, Summary: "dirty build"},
		{Name: "GitHub auth", Status: StatusWarn, Summary: "No GITHUB_TOKEN and gh auth status failed"},
		{Name: "Gateway Slack", Status: StatusWarn, Summary: "disabled"},
		{Name: "config schema", Status: StatusWarn, Summary: "config schema version is behind; run migrate"},
		{Name: "provider setup", Status: StatusFail, Summary: "endpoint unconfigured"},
		{Name: "provider health", Status: StatusSkip, Summary: "skipped (--offline)"},
	}

	issues := CollectDoctorIssues(results)
	// Hermes doctor keeps optional WARN checks visible in their sections without
	// automatically appending them to the final issues list. Only the actionable
	// config-schema warning and provider failure enter the computed count.
	if len(issues) != 2 {
		t.Fatalf("CollectDoctorIssues = %d, want 2 actionable issues: %+v", len(issues), issues)
	}
	for _, is := range issues {
		for _, nonActionable := range []string{"build identity", "GitHub auth", "Gateway Slack"} {
			if is.Name == nonActionable {
				t.Fatalf("non-actionable warning %q leaked into issues: %+v", nonActionable, issues)
			}
		}
	}

	out := RenderDoctorIssuesSummary(issues)
	if !strings.Contains(out, "Found 2 issue(s) to address:") {
		t.Fatalf("summary missing computed count line, got:\n%s", out)
	}
	for i, want := range []string{"1.", "2."} {
		if !strings.Contains(out, want) {
			t.Fatalf("summary missing numbered item %d (%q):\n%s", i+1, want, out)
		}
	}
	// At least one fixable class (config schema/version) -> Gormes-owned --fix tip.
	if !strings.Contains(out, `gormes doctor --fix`) {
		t.Fatalf("summary missing `gormes doctor --fix` tip when a fixable issue exists:\n%s", out)
	}
	// Owned divergence: never Hermes wording/paths.
	for _, forbidden := range []string{"hermes doctor", "hermes setup", "~/.hermes", "/.hermes"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("summary leaked Hermes-owned wording %q (must be Gormes-owned):\n%s", forbidden, out)
		}
	}
}

func TestDoctorIssuesSummaryZeroIssuesNoFixTip(t *testing.T) {
	results := []CheckResult{
		{Name: "build identity", Status: StatusPass, Summary: "version=0.2.12"},
		{Name: "provider health", Status: StatusSkip, Summary: "skipped (--offline)"},
	}
	issues := CollectDoctorIssues(results)
	if len(issues) != 0 {
		t.Fatalf("CollectDoctorIssues = %d, want 0 for all PASS/SKIP", len(issues))
	}
	out := RenderDoctorIssuesSummary(issues)
	if strings.Contains(out, "Found") || strings.Contains(out, "gormes doctor --fix") {
		t.Fatalf("zero-issue summary must not print a Found-N list or --fix tip, got:\n%s", out)
	}
	if !strings.Contains(out, "No issues") {
		t.Fatalf("zero-issue summary must print a clean no-issues line, got:\n%s", out)
	}
}

func TestDoctorIssuesSummaryNoFixTipWhenNoFixableIssue(t *testing.T) {
	results := []CheckResult{
		{Name: "provider setup", Status: StatusFail, Summary: "endpoint unconfigured"},
	}
	out := RenderDoctorIssuesSummary(CollectDoctorIssues(results))
	if !strings.Contains(out, "Found 1 issue(s) to address:") {
		t.Fatalf("expected 1-issue summary, got:\n%s", out)
	}
	if strings.Contains(out, "gormes doctor --fix") {
		t.Fatalf("no auto-fixable issue present; --fix tip must be omitted, got:\n%s", out)
	}
}

// `gormes doctor --fix` attempts every source-backed auto-fixable class via
// the injected applier (config-version migrate), then reports computed
// fixed-vs-still-manual results in Gormes wording. The applier seam is
// injected so this package stays dependency-light and the orchestration is
// hermetically testable (no config/filesystem here).
func TestRunDoctorFixAppliesEveryAutoFixableClassAndReportsComputed(t *testing.T) {
	classes := AutoFixableClasses()
	if len(classes) == 0 {
		t.Fatalf("AutoFixableClasses must enumerate >=1 source-backed class")
	}
	seen := map[string]bool{}
	outcomes := RunDoctorFix(func(class string) DoctorFixOutcome {
		seen[class] = true
		if class == doctorFixClassConfigVersion {
			return DoctorFixOutcome{Class: class, Name: "config schema", Fixed: true, Detail: "migrated _config_version v0→v1"}
		}
		return DoctorFixOutcome{Class: class, Name: class, AlreadyOK: true}
	})
	for _, c := range classes {
		if !seen[c] {
			t.Fatalf("RunDoctorFix did not apply class %q", c)
		}
	}

	// Residual manual issues are the non-auto-fixable actionable findings.
	manual := CollectDoctorIssues([]CheckResult{
		{Name: "provider setup", Status: StatusFail, Summary: "endpoint unconfigured"},
	})
	out := RenderDoctorFixReport(outcomes, manual)
	if !strings.Contains(out, "✓ Fixed: config schema — migrated _config_version v0→v1") {
		t.Fatalf("fix report missing computed Fixed line:\n%s", out)
	}
	if !strings.Contains(out, "Still manual (1):") {
		t.Fatalf("fix report missing computed still-manual count:\n%s", out)
	}
	if !strings.Contains(out, "provider setup") {
		t.Fatalf("fix report must list the residual manual issue:\n%s", out)
	}
	for _, forbidden := range []string{"hermes doctor", "hermes setup", "~/.hermes", "/.hermes"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("fix report leaked Hermes-owned wording %q (must be Gormes-owned):\n%s", forbidden, out)
		}
	}
}

func TestRenderDoctorFixReportNothingToDo(t *testing.T) {
	outcomes := []DoctorFixOutcome{
		{Class: doctorFixClassConfigVersion, Name: "config schema", AlreadyOK: true},
	}
	out := RenderDoctorFixReport(outcomes, nil)
	if !strings.Contains(out, "config schema") || !strings.Contains(out, "already current") {
		t.Fatalf("already-current outcome must be reported, not dropped:\n%s", out)
	}
	if strings.Contains(out, "✓ Fixed:") || strings.Contains(out, "Still manual") {
		t.Fatalf("no fixes applied and no residual manual issues; report must not claim either:\n%s", out)
	}
}
