package backup

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/backup/archive"
)

type BackupResult = archive.BackupResult
type BackupListing = archive.BackupListing

func ListBackups(backupDir string) ([]BackupListing, error) { return archive.ListBackups(backupDir) }
func PruneBackups(backupDir string, keep int) (removedCount int, freedBytes int64, err error) {
	return archive.PruneBackups(backupDir, keep)
}
func WriteBackupZip(ctx context.Context, sourceDir, destPath string) (BackupResult, error) {
	return archive.WriteBackupZip(ctx, sourceDir, destPath)
}
