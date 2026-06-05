package gormescli

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	apprestore "github.com/TrebuchetDynamics/gormes-agent/internal/app/restore"
)

type RestoreBuildProvenance = apprestore.BuildProvenance
type RestoreCommandSeams = apprestore.Seams
type RestoreOptions = apprestore.Options
type RestoreRequest = apprestore.Request

type RestoreCommandOptions struct {
	BuildProvenance func() BuildProvenance
	Seams           RestoreCommandSeams
}

func NewRestoreCommand(opts RestoreCommandOptions) *cobra.Command {
	build := opts.BuildProvenance
	if build == nil {
		build = func() BuildProvenance { return BuildProvenance{} }
	}
	seams := opts.Seams
	if seams.BackupsDir == nil {
		seams.BackupsDir = DefaultRestoreBackupsDir
	}
	if seams.HomeDir == nil {
		seams.HomeDir = DefaultRestoreHomeDir
	}
	options := RestoreOptions{BuildProvenance: func() RestoreBuildProvenance {
		provenance := build()
		return RestoreBuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
	}}
	var list bool
	var latest bool
	var path string
	var yes bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Discover and restore from a pre-update backup zip",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !list && !latest && path == "" && asJSON {
				const msg = "restore: pass --list to enumerate backups, --latest for the newest, or --path <zip> for a specific one"
				return EmitJSONInputError(cmd, "missing_argument", msg, build())
			}
			return RunRestore(cmd.Context(), cmd.OutOrStdout(), seams, RestoreRequest{
				List:   list,
				Latest: latest,
				Path:   path,
				Yes:    yes,
				JSON:   asJSON,
			}, options)
		},
	}
	cmd.Flags().BoolVar(&list, "list", false, "list available pre-update backup zips, newest first")
	cmd.Flags().BoolVar(&latest, "latest", false, "restore the newest pre-update backup (resolved by mtime)")
	cmd.Flags().StringVar(&path, "path", "", "path to a pre-update-*.zip to restore (overwrites files in GORMES_HOME)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "confirm the destructive restore (without --yes the command runs as a dry-run preview)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `--list` returns `{build, backups: [...]}`; dry-run returns `{build, action: 'preview', path, dest, dry_run, would_overwrite, would_create}`; `--yes` returns `{build, action: 'restored', path, dest, file_count, overwrote, created}`")
	return cmd
}

func RunRestore(ctx context.Context, out io.Writer, seams RestoreCommandSeams, request RestoreRequest, options RestoreOptions) error {
	return apprestore.Run(ctx, out, seams, request, options)
}

func DefaultRestoreBackupsDir() string {
	return apprestore.DefaultBackupsDir()
}

func DefaultRestoreHomeDir() string {
	return apprestore.DefaultHomeDir()
}
