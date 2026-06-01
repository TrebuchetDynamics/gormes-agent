package backup

import (
	"context"

	restorepkg "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/backup/restore"
)

func ValidateRestoreZip(zipPath string) error { return restorepkg.ValidateRestoreZip(zipPath) }
func RestoreFromZip(ctx context.Context, zipPath, destDir string) error {
	return restorepkg.RestoreFromZip(ctx, zipPath, destDir)
}

type RestoreZipImpact = restorepkg.RestoreZipImpact

func SummarizeRestoreZipImpact(zipPath, destDir string) (RestoreZipImpact, error) {
	return restorepkg.SummarizeRestoreZipImpact(zipPath, destDir)
}
