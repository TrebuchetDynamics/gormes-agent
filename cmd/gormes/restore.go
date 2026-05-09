package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type restoreCommandSeams struct {
	// BackupsDir resolves the directory the writer wrote to. Default is
	// `<GormesHome>/backups`. Tests inject a temp dir.
	BackupsDir func() string
	// HomeDir resolves the destination root of the extract — the
	// directory the zip was originally taken from. Default is
	// `GormesHome()`. Tests inject a temp dir.
	HomeDir func() string
}

func newRestoreCommand() *cobra.Command {
	return newRestoreCommandWithSeams(restoreCommandSeams{})
}

func newRestoreCommandWithSeams(seams restoreCommandSeams) *cobra.Command {
	if seams.BackupsDir == nil {
		seams.BackupsDir = defaultBackupsDir
	}
	if seams.HomeDir == nil {
		seams.HomeDir = config.GormesHome
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
			out := cmd.OutOrStdout()
			if list {
				return runRestoreList(cmd, seams, asJSON)
			}
			resolvedPath := path
			if latest {
				resolved, err := resolveLatestBackup(seams.BackupsDir())
				if err != nil {
					return err
				}
				resolvedPath = resolved
			}
			if resolvedPath == "" {
				const msg = "restore: pass --list to enumerate backups, --latest for the newest, or --path <zip> for a specific one"
				if asJSON {
					return emitJSONInputError(cmd, "missing_argument", msg)
				}
				return fmt.Errorf("%s", msg)
			}
			home := seams.HomeDir()
			if home == "" {
				return fmt.Errorf("restore: GORMES_HOME is unset; cannot resolve restore root")
			}
			if !yes {
				// Validate the zip is openable + free of path-traversal
				// entries BEFORE printing "would extract". Otherwise a
				// corrupt or malicious zip would mislead the operator
				// into running --yes only to fail mid-extract.
				if validateErr := cli.ValidateRestoreZip(resolvedPath); validateErr != nil {
					return validateErr
				}
				if asJSON {
					impact, _ := cli.SummarizeRestoreZipImpact(resolvedPath, home)
					body, marshalErr := json.MarshalIndent(restorePreviewJSON{
						Build:          newBuildProvenance(),
						Action:         "preview",
						Path:           resolvedPath,
						Dest:           home,
						DryRun:         true,
						WouldOverwrite: impact.Overwrite,
						WouldCreate:    impact.Create,
					}, "", "  ")
					if marshalErr != nil {
						return marshalErr
					}
					fmt.Fprintln(out, string(body))
					return nil
				}
				fmt.Fprintf(out, "DRY RUN — would extract %s", resolvedPath)
				if info, statErr := os.Stat(resolvedPath); statErr == nil {
					fmt.Fprintf(out,
						" (%s, %s)",
						formatRestoreSize(info.Size()),
						formatRestoreAge(time.Since(info.ModTime())),
					)
				}
				fmt.Fprintf(out, " into %s\n", home)
				if impact, impactErr := cli.SummarizeRestoreZipImpact(resolvedPath, home); impactErr == nil {
					fmt.Fprintf(out,
						"would overwrite %d existing file(s), create %d new\n",
						impact.Overwrite, impact.Create,
					)
				}
				fmt.Fprintln(out, "Re-run with --yes to actually restore (existing files will be overwritten).")
				return nil
			}
			// Capture pre-extract impact so JSON consumers see how
			// many files overwrote vs. landed new. Computed BEFORE
			// extraction (after which every entry would look like an
			// overwrite, defeating the signal).
			impact, _ := cli.SummarizeRestoreZipImpact(resolvedPath, home)
			if err := cli.RestoreFromZip(cmd.Context(), resolvedPath, home); err != nil {
				return err
			}
			if asJSON {
				body, err := json.MarshalIndent(restoreOutcomeJSON{
					Build:     newBuildProvenance(),
					Action:    "restored",
					Path:      resolvedPath,
					Dest:      home,
					FileCount: impact.Overwrite + impact.Create,
					Overwrote: impact.Overwrite,
					Created:   impact.Create,
				}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(out, string(body))
				return nil
			}
			fmt.Fprintf(out, "restored %s into %s\n", filepath.Base(resolvedPath), home)
			return nil
		},
	}
	cmd.Flags().BoolVar(&list, "list", false, "list available pre-update backup zips, newest first")
	cmd.Flags().BoolVar(&latest, "latest", false, "restore the newest pre-update backup (resolved by mtime)")
	cmd.Flags().StringVar(&path, "path", "", "path to a pre-update-*.zip to restore (overwrites files in GORMES_HOME)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "confirm the destructive restore (without --yes the command runs as a dry-run preview)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `--list` returns `{build, backups: [...]}`; dry-run returns `{build, action: 'preview', path, dest, dry_run, would_overwrite, would_create}`; `--yes` returns `{build, action: 'restored', path, dest, file_count, overwrote, created}`")
	return cmd
}

// restoreListReportJSON is the wire shape for `restore --list --json`.
// Build provenance leads, then the backups array — same convention as
// update --json / doctor --json / status --json.
type restoreListReportJSON struct {
	Build   buildProvenanceJSON `json:"build"`
	Backups []backupListingJSON `json:"backups"`
}

// restoreOutcomeJSON is the wire shape for `restore --path X --yes --json`.
// Operator scripts driving Gormes restore in automation parse this to
// verify what landed: which zip was extracted, into which dest root,
// how many files moved, and how many were net-new vs. overwrites.
// Build provenance leads — same convention as the rest of the --json arc.
type restoreOutcomeJSON struct {
	Build     buildProvenanceJSON `json:"build"`
	Action    string              `json:"action"`
	Path      string              `json:"path"`
	Dest      string              `json:"dest"`
	FileCount int                 `json:"file_count"`
	Overwrote int                 `json:"overwrote"`
	Created   int                 `json:"created"`
}

// restorePreviewJSON is the wire shape for `restore --path X --json`
// without --yes — a dry-run preview. Operator scripts driving restore
// gate decisions (CI/CD approvals, kanban hooks, audit logs) parse
// this to learn the resolved zip, the dest root, and the blast radius
// (`would_overwrite` + `would_create`) BEFORE pulling the trigger
// with --yes. `dry_run: true` and `action: "preview"` distinguish it
// from the post-extract `restored` outcome shape.
type restorePreviewJSON struct {
	Build          buildProvenanceJSON `json:"build"`
	Action         string              `json:"action"`
	Path           string              `json:"path"`
	Dest           string              `json:"dest"`
	DryRun         bool                `json:"dry_run"`
	WouldOverwrite int                 `json:"would_overwrite"`
	WouldCreate    int                 `json:"would_create"`
}

type backupListingJSON struct {
	Path      string    `json:"path"`
	SizeBytes int64     `json:"size_bytes"`
	ModTime   time.Time `json:"mod_time"`
}

// resolveLatestBackup returns the path of the newest pre-update zip in
// backupsDir. Returns a typed error when the directory is missing or
// empty so operators see why `--latest` failed.
func resolveLatestBackup(backupsDir string) (string, error) {
	entries, err := cli.ListBackups(backupsDir)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("restore: no backups found in %s — to create backups run `gormes update --backup` or set `[updates] pre_update_backup = true` in config.toml", backupsDir)
	}
	return entries[0].Path, nil
}

func runRestoreList(cmd *cobra.Command, seams restoreCommandSeams, asJSON bool) error {
	out := cmd.OutOrStdout()
	dir := seams.BackupsDir()
	entries, err := cli.ListBackups(dir)
	if err != nil {
		return err
	}
	if asJSON {
		records := make([]backupListingJSON, len(entries))
		for i, e := range entries {
			records[i] = backupListingJSON{
				Path:      e.Path,
				SizeBytes: e.SizeBytes,
				ModTime:   e.ModTime,
			}
		}
		body, marshalErr := json.MarshalIndent(restoreListReportJSON{
			Build:   newBuildProvenance(),
			Backups: records,
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(out, string(body))
		return nil
	}
	if len(entries) == 0 {
		fmt.Fprintln(out, "no backups found in "+dir)
		return nil
	}
	fmt.Fprintf(out, "Backups in %s (newest first):\n", dir)
	now := time.Now()
	for _, e := range entries {
		fmt.Fprintf(out,
			"  %s  %s  %s\n",
			filepath.Base(e.Path),
			formatRestoreSize(e.SizeBytes),
			formatRestoreAge(now.Sub(e.ModTime)),
		)
	}
	return nil
}

func defaultBackupsDir() string {
	return filepath.Join(config.GormesHome(), "backups")
}

// formatRestoreSize renders a byte count with the smallest unit that
// keeps the number under 1024. Mirrors lifecycle's formatBackupSize so
// the operator-visible columns line up.
func formatRestoreSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * 1024
		gb = 1024 * 1024 * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// formatRestoreAge renders a duration as a coarse human-readable age
// suitable for a one-row column ("3m ago", "2h ago", "5d ago").
// Operators use this to spot the right rollback target at a glance.
func formatRestoreAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	default:
		return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
	}
}
