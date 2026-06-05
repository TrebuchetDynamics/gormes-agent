package cli

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/backup"
)

// ValidateRestoreZip opens the zip at zipPath and walks its entries
// without writing anything to disk. Returns a typed error when the
// archive is unreadable or contains a path-traversal entry that
// RestoreFromZip would reject at extract time. Used by the dry-run
// preview so operators see corruption/traversal up-front instead of
// after committing to --yes.
func ValidateRestoreZip(zipPath string) error { return backup.ValidateRestoreZip(zipPath) }

// RestoreFromZip extracts every entry of the zip at zipPath into
// destDir, overwriting existing files. This is the rollback path
// consumed by `gormes restore --path`.
func RestoreFromZip(ctx context.Context, zipPath, destDir string) error {
	return backup.RestoreFromZip(ctx, zipPath, destDir)
}

// RestoreZipImpact summarizes how a zip restore will affect the
// destination tree without writing anything. The dry-run preview uses
// this so operators see the blast radius (overwrite vs. create counts)
// before committing to --yes.
type RestoreZipImpact = backup.RestoreZipImpact

// SummarizeRestoreZipImpact walks the zip at zipPath against the on-disk
// destDir and classifies each file entry as either an overwrite (the
// resolved on-disk target already exists) or a create (it does not).
func SummarizeRestoreZipImpact(zipPath, destDir string) (RestoreZipImpact, error) {
	return backup.SummarizeRestoreZipImpact(zipPath, destDir)
}
