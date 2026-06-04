package restore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type Options struct {
	BuildProvenance func() BuildProvenance
}

func (o Options) buildProvenance() BuildProvenance {
	if o.BuildProvenance != nil {
		return o.BuildProvenance()
	}
	return BuildProvenance{}
}

type Seams struct {
	BackupsDir func() string
	HomeDir    func() string
}

type Request struct {
	List   bool
	Latest bool
	Path   string
	Yes    bool
	JSON   bool
}

func Run(ctx context.Context, out io.Writer, seams Seams, request Request, options Options) error {
	seams = normalizeSeams(seams)
	if request.List {
		return RunList(out, seams.BackupsDir(), request.JSON, options)
	}
	resolvedPath := request.Path
	if request.Latest {
		resolved, err := ResolveLatestBackup(seams.BackupsDir())
		if err != nil {
			return err
		}
		resolvedPath = resolved
	}
	if resolvedPath == "" {
		return fmt.Errorf("restore: pass --list to enumerate backups, --latest for the newest, or --path <zip> for a specific one")
	}
	home := seams.HomeDir()
	if home == "" {
		return fmt.Errorf("restore: GORMES_HOME is unset; cannot resolve restore root")
	}
	if !request.Yes {
		if validateErr := cli.ValidateRestoreZip(resolvedPath); validateErr != nil {
			return validateErr
		}
		if request.JSON {
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
	if err := cli.RestoreFromZip(ctx, resolvedPath, home); err != nil {
		return err
	}
	if request.JSON {
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
}

func normalizeSeams(seams Seams) Seams {
	if seams.BackupsDir == nil {
		seams.BackupsDir = DefaultBackupsDir
	}
	if seams.HomeDir == nil {
		seams.HomeDir = DefaultHomeDir
	}
	return seams
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

func RunList(out io.Writer, backupsDir string, asJSON bool, options Options) error {
	entries, err := cli.ListBackups(backupsDir)
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
		fmt.Fprintln(out, "no backups found in "+backupsDir)
		return nil
	}
	fmt.Fprintf(out, "Backups in %s (newest first):\n", backupsDir)
	now := time.Now()
	for _, e := range entries {
		fmt.Fprintf(out, "  %s  %s  %s\n", filepath.Base(e.Path), FormatSize(e.SizeBytes), FormatAge(now.Sub(e.ModTime)))
	}
	return nil
}

func DefaultBackupsDir() string {
	return filepath.Join(config.GormesHome(), "lifecycle", "backups")
}

func DefaultHomeDir() string {
	return config.GormesHome()
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
