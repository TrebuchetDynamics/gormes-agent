package main

import (
	"encoding/json"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var progressPath string
	var asJSON bool
	cmd := &cobra.Command{
		Use:           "status",
		Short:         "Show Gormes runtime and progress blockers",
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				blockers, err := cli.CollectStatusBlockers(cli.StatusReportOptions{ProgressPath: progressPath})
				if err != nil {
					return err
				}
				if blockers == nil {
					blockers = []cli.StatusBlocker{}
				}
				body, err := json.MarshalIndent(map[string]any{"blockers": blockers}, "", "  ")
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return err
			}
			report, err := cli.RenderStatusReport(cli.StatusReportOptions{ProgressPath: progressPath})
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), report, renderSystemStatusLine(cmd))
			return err
		},
	}
	cmd.Flags().StringVar(&progressPath, "progress", cli.DefaultStatusProgressPath, "progress.json path used for blocker status")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a machine-readable {blockers: [...]} JSON document (suitable for monitoring/automation)")
	return cmd
}

func renderSystemStatusLine(cmd *cobra.Command) string {
	snapshot, err := cliSystemEventsManager().Snapshot(cmd.Context())
	if err != nil {
		return fmt.Sprintf("system: %s reason=status_unavailable audit=%s\n", toolspkg.SystemEventCodeUnavailable, config.ToolAuditLogPath())
	}
	return toolspkg.FormatSystemStatus(snapshot, config.ToolAuditLogPath()) + "\n"
}
