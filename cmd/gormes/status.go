package main

import (
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var progressPath string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show Gormes runtime and progress blockers",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := cli.RenderStatusReport(cli.StatusReportOptions{ProgressPath: progressPath})
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), report)
			return err
		},
	}
	cmd.Flags().StringVar(&progressPath, "progress", cli.DefaultStatusProgressPath, "progress.json path used for blocker status")
	return cmd
}
