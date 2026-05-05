package main

import (
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
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
			_, err = fmt.Fprint(cmd.OutOrStdout(), report, renderSystemStatusLine(cmd))
			return err
		},
	}
	cmd.Flags().StringVar(&progressPath, "progress", cli.DefaultStatusProgressPath, "progress.json path used for blocker status")
	return cmd
}

func renderSystemStatusLine(cmd *cobra.Command) string {
	snapshot, err := cliSystemEventsManager().Snapshot(cmd.Context())
	if err != nil {
		return fmt.Sprintf("system: %s reason=status_unavailable audit=%s\n", toolspkg.SystemEventCodeUnavailable, config.ToolAuditLogPath())
	}
	return toolspkg.FormatSystemStatus(snapshot, config.ToolAuditLogPath()) + "\n"
}
