package backup

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

// PreUpdateBackupRequest captures the update lifecycle's backup archive
// contract: write a UTC-stamped pre-update zip under the operator home and
// prune older operator-owned backup archives after the write succeeds.
type PreUpdateBackupRequest struct {
	Home string
	Now  time.Time
	Keep int
}

// WritePreUpdateBackup writes one pre-update backup archive under
// <Home>/lifecycle/backups and best-effort prunes older pre-update archives in
// that directory. Prune failures are intentionally non-fatal because the newly
// written archive is already usable rollback evidence.
func WritePreUpdateBackup(ctx context.Context, req PreUpdateBackupRequest) (BackupResult, error) {
	if !textvalue.IsNonBlank(req.Home) {
		return BackupResult{}, fmt.Errorf("backup: home is empty")
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	dest := filepath.Join(req.Home, "lifecycle", "backups", "pre-update-"+now.Format("20060102T150405Z")+".zip")
	res, err := WriteBackupZip(ctx, req.Home, dest)
	if err != nil {
		return res, err
	}
	if count, freed, _ := PruneBackups(filepath.Dir(dest), req.Keep); count > 0 {
		res.PrunedCount = count
		res.PrunedBytes = freed
	}
	return res, nil
}
