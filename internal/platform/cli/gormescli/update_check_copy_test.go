package gormescli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

// runUpdateCommandWithReportAndArgs is a check-mode-aware twin of
// runUpdateCommandWithReport: it accepts cobra args so the test can pass
// `--check` to exercise the readiness-check copy path.
func runUpdateCommandWithReportAndArgs(t *testing.T, report cli.UpdateReport, args ...string) (stdout, stderr string) {
	t.Helper()
	cmd := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(_ context.Context, _ cli.UpdateLifecycleOptions) cli.UpdateReport {
			return report
		},
	})
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	_ = cmd.Execute()
	return outBuf.String(), errBuf.String()
}

// TestUpdateCommand_CheckMode_BannerSaysChecking guards a UX papercut found
// during install testing: `gormes update --check` previously printed
// "⚕ Updating Gormes Agent..." followed by "✓ Update complete!" — copy
// that suggests an update happened. `--check` is a readiness probe that
// performs no checkout mutations, so the copy must say so.
//
// Banner contract under --check:
//   - leading line says "Checking" (not "Updating")
//   - closing line says "Update check complete" (not "Update complete!")
func TestUpdateCommand_CheckMode_BannerSaysChecking(t *testing.T) {
	stdout, _ := runUpdateCommandWithReportAndArgs(t,
		cli.UpdateReport{
			Branch: "main",
			Evidence: []cli.UpdateEvidence{
				{Kind: cli.UpdateEvidenceCheck, Detail: "checked update readiness for main"},
			},
		},
		"--check",
	)
	if strings.Contains(stdout, "⚕ Updating Gormes Agent...") {
		t.Errorf("--check banner must not say 'Updating Gormes Agent...'; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Checking Gormes Agent") {
		t.Errorf("--check banner must say 'Checking Gormes Agent...'; got:\n%s", stdout)
	}
	if strings.Contains(stdout, "✓ Update complete!") {
		t.Errorf("--check footer must not say 'Update complete!'; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Update check complete") {
		t.Errorf("--check footer must say 'Update check complete'; got:\n%s", stdout)
	}
}

// TestUpdateCommand_NonCheckMode_KeepsUpdatingBanner is the regression
// fence: without --check the existing "Updating Gormes Agent..." banner
// and "✓ Update complete!" footer must remain untouched so existing
// integration tests and operator muscle memory keep working.
func TestUpdateCommand_NonCheckMode_KeepsUpdatingBanner(t *testing.T) {
	stdout, _ := runUpdateCommandWithReportAndArgs(t,
		cli.UpdateReport{
			Branch: "main",
			Evidence: []cli.UpdateEvidence{
				{Kind: cli.UpdateEvidenceAutostashCreated, Detail: "stashed local changes"},
			},
		},
		// no --check
	)
	if !strings.Contains(stdout, "⚕ Updating Gormes Agent...") {
		t.Errorf("non-check update must keep '⚕ Updating Gormes Agent...' banner; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "✓ Update complete!") {
		t.Errorf("non-check update must keep '✓ Update complete!' footer; got:\n%s", stdout)
	}
}
