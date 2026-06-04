package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type restoreCommandSeams = gormescli.RestoreCommandSeams

func newRestoreCommand() *cobra.Command {
	return newRestoreCommandWithSeams(restoreCommandSeams{})
}

func newRestoreCommandWithSeams(seams restoreCommandSeams) *cobra.Command {
	options := restoreCommandOptions()
	if seams.BackupsDir == nil {
		seams.BackupsDir = gormescli.DefaultRestoreBackupsDir
	}
	if seams.HomeDir == nil {
		seams.HomeDir = gormescli.DefaultRestoreHomeDir
	}
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
				return emitJSONInputError(cmd, "missing_argument", msg)
			}
			return gormescli.RunRestore(cmd.Context(), cmd.OutOrStdout(), seams, gormescli.RestoreRequest{
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

func restoreCommandOptions() gormescli.RestoreOptions {
	return gormescli.RestoreOptions{BuildProvenance: restoreBuildProvenance}
}

func restoreBuildProvenance() gormescli.RestoreBuildProvenance {
	build := newBuildProvenance()
	return gormescli.RestoreBuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}
