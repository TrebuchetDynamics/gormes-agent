package restore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type Options struct {
	BuildProvenance func() BuildProvenance
	JSONInputError  func(*cobra.Command, string, string) error
}

func (o Options) buildProvenance() BuildProvenance {
	if o.BuildProvenance != nil {
		return o.BuildProvenance()
	}
	return BuildProvenance{}
}

func (o Options) emitJSONInputError(cmd *cobra.Command, action, msg string) error {
	if o.JSONInputError != nil {
		return o.JSONInputError(cmd, action, msg)
	}
	return fmt.Errorf("%s", msg)
}

type Seams struct {
	BackupsDir func() string
	HomeDir    func() string
}

func NewCommand(options Options) *cobra.Command {
	return NewCommandWithSeams(Seams{}, options)
}

func NewCommandWithSeams(seams Seams, options Options) *cobra.Command {
	if seams.BackupsDir == nil {
		seams.BackupsDir = DefaultBackupsDir
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
				return RunList(cmd, seams, asJSON, options)
			}
			resolvedPath := path
			if latest {
				resolved, err := ResolveLatestBackup(seams.BackupsDir())
				if err != nil {
					return err
				}
				resolvedPath = resolved
			}
			if resolvedPath == "" {
				const msg = "restore: pass --list to enumerate backups, --latest for the newest, or --path <zip> for a specific one"
				if asJSON {
					return options.emitJSONInputError(cmd, "missing_argument", msg)
				}
				return fmt.Errorf("%s", msg)
			}
			home := seams.HomeDir()
			if home == "" {
				return fmt.Errorf("restore: GORMES_HOME is unset; cannot resolve restore root")
			}
			if !yes {
				if validateErr := cli.ValidateRestoreZip(resolvedPath); validateErr != nil {
					return validateErr
				}
				if asJSON {
					impact, _ := cli.SummarizeRestoreZipImpact(resolvedPath, home)
					body, marshalErr := json.MarshalIndent(PreviewJSON{
						Build:          options.buildProvenance(),
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
					fmt.Fprintf(out, " (%s, %s)", FormatSize(info.Size()), FormatAge(time.Since(info.ModTime())))
				}
				fmt.Fprintf(out, " into %s\n", home)
				if impact, impactErr := cli.SummarizeRestoreZipImpact(resolvedPath, home); impactErr == nil {
					fmt.Fprintf(out, "would overwrite %d existing file(s), create %d new\n", impact.Overwrite, impact.Create)
				}
				fmt.Fprintln(out, "Re-run with --yes to actually restore (existing files will be overwritten).")
				return nil
			}
			impact, _ := cli.SummarizeRestoreZipImpact(resolvedPath, home)
			if err := cli.RestoreFromZip(cmd.Context(), resolvedPath, home); err != nil {
				return err
			}
			if asJSON {
				body, err := json.MarshalIndent(OutcomeJSON{
					Build:     options.buildProvenance(),
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

type ListReportJSON struct {
	Build   BuildProvenance `json:"build"`
	Backups []BackupJSON    `json:"backups"`
}

type OutcomeJSON struct {
	Build     BuildProvenance `json:"build"`
	Action    string          `json:"action"`
	Path      string          `json:"path"`
	Dest      string          `json:"dest"`
	FileCount int             `json:"file_count"`
	Overwrote int             `json:"overwrote"`
	Created   int             `json:"created"`
}

type PreviewJSON struct {
	Build          BuildProvenance `json:"build"`
	Action         string          `json:"action"`
	Path           string          `json:"path"`
	Dest           string          `json:"dest"`
	DryRun         bool            `json:"dry_run"`
	WouldOverwrite int             `json:"would_overwrite"`
	WouldCreate    int             `json:"would_create"`
}

type BackupJSON struct {
	Path      string    `json:"path"`
	SizeBytes int64     `json:"size_bytes"`
	ModTime   time.Time `json:"mod_time"`
}

func ResolveLatestBackup(backupsDir string) (string, error) {
	entries, err := cli.ListBackups(backupsDir)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("restore: no backups found in %s — to create backups run `gormes update --backup` or set `[updates] pre_update_backup = true` in config.toml", backupsDir)
	}
	return entries[0].Path, nil
}

func RunList(cmd *cobra.Command, seams Seams, asJSON bool, options Options) error {
	out := cmd.OutOrStdout()
	dir := seams.BackupsDir()
	entries, err := cli.ListBackups(dir)
	if err != nil {
		return err
	}
	if asJSON {
		records := make([]BackupJSON, len(entries))
		for i, e := range entries {
			records[i] = BackupJSON{Path: e.Path, SizeBytes: e.SizeBytes, ModTime: e.ModTime}
		}
		body, marshalErr := json.MarshalIndent(ListReportJSON{Build: options.buildProvenance(), Backups: records}, "", "  ")
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
		fmt.Fprintf(out, "  %s  %s  %s\n", filepath.Base(e.Path), FormatSize(e.SizeBytes), FormatAge(now.Sub(e.ModTime)))
	}
	return nil
}

func DefaultBackupsDir() string {
	return filepath.Join(config.GormesHome(), "lifecycle", "backups")
}

func FormatSize(bytes int64) string {
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

func FormatAge(d time.Duration) string {
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
