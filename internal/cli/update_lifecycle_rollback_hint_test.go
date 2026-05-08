package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestUpdateLifecycle_FailureAfterBackupSurfacesRollbackHint proves the
// operator-visible rollback workflow: when a pre-update backup wrote
// successfully but the update later fails, the lifecycle appends an
// `update_rollback_hint` evidence record naming
// `gormes restore --latest --yes`. Operators staring at a broken
// install should see the recovery command inline, not have to remember
// the restore CLI shape from documentation.
func TestUpdateLifecycle_FailureAfterBackupSurfacesRollbackHint(t *testing.T) {
	writer := &fakeBackupWriter{result: BackupResult{
		Path:       "/home/op/.gormes/backups/pre-update-x.zip",
		SizeBytes:  4_000,
		DurationMs: 100,
	}}
	// Git runner returns false for rev-parse so the lifecycle
	// short-circuits with not_managed_checkout AFTER the backup ran.
	git := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "false\n"},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir:  "/repo/gormes",
		Branch:       "main",
		Backup:       true,
		BackupWriter: writer.Write,
		Git:          git,
	})

	if !report.Failed {
		t.Fatalf("expected Failed=true after rev-parse=false; got %+v", report)
	}
	if !findEvidenceKind(report, UpdateEvidencePreBackupCompleted) {
		t.Fatalf("test prerequisite missing: pre_backup_completed must be in report; got %+v", report.Evidence)
	}
	hint := findFirstEvidenceDetail(report, UpdateEvidenceRollbackHint, "gormes restore")
	if hint == "" {
		t.Fatalf("update_rollback_hint must be emitted when backup succeeded but update failed; got %+v", report.Evidence)
	}
	if !strings.Contains(hint, "--latest") || !strings.Contains(hint, "--yes") {
		t.Fatalf("hint must name `gormes restore --latest --yes`; got %q", hint)
	}
}

// TestUpdateLifecycle_FailureWithoutBackupSkipsRollbackHint proves the
// hint is gated on a usable backup. When `--no-backup` was set (or no
// writer ran), there's no zip to restore from — emitting the hint
// would point operators at a path that doesn't exist.
func TestUpdateLifecycle_FailureWithoutBackupSkipsRollbackHint(t *testing.T) {
	git := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "false\n"},
	})
	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		NoBackup:    true,
		Git:         git,
	})

	if !report.Failed {
		t.Fatalf("expected Failed=true; got %+v", report)
	}
	if findEvidenceKind(report, UpdateEvidenceRollbackHint) {
		t.Fatalf("rollback hint must NOT appear without a completed backup; got %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_BackupFailedSkipsRollbackHint proves the gating
// is on `pre_backup_completed`, not on the mere presence of a backup
// writer. When the writer errored, no zip exists — so no hint.
func TestUpdateLifecycle_BackupFailedSkipsRollbackHint(t *testing.T) {
	writer := &fakeBackupWriter{err: errors.New("disk full")}
	git := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "false\n"},
	})
	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir:  "/repo/gormes",
		Branch:       "main",
		Backup:       true,
		BackupWriter: writer.Write,
		Git:          git,
	})

	if !report.Failed {
		t.Fatalf("expected Failed=true; got %+v", report)
	}
	if !findEvidenceKind(report, UpdateEvidencePreBackupFailed) {
		t.Fatalf("test prerequisite missing: pre_backup_failed must be in report; got %+v", report.Evidence)
	}
	if findEvidenceKind(report, UpdateEvidenceRollbackHint) {
		t.Fatalf("rollback hint must NOT appear when backup itself failed; got %+v", report.Evidence)
	}
}
