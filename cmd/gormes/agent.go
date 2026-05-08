package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/agenttemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type agentResetOptions struct {
	Target string
	Force  bool
	DryRun bool
	JSON   bool
}

// agentResetReportJSON is the wire shape for `agent reset --json`.
// Fleet automation seeding agent context across many machines parses
// this to confirm which template files landed (or, in dry-run mode,
// which would land). Build provenance leads — same convention as the
// rest of the `--json` arc.
type agentResetReportJSON struct {
	Build  buildProvenanceJSON       `json:"build"`
	Target string                    `json:"target"`
	DryRun bool                      `json:"dry_run"`
	Files  []agentResetFileJSON      `json:"files"`
}

type agentResetFileJSON struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

func newAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "agent",
		Short:        "Manage Gormes agent context templates",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(newAgentResetCommand())
	return cmd
}

func newAgentResetCommand() *cobra.Command {
	opts := agentResetOptions{Target: config.GormesHome()}
	cmd := &cobra.Command{
		Use:          "reset",
		Short:        "Seed default Gormes agent context templates",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAgentResetCommand(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Target, "target", opts.Target, "target directory for agent context templates")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "overwrite existing template files")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "report reset actions without writing files")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "emit machine-readable JSON: `{build, target, dry_run, files: [{path, action}]}`")
	return cmd
}

func runAgentResetCommand(cmd *cobra.Command, opts agentResetOptions) error {
	result, err := agenttemplate.ApplyDefaultTemplates(agenttemplate.WriteOptions{
		TargetDir: opts.Target,
		Force:     opts.Force,
		DryRun:    opts.DryRun,
	})
	if err != nil {
		return fmt.Errorf("gormes agent reset: %w", err)
	}
	if opts.JSON {
		report := agentResetReportJSON{
			Build:  newBuildProvenance(),
			Target: result.TargetDir,
			DryRun: opts.DryRun,
			Files:  make([]agentResetFileJSON, len(result.Files)),
		}
		for i, f := range result.Files {
			report.Files[i] = agentResetFileJSON{
				Path:   f.Path,
				Action: string(f.Action),
			}
		}
		body, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "target: %s\n", result.TargetDir)
	for _, file := range result.Files {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", file.Action, file.Path)
	}
	return nil
}
