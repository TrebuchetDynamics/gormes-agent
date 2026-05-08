package main

import (
	"encoding/json"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/spf13/cobra"
)

// statusReportJSON is the wire shape for `gormes status --json`.
// Build provenance leads, then the blockers array — same convention
// as update --json / doctor --json. The `system` block + `audit_path`
// mirror the system-events line the text surface prints
// via renderSystemStatusLine, so JSON consumers can ingest the same
// information without scraping prose.
type statusReportJSON struct {
	Build     buildProvenanceJSON           `json:"build"`
	Blockers  []cli.StatusBlocker           `json:"blockers"`
	System    toolspkg.SystemEventsSnapshot `json:"system"`
	AuditPath string                        `json:"audit_path"`
}

func newStatusCommand() *cobra.Command {
	var progressPath string
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "status",
		Short:        "Show Gormes runtime and progress blockers",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				blockers, err := cli.CollectStatusBlockers(cli.StatusReportOptions{ProgressPath: progressPath})
				if err != nil {
					return err
				}
				if blockers == nil {
					blockers = []cli.StatusBlocker{}
				}
				system := collectSystemSnapshotForJSON(cmd)
				body, err := json.MarshalIndent(statusReportJSON{
					Build:     newBuildProvenance(),
					Blockers:  blockers,
					System:    system,
					AuditPath: config.ToolAuditLogPath(),
				}, "", "  ")
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

// collectSystemSnapshotForJSON returns a snapshot suitable for the
// JSON wire shape, normalizing nil slices to empty arrays so consumers
// can iterate without nil-checks (same convention as
// emitSessionListJSON returning `[]` for empty inventories).
func collectSystemSnapshotForJSON(cmd *cobra.Command) toolspkg.SystemEventsSnapshot {
	snapshot, err := cliSystemEventsManager().Snapshot(cmd.Context())
	if err != nil {
		return toolspkg.SystemEventsSnapshot{
			Events:   []toolspkg.SystemEvent{},
			Presence: []toolspkg.SystemPresenceEntry{},
		}
	}
	if snapshot.Events == nil {
		snapshot.Events = []toolspkg.SystemEvent{}
	}
	if snapshot.Presence == nil {
		snapshot.Presence = []toolspkg.SystemPresenceEntry{}
	}
	return snapshot
}
