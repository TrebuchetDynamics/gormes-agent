package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/agenttemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type agentResetOptions struct {
	Target string
	Force  bool
	DryRun bool
}

func newAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "agent",
		Short:        "Manage Gormes agent context templates",
		SilenceUsage: true,
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
	fmt.Fprintf(cmd.OutOrStdout(), "target: %s\n", result.TargetDir)
	for _, file := range result.Files {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", file.Action, file.Path)
	}
	return nil
}
