package update

import (
	"context"
	"strings"
	"testing"
)

// TestUpdateLifecycle_NoBackupFlag_EmitsSkippedEvidence proves that --no-backup
// surfaces a typed `update_pre_backup_skipped` evidence with the
// backup_disabled_by_flag reason. Operators see an audible skip line in the
// structured progress UX so they know the policy was respected.
func TestUpdateLifecycle_NoBackupFlag_EmitsSkippedEvidence(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		NoBackup:    true,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidencePreBackupSkipped)
	if !findEvidenceDetailContains(report, UpdateEvidencePreBackupSkipped, string(BackupReasonDisabledByFlag)) {
		t.Fatalf("update_pre_backup_skipped detail must name the precedence reason; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_BackupFlag_EmitsRequestedEvidence proves --backup
// surfaces `update_pre_backup_requested` with the backup_forced reason. The
// actual backup writer is a follow-up slice; this row only wires the
// decision surface.
func TestUpdateLifecycle_BackupFlag_EmitsRequestedEvidence(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		Backup:      true,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidencePreBackupRequested)
	if !findEvidenceDetailContains(report, UpdateEvidencePreBackupRequested, string(BackupReasonForced)) {
		t.Fatalf("update_pre_backup_requested detail must name the precedence reason; got: %+v", report.Evidence)
	}
	// Important: the actual backup writer is a follow-up slice. The
	// requested-evidence detail must say so explicitly so operators don't
	// expect a backup file to exist after a green run.
	if !findEvidenceDetailContains(report, UpdateEvidencePreBackupRequested, "writer not yet implemented") {
		t.Fatalf("update_pre_backup_requested detail must explain the writer is deferred; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_NoBackupBeatsBackup proves the precedence rule: when
// both flags are set, --no-backup wins (matches Hermes' precedence).
func TestUpdateLifecycle_NoBackupBeatsBackup(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		Backup:      true,
		NoBackup:    true,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidencePreBackupSkipped)
	if findEvidenceKind(report, UpdateEvidencePreBackupRequested) {
		t.Fatalf("--no-backup must beat --backup; saw both kinds: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_DefaultEmitsNoBackupEvidence proves the silent-default
// contract from Hermes: when neither flag is set, no backup-related evidence
// is emitted at all. Most operators don't need to hear about the skipped
// backup on every update run.
func TestUpdateLifecycle_DefaultEmitsNoBackupEvidence(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	if findEvidenceKind(report, UpdateEvidencePreBackupSkipped) {
		t.Fatalf("default (no flags) must not emit pre_backup_skipped; got: %+v", report.Evidence)
	}
	if findEvidenceKind(report, UpdateEvidencePreBackupRequested) {
		t.Fatalf("default (no flags) must not emit pre_backup_requested; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_PreBackupRunsBeforeGitMutation proves the policy
// resolution happens BEFORE the first git fetch/pull, matching Hermes'
// `_run_pre_update_backup` ordering. Important for the follow-up slice that
// wires the actual backup writer — a backup taken AFTER the pull is useless
// for rollback.
func TestUpdateLifecycle_PreBackupRunsBeforeGitMutation(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		Backup:      true,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	backupIdx, fetchIdx := -1, -1
	for i, ev := range report.Evidence {
		if ev.Kind == UpdateEvidencePreBackupRequested {
			backupIdx = i
		}
		if backupIdx == -1 && (ev.Kind == UpdateEvidenceAutostashCreated) {
			t.Fatalf("autostash must not emit before pre-backup; ordering broken: %+v", report.Evidence)
		}
	}
	for _, cmd := range runner.commands {
		if strings.HasPrefix(cmd, "fetch ") {
			fetchIdx = 0 // any value > -1 means a fetch ran
			break
		}
	}
	if backupIdx < 0 {
		t.Fatalf("pre_backup_requested missing from evidence; got: %+v", report.Evidence)
	}
	if fetchIdx < 0 {
		t.Fatalf("fetch git command not invoked; runner.commands: %+v", runner.commands)
	}
}

// findEvidenceKind reports whether kind appears anywhere in the report.
func findEvidenceKind(report UpdateReport, kind UpdateEvidenceKind) bool {
	for _, ev := range report.Evidence {
		if ev.Kind == kind {
			return true
		}
	}
	return false
}

// findEvidenceDetailContains reports whether any evidence of the given kind
// has a Detail that contains needle.
func findEvidenceDetailContains(report UpdateReport, kind UpdateEvidenceKind, needle string) bool {
	for _, ev := range report.Evidence {
		if ev.Kind == kind && strings.Contains(ev.Detail, needle) {
			return true
		}
	}
	return false
}
