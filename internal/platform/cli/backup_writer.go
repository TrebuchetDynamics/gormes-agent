package cli

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/backup"
)

// BackupListing describes one pre-update backup zip discovered under
// the operator's backups directory. Returned by ListBackups for the
// `gormes restore --list` operator surface.
type BackupListing = backup.BackupListing

type PreUpdateBackupRequest = backup.PreUpdateBackupRequest

// ListBackups enumerates `pre-update-*.zip` files under backupDir and
// returns them sorted newest-first by mtime. Files that don't match the
// pattern are ignored (operators may store unrelated files in the same
// directory). A missing backupDir is a quiet empty-list.
func ListBackups(backupDir string) ([]BackupListing, error) { return backup.ListBackups(backupDir) }

// PruneBackups removes older pre-update backup zips from backupDir,
// keeping the newest `keep` files by mtime. Returns the number of files
// removed and total bytes freed.
func PruneBackups(backupDir string, keep int) (removedCount int, freedBytes int64, err error) {
	return backup.PruneBackups(backupDir, keep)
}

func WritePreUpdateBackup(ctx context.Context, req PreUpdateBackupRequest) (BackupResult, error) {
	res, err := backup.WritePreUpdateBackup(ctx, req)
	if err != nil {
		return BackupResult{}, err
	}
	return BackupResult{
		Path:        res.Path,
		SizeBytes:   res.SizeBytes,
		DurationMs:  res.DurationMs,
		FileCount:   res.FileCount,
		PrunedCount: res.PrunedCount,
		PrunedBytes: res.PrunedBytes,
	}, nil
}

// WriteBackupZip walks sourceDir and writes a zip archive containing
// every file that does NOT match the IsExcludedFromBackup rules. The
// zip is written atomically: it streams to destPath.tmp first and
// renames on success.
func WriteBackupZip(ctx context.Context, sourceDir, destPath string) (BackupResult, error) {
	res, err := backup.WriteBackupZip(ctx, sourceDir, destPath)
	if err != nil {
		return BackupResult{}, err
	}
	return BackupResult{
		Path:        res.Path,
		SizeBytes:   res.SizeBytes,
		DurationMs:  res.DurationMs,
		FileCount:   res.FileCount,
		PrunedCount: res.PrunedCount,
		PrunedBytes: res.PrunedBytes,
	}, nil
}
