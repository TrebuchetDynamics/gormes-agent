package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeBackupWriter struct {
	result BackupResult
	err    error
	calls  int
}

func (f *fakeBackupWriter) Write(_ context.Context) (BackupResult, error) {
	f.calls++
	return f.result, f.err
}

// TestUpdateLifecycle_BackupRequestedNilWriter_KeepsRequestedDeferralEvidence
// proves backward-compat with the prior policy-only slice (56efc4042):
// when --backup is set but no writer is wired, the lifecycle still emits
// `update_pre_backup_requested` with the "writer not yet implemented"
// detail so existing tests and operator transcripts don't regress.
func TestUpdateLifecycle_BackupRequestedNilWriter_KeepsRequestedDeferralEvidence(t *testing.T) {
	report := runCleanLifecycle(t, UpdateLifecycleOptions{
		Backup: true,
		// BackupWriter intentionally nil
	})
	if findFirstEvidenceDetail(report, UpdateEvidencePreBackupRequested, "writer not yet implemented") == "" {
		t.Fatalf("nil BackupWriter must keep the requested-deferral detail; got: %+v", report.Evidence)
	}
	if findEvidenceKind(report, UpdateEvidencePreBackupCompleted) {
		t.Fatalf("nil BackupWriter must not emit completed evidence; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_BackupWriterSuccess_EmitsCompletedEvidence proves
// that when --backup is set AND the BackupWriter seam is wired, a
// successful run emits `update_pre_backup_completed` with the path,
// size, and duration in the detail. The deferral-detail variant from
// the prior slice is REPLACED — operators see the real backup outcome.
func TestUpdateLifecycle_BackupWriterSuccess_EmitsCompletedEvidence(t *testing.T) {
	writer := &fakeBackupWriter{result: BackupResult{
		Path:       "/home/op/.gormes/backups/pre-update-20260507T210748Z.zip",
		SizeBytes:  4_237_312,
		DurationMs: 1_842,
	}}
	report := runCleanLifecycle(t, UpdateLifecycleOptions{
		Backup:       true,
		BackupWriter: writer.Write,
	})
	if writer.calls != 1 {
		t.Fatalf("BackupWriter must be called exactly once when --backup is set; got %d", writer.calls)
	}
	if findFirstEvidenceDetail(report, UpdateEvidencePreBackupCompleted, "backups/pre-update-") == "" {
		t.Fatalf("update_pre_backup_completed must include the backup path; got: %+v", report.Evidence)
	}
	// Operators read this detail in the structured progress UX, so the
	// formatted size and duration must be human-readable.
	completedDetail := findFirstEvidenceDetail(report, UpdateEvidencePreBackupCompleted, "backups/pre-update-")
	if !strings.Contains(completedDetail, "MB") && !strings.Contains(completedDetail, "KB") {
		t.Fatalf("completed detail must include human-readable size; got %q", completedDetail)
	}
	if !strings.Contains(completedDetail, "1.84s") && !strings.Contains(completedDetail, "1842ms") {
		t.Fatalf("completed detail must include human-readable duration; got %q", completedDetail)
	}
	// The requested-deferral detail must NOT appear when a real writer
	// runs — otherwise operators would see two contradictory lines.
	for _, ev := range report.Evidence {
		if ev.Kind == UpdateEvidencePreBackupRequested && strings.Contains(ev.Detail, "writer not yet implemented") {
			t.Fatalf("when BackupWriter is wired, the deferral detail must NOT also be emitted; got: %+v", report.Evidence)
		}
	}
}

// TestUpdateLifecycle_BackupWriterFailure_EmitsFailedEvidenceContinues
// proves the best-effort contract: a writer error emits
// `update_pre_backup_failed` but the update completes successfully
// (matches Hermes' "never raises — a backup failure should not block the
// update itself").
func TestUpdateLifecycle_BackupWriterFailure_EmitsFailedEvidenceContinues(t *testing.T) {
	writer := &fakeBackupWriter{err: errors.New("zip: short write")}
	report := runCleanLifecycle(t, UpdateLifecycleOptions{
		Backup:       true,
		BackupWriter: writer.Write,
	})
	if report.Failed {
		t.Fatalf("backup writer error must NOT fail the update; got: %+v", report)
	}
	if findFirstEvidenceDetail(report, UpdateEvidencePreBackupFailed, "short write") == "" {
		t.Fatalf("update_pre_backup_failed must include the error message; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_BackupWriterNotInvokedWhenNoBackup proves writer is
// not called on the --no-backup path. The lifecycle must short-circuit
// at the policy resolution and emit only the skipped evidence.
func TestUpdateLifecycle_BackupWriterNotInvokedWhenNoBackup(t *testing.T) {
	writer := &fakeBackupWriter{result: BackupResult{Path: "should-not-be-used"}}
	report := runCleanLifecycle(t, UpdateLifecycleOptions{
		NoBackup:     true,
		BackupWriter: writer.Write,
	})
	if writer.calls != 0 {
		t.Fatalf("--no-backup must short-circuit the writer; got %d calls", writer.calls)
	}
	if !findEvidenceKind(report, UpdateEvidencePreBackupSkipped) {
		t.Fatalf("--no-backup must still emit skipped evidence; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_BackupCompletedDetailIncludesPruneInfoWhenPositive
// proves the structured progress UX surfaces the prune count + freed
// bytes in the same evidence record as the completed backup, but only
// when at least one older backup was actually removed. Common case
// (PrunedCount=0) keeps the transcript short.
func TestUpdateLifecycle_BackupCompletedDetailIncludesPruneInfoWhenPositive(t *testing.T) {
	writer := &fakeBackupWriter{result: BackupResult{
		Path:        "/home/op/.gormes/backups/pre-update-x.zip",
		SizeBytes:   2_500_000,
		DurationMs:  1_200,
		PrunedCount: 3,
		PrunedBytes: 12_000_000,
	}}
	report := runCleanLifecycle(t, UpdateLifecycleOptions{
		Backup:       true,
		BackupWriter: writer.Write,
	})
	detail := findFirstEvidenceDetail(report, UpdateEvidencePreBackupCompleted, "pre-update-x.zip")
	if !strings.Contains(detail, "pruned 3 older") {
		t.Fatalf("detail must include prune count when >0; got %q", detail)
	}
	if !strings.Contains(detail, "freed") {
		t.Fatalf("detail must include freed-bytes phrasing; got %q", detail)
	}
}

// TestUpdateLifecycle_BackupCompletedDetailOmitsPruneSuffixWhenZero
// proves the no-prune common case omits the prune-suffix entirely.
// Operators on their first --backup run shouldn't see "pruned 0 older"
// noise.
func TestUpdateLifecycle_BackupCompletedDetailOmitsPruneSuffixWhenZero(t *testing.T) {
	writer := &fakeBackupWriter{result: BackupResult{
		Path:       "/tmp/x.zip",
		SizeBytes:  1024,
		DurationMs: 50,
		// PrunedCount=0 (default)
	}}
	report := runCleanLifecycle(t, UpdateLifecycleOptions{
		Backup:       true,
		BackupWriter: writer.Write,
	})
	detail := findFirstEvidenceDetail(report, UpdateEvidencePreBackupCompleted, "/tmp/x.zip")
	if strings.Contains(detail, "pruned") {
		t.Fatalf("detail must omit prune phrasing when PrunedCount=0; got %q", detail)
	}
}

// TestUpdateLifecycle_BackupSizeFormatting proves the size formatter
// scales: bytes < 1024 → "B", < 1MB → "KB", < 1GB → "MB", >= 1GB → "GB".
// Tested via the lifecycle-emitted detail to keep formatter/lifecycle
// behavior locked together.
func TestUpdateLifecycle_BackupSizeFormatting(t *testing.T) {
	cases := []struct {
		size int64
		want string
	}{
		{512, "512B"},
		{4096, "4.0KB"},
		{1_500_000, "1.4MB"},
		{2_500_000_000, "2.3GB"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			writer := &fakeBackupWriter{result: BackupResult{
				Path:       "/tmp/x.zip",
				SizeBytes:  tc.size,
				DurationMs: 100,
			}}
			report := runCleanLifecycle(t, UpdateLifecycleOptions{
				Backup:       true,
				BackupWriter: writer.Write,
			})
			detail := findFirstEvidenceDetail(report, UpdateEvidencePreBackupCompleted, "/tmp/x.zip")
			if !strings.Contains(detail, tc.want) {
				t.Fatalf("expected size %q in detail; got %q", tc.want, detail)
			}
		})
	}
}
