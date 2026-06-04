package gormescli

import (
	"context"
	"io"

	apprestore "github.com/TrebuchetDynamics/gormes-agent/internal/app/restore"
)

type RestoreBuildProvenance = apprestore.BuildProvenance
type RestoreCommandSeams = apprestore.Seams
type RestoreOptions = apprestore.Options
type RestoreRequest = apprestore.Request

func RunRestore(ctx context.Context, out io.Writer, seams RestoreCommandSeams, request RestoreRequest, options RestoreOptions) error {
	return apprestore.Run(ctx, out, seams, request, options)
}

func DefaultRestoreBackupsDir() string {
	return apprestore.DefaultBackupsDir()
}

func DefaultRestoreHomeDir() string {
	return apprestore.DefaultHomeDir()
}
