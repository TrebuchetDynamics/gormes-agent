package gormescli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	hermesrowbacked "github.com/TrebuchetDynamics/gormes-agent/internal/app/hermesrowbacked"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/spf13/cobra"
)

const (
	HermesGatewayCronRow = hermesrowbacked.GatewayCronRow
	HermesDiagnosticsRow = hermesrowbacked.DiagnosticsRow
	HermesConfigRow      = hermesrowbacked.ConfigRow
	HermesToolRow        = hermesrowbacked.ToolRow
	HermesSkillsRow      = hermesrowbacked.SkillsRow
	HermesMemoryRow      = hermesrowbacked.MemoryRow
	HermesKanbanRow      = hermesrowbacked.KanbanRow
)

// NewDumpCommand returns the Hermes-compatible debug-dump placeholder surface.
func NewDumpCommand(opts RowBackedCommandOptions) *cobra.Command {
	return NewRowBackedCommand(newHermesRowBackedSpec(hermesrowbacked.DumpSpec()), opts)
}

// NewDebugCommand returns the Hermes-compatible debug bundle placeholder tree.
func NewDebugCommand(opts RowBackedCommandOptions) *cobra.Command {
	return newHermesRowBackedParent(
		"debug",
		"Manage Hermes-compatible debug share bundles",
		NewRowBackedCommand(newHermesRowBackedSpec(hermesrowbacked.DebugShareSpec()), opts),
		NewRowBackedCommand(newHermesRowBackedSpec(hermesrowbacked.DebugDeleteSpec()), opts),
	)
}

// NewBackupCommand returns a Hermes-compatible full-home backup surface. The
// generated zip uses the same exclusion and restore-compatible archive format
// as the update pre-backup flow.
func NewBackupCommand(opts RowBackedCommandOptions) *cobra.Command {
	build := opts.BuildProvenance
	if build == nil {
		build = func() BuildProvenance { return BuildProvenance{} }
	}
	var source string
	var output string
	var dryRun bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create a restore-compatible backup zip of GORMES_HOME",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolvedSource := source
			if resolvedSource == "" {
				resolvedSource = config.GormesHome()
			}
			resolvedOutput := output
			if resolvedOutput == "" {
				resolvedOutput = filepath.Join(resolvedSource, "backups", "pre-update-"+time.Now().UTC().Format("20060102T150405Z")+".zip")
			}
			if dryRun {
				return emitBackupReport(cmd, backupReportJSON{Build: build(), Action: "preview", Source: resolvedSource, Path: resolvedOutput, DryRun: true}, asJSON)
			}
			res, err := cli.WriteBackupZip(cmd.Context(), resolvedSource, resolvedOutput)
			if err != nil {
				return err
			}
			return emitBackupReport(cmd, backupReportJSON{Build: build(), Action: "backup_created", Source: resolvedSource, Path: res.Path, FileCount: res.FileCount, SizeBytes: res.SizeBytes}, asJSON)
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "directory to back up (default: GORMES_HOME)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "destination zip path (default: <source>/backups/pre-update-<UTC>.zip)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview the source and destination without writing a zip")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable backup result JSON")
	return cmd
}

type backupReportJSON struct {
	Build     BuildProvenance `json:"build"`
	Action    string          `json:"action"`
	Source    string          `json:"source"`
	Path      string          `json:"path"`
	DryRun    bool            `json:"dry_run,omitempty"`
	FileCount int             `json:"file_count,omitempty"`
	SizeBytes int64           `json:"size_bytes,omitempty"`
}

func emitBackupReport(cmd *cobra.Command, report backupReportJSON, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	if report.DryRun {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "backup preview: %s -> %s\n", report.Source, report.Path)
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "backup created: %s files=%d bytes=%d\n", report.Path, report.FileCount, report.SizeBytes)
	return err
}

// NewImportCommand returns the Hermes-compatible import placeholder surface.
func NewImportCommand(opts RowBackedCommandOptions) *cobra.Command {
	return NewRowBackedCommand(newHermesRowBackedSpec(hermesrowbacked.ImportSpec()), opts)
}

func newHermesRowBackedSpec(spec hermesrowbacked.Spec) RowBackedCommandSpec {
	return RowBackedCommandSpec{
		Use:         spec.Use,
		Short:       spec.Short,
		Row:         spec.Row,
		Destructive: spec.Destructive,
	}
}

func newHermesRowBackedParent(use, short string, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
	}
	cmd.AddCommand(children...)
	return cmd
}
