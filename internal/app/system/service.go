package system

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

var BuildProvenanceFunc = func() BuildProvenance { return BuildProvenance{} }

type exitCodeError struct {
	code int
	err  error
}

func (e exitCodeError) Error() string { return e.err.Error() }
func (e exitCodeError) Unwrap() error { return e.err }
func (e exitCodeError) ExitCode() int { return e.code }

func ParseEventMode(raw string) (tools.SystemEventMode, error) {
	switch tools.SystemEventMode(strings.TrimSpace(raw)) {
	case "", tools.SystemEventModeNextHeartbeat:
		return tools.SystemEventModeNextHeartbeat, nil
	case tools.SystemEventModeNow:
		return tools.SystemEventModeNow, nil
	default:
		return "", fmt.Errorf("system event: --mode must be now or next-heartbeat")
	}
}

func FirstDegradedMessage(items []tools.SystemDegradedStatus) string {
	if len(items) == 0 {
		return "system_unavailable"
	}
	item := items[0]
	if item.Message != "" {
		return item.Message
	}
	if item.Reason != "" {
		return item.Reason
	}
	return item.Code
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "Enqueue system events and inspect heartbeat or presence state",
	}
	cmd.AddCommand(newEventCommand(), newHeartbeatCommand(), newPresenceCommand())
	return cmd
}

func newEventCommand() *cobra.Command {
	var textFlag string
	var modeFlag string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "event [text]",
		Short: "Enqueue a system event and optionally wake heartbeat",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			text := strings.TrimSpace(textFlag)
			if text == "" {
				text = strings.TrimSpace(strings.Join(args, " "))
			}
			if text == "" {
				return exitCodeError{code: 2, err: errors.New("system event: text is required")}
			}
			mode, err := ParseEventMode(modeFlag)
			if err != nil {
				return exitCodeError{code: 2, err: err}
			}
			result := defaultManager().EnqueueEvent(cmd.Context(), tools.SystemEventRequest{
				Text: text,
				Mode: mode,
			})
			writeSystemEventResult(cmd, result, jsonOut)
			if !result.OK {
				return exitCodeError{code: 1, err: errors.New(FirstDegradedMessage(result.Degraded))}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&textFlag, "text", "", "system event text")
	cmd.Flags().StringVar(&modeFlag, "mode", string(tools.SystemEventModeNextHeartbeat), "wake mode: now or next-heartbeat")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	return cmd
}

func newHeartbeatCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "heartbeat [last|enable|disable|beat]",
		Short: "Show or control heartbeat state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			action := "last"
			if len(args) > 0 {
				action = strings.ToLower(strings.TrimSpace(args[0]))
			}
			manager := defaultManager()
			var result tools.SystemEventResult
			switch action {
			case "", "last", "show", "status":
				snapshot, err := manager.Snapshot(cmd.Context())
				if err != nil {
					return err
				}
				result = tools.SystemEventResult{
					OK:        len(snapshot.Degraded) == 0,
					Code:      tools.SystemEventCodeHeartbeat,
					Heartbeat: tools.SystemHeartbeatResult{Enabled: snapshot.Heartbeat.Enabled, LastBeatAt: snapshot.Heartbeat.LastBeatAt},
					Degraded:  snapshot.Degraded,
				}
			case "enable":
				result = manager.SetHeartbeat(cmd.Context(), true)
			case "disable":
				result = manager.SetHeartbeat(cmd.Context(), false)
			case "beat":
				result = manager.RecordHeartbeat(cmd.Context(), "manual")
			default:
				return exitCodeError{code: 2, err: fmt.Errorf("system heartbeat: unsupported action %q", action)}
			}
			writeSystemEventResult(cmd, result, jsonOut)
			if !result.OK {
				return exitCodeError{code: 1, err: errors.New(FirstDegradedMessage(result.Degraded))}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	return cmd
}

func newPresenceCommand() *cobra.Command {
	var component string
	var status string
	var reason string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "presence",
		Short: "List or update system presence entries",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager := defaultManager()
			if strings.TrimSpace(component) != "" || strings.TrimSpace(status) != "" || strings.TrimSpace(reason) != "" {
				result := manager.UpdatePresence(cmd.Context(), tools.SystemPresenceUpdate{
					Component: component,
					Status:    status,
					Reason:    reason,
				})
				if !result.OK {
					writeSystemPresenceResult(cmd, result, jsonOut)
					return exitCodeError{code: 1, err: errors.New(FirstDegradedMessage(result.Degraded))}
				}
			}
			snapshot, err := manager.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			writeSystemPresenceSnapshot(cmd, snapshot, jsonOut)
			if len(snapshot.Degraded) > 0 {
				return exitCodeError{code: 1, err: errors.New(FirstDegradedMessage(snapshot.Degraded))}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&component, "component", "", "component name to mark present before listing")
	cmd.Flags().StringVar(&status, "status", "", "component status for --component")
	cmd.Flags().StringVar(&reason, "reason", "", "component presence reason for --component")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	return cmd
}

func DefaultManager() tools.SystemEventsManager {
	return tools.NewSystemEventsManager(tools.SystemEventsOptions{
		StatePath: filepath.Join(config.GormesHome(), "system", "state.json"),
		AuditPath: config.ToolAuditLogPath(),
	})
}

func defaultManager() tools.SystemEventsManager { return DefaultManager() }

// systemEventReportJSON, systemPresenceReportJSON, and
// systemSnapshotReportJSON wrap the internal/tools result types with
// build provenance so fleet automation pushing runtime presence/event
// state across machines can attribute each JSON document to the binary
// version that emitted it. Existing top-level fields stay addressable
// through struct embedding — Go's JSON decoder ignores the unknown
// `build` field for callers parsing the old shape.
type systemEventReportJSON struct {
	Build BuildProvenance `json:"build"`
	tools.SystemEventResult
}

type systemPresenceReportJSON struct {
	Build BuildProvenance `json:"build"`
	tools.SystemPresenceResult
}

type systemSnapshotReportJSON struct {
	Build BuildProvenance `json:"build"`
	tools.SystemEventsSnapshot
}

func writeSystemEventResult(cmd *cobra.Command, result tools.SystemEventResult, jsonOut bool) {
	if jsonOut {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(systemEventReportJSON{
			Build:             BuildProvenanceFunc(),
			SystemEventResult: result,
		})
		return
	}
	if result.OK {
		fmt.Fprintf(cmd.OutOrStdout(), "%s ok\n", result.Code)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%s degraded\n", result.Code)
	}
	if result.Event.Text != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "event: %s\n", result.Event.Text)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "heartbeat: enabled=%t", result.Heartbeat.Enabled)
	if !result.Heartbeat.LastBeatAt.IsZero() {
		fmt.Fprintf(cmd.OutOrStdout(), " last_beat_at=%s", result.Heartbeat.LastBeatAt.Format("2006-01-02T15:04:05Z07:00"))
	}
	if result.Heartbeat.Triggered {
		fmt.Fprint(cmd.OutOrStdout(), " triggered=true")
	}
	fmt.Fprintln(cmd.OutOrStdout())
	for _, degraded := range result.Degraded {
		fmt.Fprintf(cmd.OutOrStdout(), "degraded: reason=%s path=%s message=%s\n", degraded.Reason, degraded.Path, degraded.Message)
	}
}

func writeSystemPresenceResult(cmd *cobra.Command, result tools.SystemPresenceResult, jsonOut bool) {
	if jsonOut {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(systemPresenceReportJSON{
			Build:                BuildProvenanceFunc(),
			SystemPresenceResult: result,
		})
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s ok=%t component=%s status=%s\n", result.Code, result.OK, result.Entry.Component, result.Entry.Status)
}

func writeSystemPresenceSnapshot(cmd *cobra.Command, snapshot tools.SystemEventsSnapshot, jsonOut bool) {
	if jsonOut {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(systemSnapshotReportJSON{
			Build:                BuildProvenanceFunc(),
			SystemEventsSnapshot: snapshot,
		})
		return
	}
	if len(snapshot.Presence) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "presence: none")
		return
	}
	for _, entry := range snapshot.Presence {
		fmt.Fprintf(cmd.OutOrStdout(), "%s status=%s last_seen_at=%s", entry.Component, entry.Status, entry.LastSeenAt.Format("2006-01-02T15:04:05Z07:00"))
		if entry.Reason != "" {
			fmt.Fprintf(cmd.OutOrStdout(), " reason=%q", entry.Reason)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}
}
