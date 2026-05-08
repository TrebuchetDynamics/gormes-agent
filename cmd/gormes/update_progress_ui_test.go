package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

// runUpdateCommandWithReport synthesizes a cli.UpdateReport and runs the
// update command through the cobra plumbing so the test exercises the same
// stdout pipeline operators see, including printUpdateReport.
func runUpdateCommandWithReport(t *testing.T, report cli.UpdateReport) (stdout, stderr string) {
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
	cmd.SetArgs(nil)
	_ = cmd.Execute()
	return outBuf.String(), errBuf.String()
}

// TestUpdateCommand_StructuredHeaderEmitsBanner proves the human transcript
// now leads with the Hermes-style ⚕ banner. Closes the audit-row gap that
// the operator-visible output was a Spartan `update branch: …` followed by
// raw evidence_kind\tdetail rows.
func TestUpdateCommand_StructuredHeaderEmitsBanner(t *testing.T) {
	stdout, _ := runUpdateCommandWithReport(t, cli.UpdateReport{
		Branch: "main",
		Evidence: []cli.UpdateEvidence{
			{Kind: cli.UpdateEvidenceCheck, Detail: "checked update readiness for main"},
		},
	})
	if !strings.Contains(stdout, "⚕ Updating Gormes Agent...") {
		t.Fatalf("update output should lead with the ⚕ banner; got:\n%s", stdout)
	}
	// Banner must come BEFORE the existing per-evidence rows so the visual
	// hierarchy reads top-down.
	bannerIdx := strings.Index(stdout, "⚕ Updating Gormes Agent...")
	branchIdx := strings.Index(stdout, "update branch:")
	if bannerIdx >= branchIdx {
		t.Fatalf("⚕ banner must precede the `update branch:` line; got:\n%s", stdout)
	}
}

// TestUpdateCommand_SuccessAddsCheckmarkSummary proves the report ends with
// a single `✓ Update complete!` summary line, matching Hermes' update UX.
func TestUpdateCommand_SuccessAddsCheckmarkSummary(t *testing.T) {
	stdout, _ := runUpdateCommandWithReport(t, cli.UpdateReport{
		Branch: "main",
		Evidence: []cli.UpdateEvidence{
			{Kind: cli.UpdateEvidenceAutostashCreated, Detail: "stashed local changes"},
			{Kind: cli.UpdateEvidenceAutostashRestored, Detail: "restored stash refs/stash@{0}"},
		},
	})
	if !strings.Contains(stdout, "✓ Update complete!") {
		t.Fatalf("successful update should end with `✓ Update complete!`; got:\n%s", stdout)
	}
	// The summary must come AFTER the per-evidence block.
	summaryIdx := strings.Index(stdout, "✓ Update complete!")
	lastEvidenceIdx := strings.LastIndex(stdout, "update_autostash_restored")
	if summaryIdx < lastEvidenceIdx {
		t.Fatalf("`✓ Update complete!` must come after the last evidence row; got:\n%s", stdout)
	}
}

// TestUpdateCommand_FailedAddsCrossSummary proves a failed update ends with
// `✗ Update failed` and routes operator recovery guidance to stderr (where
// it always went; this row only adds the summary glyph in stdout).
func TestUpdateCommand_FailedAddsCrossSummary(t *testing.T) {
	stdout, stderr := runUpdateCommandWithReport(t, cli.UpdateReport{
		Branch: "main",
		Failed: true,
		Evidence: []cli.UpdateEvidence{
			{Kind: cli.UpdateEvidenceNetworkError, Detail: "dial tcp: connection refused"},
		},
		OperatorRecovery: "Check your network connection and rerun gormes update.",
	})
	if !strings.Contains(stdout, "✗ Update failed") {
		t.Fatalf("failed update should end stdout with `✗ Update failed`; got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "Check your network connection") {
		t.Fatalf("operator recovery must still route to stderr; got stderr:\n%s", stderr)
	}
}

// TestUpdateCommand_PreservesEvidenceKindsForMachineReadability proves
// every UpdateEvidenceKind string still appears verbatim in stdout so
// downstream parsers and existing tests that grep for `update_*` kinds
// continue to work. This is the backward-compatibility contract.
func TestUpdateCommand_PreservesEvidenceKindsForMachineReadability(t *testing.T) {
	wantKinds := []cli.UpdateEvidenceKind{
		cli.UpdateEvidenceAutostashCreated,
		cli.UpdateEvidenceAutostashRestored,
		cli.UpdateEvidenceGatewayRestarted,
	}
	evidence := make([]cli.UpdateEvidence, len(wantKinds))
	for i, k := range wantKinds {
		evidence[i] = cli.UpdateEvidence{Kind: k, Detail: "fixture"}
	}
	stdout, _ := runUpdateCommandWithReport(t, cli.UpdateReport{
		Branch:   "main",
		Evidence: evidence,
	})
	for _, k := range wantKinds {
		if !strings.Contains(stdout, string(k)) {
			t.Fatalf("evidence kind %q must still appear verbatim in stdout for parser compatibility; got:\n%s", k, stdout)
		}
	}
}

// TestUpdateCommand_GlyphsByEvidenceClass proves outcome glyphs map
// correctly: success kinds get ✓, error/failed kinds get ✗, unavailable/
// timeout kinds get ⚠. This is the operator-visible at-a-glance scan.
func TestUpdateCommand_GlyphsByEvidenceClass(t *testing.T) {
	cases := []struct {
		kind       cli.UpdateEvidenceKind
		wantGlyph  string
		wantAround string
	}{
		{cli.UpdateEvidenceAutostashCreated, "✓", "update_autostash_created"},
		{cli.UpdateEvidenceGatewayRestarted, "✓", "update_gateway_restarted"},
		{cli.UpdateEvidenceNetworkError, "✗", "update_network_error"},
		{cli.UpdateEvidenceBranchSwitchFailed, "✗", "update_branch_switch_failed"},
		{cli.UpdateEvidenceGatewayRestartUnavailable, "⚠", "update_gateway_restart_unavailable"},
		{cli.UpdateEvidenceGatewayRestartTimeout, "⚠", "update_gateway_restart_timeout"},
		{cli.UpdateEvidenceCheck, "ℹ", "update_check"},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			stdout, _ := runUpdateCommandWithReport(t, cli.UpdateReport{
				Branch:   "main",
				Evidence: []cli.UpdateEvidence{{Kind: tc.kind, Detail: "fixture"}},
			})
			needle := tc.wantGlyph + " " + tc.wantAround
			if !strings.Contains(stdout, needle) {
				t.Fatalf("expected %q in stdout; got:\n%s", needle, stdout)
			}
		})
	}
}

// TestUpdateCommand_NoColorStripsAnsi proves NO_COLOR=1 strips every
// styling escape from stdout/stderr. Captured-transcript safety.
func TestUpdateCommand_NoColorStripsAnsi(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	stdout, stderr := runUpdateCommandWithReport(t, cli.UpdateReport{
		Branch: "main",
		Evidence: []cli.UpdateEvidence{
			{Kind: cli.UpdateEvidenceGatewayRestarted, Detail: "pid 12345"},
		},
	})
	if strings.Contains(stdout, "\x1b[") {
		t.Fatalf("NO_COLOR=1 must strip ANSI from stdout; got %q", stdout)
	}
	if strings.Contains(stderr, "\x1b[") {
		t.Fatalf("NO_COLOR=1 must strip ANSI from stderr; got %q", stderr)
	}
}

// TestUpdateCommand_BackupFlagFlowsToLifecycle proves the cobra-side
// `--backup` and `--no-backup` flags reach the UpdateLifecycleOptions
// passed to RunLifecycle, so the policy resolution in update_lifecycle.go
// receives the operator's choice.
func TestUpdateCommand_BackupFlagFlowsToLifecycle(t *testing.T) {
	var got cli.UpdateLifecycleOptions
	cmd := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(_ context.Context, opts cli.UpdateLifecycleOptions) cli.UpdateReport {
			got = opts
			return cli.UpdateReport{Branch: "main"}
		},
	})
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"--backup"})
	_ = cmd.Execute()
	if !got.Backup || got.NoBackup {
		t.Fatalf("--backup should set Backup=true NoBackup=false; got Backup=%v NoBackup=%v", got.Backup, got.NoBackup)
	}

	got = cli.UpdateLifecycleOptions{}
	cmd2 := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(_ context.Context, opts cli.UpdateLifecycleOptions) cli.UpdateReport {
			got = opts
			return cli.UpdateReport{Branch: "main"}
		},
	})
	cmd2.SetOut(&outBuf)
	cmd2.SetErr(&errBuf)
	cmd2.SetArgs([]string{"--no-backup"})
	_ = cmd2.Execute()
	if got.Backup || !got.NoBackup {
		t.Fatalf("--no-backup should set Backup=false NoBackup=true; got Backup=%v NoBackup=%v", got.Backup, got.NoBackup)
	}
}

// TestUpdateCommand_SkillSyncSeamWiredByDefault proves the default
// updateCommandSeams resolution wires SkillSyncFor so the lifecycle
// receives a non-nil runner when the checkout has a skills/ directory and
// a profile root is configured.
func TestUpdateCommand_SkillSyncSeamWiredByDefault(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "skills"), 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	t.Setenv("GORMES_HOME", tmp)
	got := defaultSkillSyncFor(tmp)
	if got == nil {
		t.Fatalf("defaultSkillSyncFor must return a non-nil runner when checkout/skills exists and GORMES_HOME is set")
	}
}

// TestUpdateCommand_SkillSyncSeamNilWhenSkillsAbsent proves the default
// adapter returns nil (silent default) when the checkout has no skills/
// directory — most non-managed checkouts. The lifecycle then emits no
// skill_sync_* evidence, matching the silent-default contract.
func TestUpdateCommand_SkillSyncSeamNilWhenSkillsAbsent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GORMES_HOME", t.TempDir())
	got := defaultSkillSyncFor(tmp)
	if got != nil {
		t.Fatalf("defaultSkillSyncFor must return nil when checkout/skills does not exist; got %T", got)
	}
}

// TestUpdateCommand_SkillSyncCompletedGlyph proves the structured UX maps
// `update_skill_sync_completed` to ✓ (success) and the failed kind to ✗.
func TestUpdateCommand_SkillSyncGlyphs(t *testing.T) {
	cases := []struct {
		kind      cli.UpdateEvidenceKind
		wantGlyph string
	}{
		{cli.UpdateEvidenceSkillSyncCompleted, "✓"},
		{cli.UpdateEvidenceSkillSyncFailed, "✗"},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			stdout, _ := runUpdateCommandWithReport(t, cli.UpdateReport{
				Branch:   "main",
				Evidence: []cli.UpdateEvidence{{Kind: tc.kind, Detail: "default: +5 new, 12 unchanged"}},
			})
			needle := tc.wantGlyph + " " + string(tc.kind)
			if !strings.Contains(stdout, needle) {
				t.Fatalf("expected %q in stdout; got:\n%s", needle, stdout)
			}
		})
	}
}

// TestUpdateCommand_WebBuildFactoryNilWhenNoPackageJson proves the
// silent-default contract: when the checkout has no web/package.json,
// the factory returns nil and the lifecycle emits no web_build_*
// evidence at all.
func TestUpdateCommand_WebBuildFactoryNilWhenNoPackageJson(t *testing.T) {
	tmp := t.TempDir()
	if got := defaultWebBuildFor(tmp, false); got != nil {
		t.Fatalf("defaultWebBuildFor must return nil when web/package.json is absent; got non-nil")
	}
}

// TestUpdateCommand_WebBuildFactoryWiredWhenPackageJsonPresent proves the
// factory returns a non-nil runner when web/package.json exists.
func TestUpdateCommand_WebBuildFactoryWiredWhenPackageJsonPresent(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "web"), 0o755); err != nil {
		t.Fatalf("mkdir web: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "web", "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if got := defaultWebBuildFor(tmp, false); got == nil {
		t.Fatalf("defaultWebBuildFor must return non-nil when web/package.json exists")
	}
}

// TestUpdateCommand_WebBuildSkipFlagShortCircuitsRunner proves --skip-web
// flows through the factory closure: even when web/package.json exists,
// the runner returns Skipped without running npm.
func TestUpdateCommand_WebBuildSkipFlagShortCircuitsRunner(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "web"), 0o755); err != nil {
		t.Fatalf("mkdir web: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "web", "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	runner := defaultWebBuildFor(tmp, true)
	if runner == nil {
		t.Fatalf("defaultWebBuildFor must return non-nil even with skipWeb=true; the runner reports Skipped")
	}
	res, err := runner(context.Background())
	if err != nil {
		t.Fatalf("--skip-web runner must not return an error; got %v", err)
	}
	if !res.Skipped {
		t.Fatalf("--skip-web runner must return Skipped=true; got %+v", res)
	}
	if !strings.Contains(res.Reason, "--skip-web") {
		t.Fatalf("--skip-web Reason must name the flag; got %q", res.Reason)
	}
}

// TestUpdateCommand_WebBuildGlyphs proves the structured UX maps each
// web_build_* evidence kind to the right glyph: completed → ✓,
// skipped → ℹ, unavailable → ⚠, failed → ✗.
func TestUpdateCommand_WebBuildGlyphs(t *testing.T) {
	cases := []struct {
		kind      cli.UpdateEvidenceKind
		wantGlyph string
	}{
		{cli.UpdateEvidenceWebBuildCompleted, "✓"},
		{cli.UpdateEvidenceWebBuildSkipped, "ℹ"},
		{cli.UpdateEvidenceWebBuildUnavailable, "⚠"},
		{cli.UpdateEvidenceWebBuildFailed, "✗"},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			stdout, _ := runUpdateCommandWithReport(t, cli.UpdateReport{
				Branch:   "main",
				Evidence: []cli.UpdateEvidence{{Kind: tc.kind, Detail: "fixture"}},
			})
			needle := tc.wantGlyph + " " + string(tc.kind)
			if !strings.Contains(stdout, needle) {
				t.Fatalf("expected %q in stdout; got:\n%s", needle, stdout)
			}
		})
	}
}

// TestUpdateCommand_ConfigMigrateGlyphs proves the structured UX maps
// each config_migrate_* evidence kind to the right glyph: completed → ✓,
// needed → ⚠ (operator action required), failed → ✗.
func TestUpdateCommand_ConfigMigrateGlyphs(t *testing.T) {
	cases := []struct {
		kind      cli.UpdateEvidenceKind
		wantGlyph string
	}{
		{cli.UpdateEvidenceConfigMigrateCompleted, "✓"},
		{cli.UpdateEvidenceConfigMigrateNeeded, "⚠"},
		{cli.UpdateEvidenceConfigMigrateFailed, "✗"},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			stdout, _ := runUpdateCommandWithReport(t, cli.UpdateReport{
				Branch:   "main",
				Evidence: []cli.UpdateEvidence{{Kind: tc.kind, Detail: "config schema v11 → v19 available"}},
			})
			needle := tc.wantGlyph + " " + string(tc.kind)
			if !strings.Contains(stdout, needle) {
				t.Fatalf("expected %q in stdout; got:\n%s", needle, stdout)
			}
		})
	}
}

// TestUpdateCommand_PreBackupGlyphs proves the structured progress UX maps
// the new pre-backup evidence kinds to clear glyphs: skipped → ℹ (dim),
// requested → ◆ (bright cyan, matching Hermes' "◆ Creating pre-update
// backup..." marker).
func TestUpdateCommand_PreBackupGlyphs(t *testing.T) {
	cases := []struct {
		kind      cli.UpdateEvidenceKind
		wantGlyph string
	}{
		{cli.UpdateEvidencePreBackupSkipped, "ℹ"},
		{cli.UpdateEvidencePreBackupRequested, "◆"},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			stdout, _ := runUpdateCommandWithReport(t, cli.UpdateReport{
				Branch:   "main",
				Evidence: []cli.UpdateEvidence{{Kind: tc.kind, Detail: "fixture"}},
			})
			needle := tc.wantGlyph + " " + string(tc.kind)
			if !strings.Contains(stdout, needle) {
				t.Fatalf("expected %q in stdout; got:\n%s", needle, stdout)
			}
		})
	}
}

// silence the "imported and not used" warning if cobra is later removed
// from this test's symbol references; keep an explicit reference.
var _ = (*cobra.Command)(nil)
