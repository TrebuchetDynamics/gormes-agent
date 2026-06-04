package security

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	restoreapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/restore"
)

type RestoreBuildProvenance = restoreapp.BuildProvenance
type RestoreOptions = restoreapp.Options
type RestoreSeams = restoreapp.Seams

func NewRestoreCommandWithSeams(seams RestoreSeams, options RestoreOptions) *cobra.Command {
	if seams.BackupsDir == nil {
		seams.BackupsDir = restoreapp.DefaultBackupsDir
	}
	if seams.HomeDir == nil {
		seams.HomeDir = restoreapp.DefaultHomeDir
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
				return emitRestoreJSONInputError(cmd, options, "missing_argument", msg)
			}
			return restoreapp.Run(cmd.Context(), cmd.OutOrStdout(), seams, restoreapp.Request{List: list, Latest: latest, Path: path, Yes: yes, JSON: asJSON}, options)
		},
	}
	cmd.Flags().BoolVar(&list, "list", false, "list available pre-update backup zips, newest first")
	cmd.Flags().BoolVar(&latest, "latest", false, "restore the newest pre-update backup (resolved by mtime)")
	cmd.Flags().StringVar(&path, "path", "", "path to a pre-update-*.zip to restore (overwrites files in GORMES_HOME)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "confirm the destructive restore (without --yes the command runs as a dry-run preview)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `--list` returns `{build, backups: [...]}`; dry-run returns `{build, action: 'preview', path, dest, dry_run, would_overwrite, would_create}`; `--yes` returns `{build, action: 'restored', path, dest, file_count, overwrote, created}`")
	return cmd
}

func DefaultRestoreBackupsDir() string { return restoreapp.DefaultBackupsDir() }
func DefaultRestoreHomeDir() string    { return restoreapp.DefaultHomeDir() }

func emitRestoreJSONInputError(cmd *cobra.Command, options RestoreOptions, action, msg string) error {
	build := RestoreBuildProvenance{}
	if options.BuildProvenance != nil {
		build = options.BuildProvenance()
	}
	body, err := json.MarshalIndent(struct {
		Build  RestoreBuildProvenance `json:"build"`
		Action string                 `json:"action"`
		Error  string                 `json:"error"`
	}{Build: build, Action: action, Error: msg}, "", "  ")
	if err == nil {
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
	}
	return fmt.Errorf("%s", msg)
}
