package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newSystemCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "Enqueue system events and inspect heartbeat or presence state",
	}
	cmd.AddCommand(newSystemEventCommand(), newSystemHeartbeatCommand(), newSystemPresenceCommand())
	return cmd
}

func newSystemEventCommand() *cobra.Command {
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
				return newExitCodeError(2, errors.New("system event: text is required"))
			}
			mode, err := parseSystemEventMode(modeFlag)
			if err != nil {
				return newExitCodeError(2, err)
			}
			result := cliSystemEventsManager().EnqueueEvent(cmd.Context(), toolspkg.SystemEventRequest{
				Text: text,
				Mode: mode,
			})
			writeSystemEventResult(cmd, result, jsonOut)
			if !result.OK {
				return newExitCodeError(1, errors.New(firstSystemDegradedMessage(result.Degraded)))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&textFlag, "text", "", "system event text")
	cmd.Flags().StringVar(&modeFlag, "mode", string(toolspkg.SystemEventModeNextHeartbeat), "wake mode: now or next-heartbeat")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	return cmd
}

func newSystemHeartbeatCommand() *cobra.Command {
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
			manager := cliSystemEventsManager()
			var result toolspkg.SystemEventResult
			switch action {
			case "", "last", "show", "status":
				snapshot, err := manager.Snapshot(cmd.Context())
				if err != nil {
					return err
				}
				result = toolspkg.SystemEventResult{
					OK:        len(snapshot.Degraded) == 0,
					Code:      toolspkg.SystemEventCodeHeartbeat,
					Heartbeat: toolspkg.SystemHeartbeatResult{Enabled: snapshot.Heartbeat.Enabled, LastBeatAt: snapshot.Heartbeat.LastBeatAt},
					Degraded:  snapshot.Degraded,
				}
			case "enable":
				result = manager.SetHeartbeat(cmd.Context(), true)
			case "disable":
				result = manager.SetHeartbeat(cmd.Context(), false)
			case "beat":
				result = manager.RecordHeartbeat(cmd.Context(), "manual")
			default:
				return newExitCodeError(2, fmt.Errorf("system heartbeat: unsupported action %q", action))
			}
			writeSystemEventResult(cmd, result, jsonOut)
			if !result.OK {
				return newExitCodeError(1, errors.New(firstSystemDegradedMessage(result.Degraded)))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	return cmd
}

func newSystemPresenceCommand() *cobra.Command {
	var component string
	var status string
	var reason string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "presence",
		Short: "List or update system presence entries",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager := cliSystemEventsManager()
			if strings.TrimSpace(component) != "" || strings.TrimSpace(status) != "" || strings.TrimSpace(reason) != "" {
				result := manager.UpdatePresence(cmd.Context(), toolspkg.SystemPresenceUpdate{
					Component: component,
					Status:    status,
					Reason:    reason,
				})
				if !result.OK {
					writeSystemPresenceResult(cmd, result, jsonOut)
					return newExitCodeError(1, errors.New(firstSystemDegradedMessage(result.Degraded)))
				}
			}
			snapshot, err := manager.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			writeSystemPresenceSnapshot(cmd, snapshot, jsonOut)
			if len(snapshot.Degraded) > 0 {
				return newExitCodeError(1, errors.New(firstSystemDegradedMessage(snapshot.Degraded)))
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

func cliSystemEventsManager() toolspkg.SystemEventsManager {
	return toolspkg.NewSystemEventsManager(toolspkg.SystemEventsOptions{
		StatePath: filepath.Join(config.GormesHome(), "system", "state.json"),
		AuditPath: config.ToolAuditLogPath(),
	})
}

func parseSystemEventMode(raw string) (toolspkg.SystemEventMode, error) {
	switch toolspkg.SystemEventMode(strings.TrimSpace(raw)) {
	case "", toolspkg.SystemEventModeNextHeartbeat:
		return toolspkg.SystemEventModeNextHeartbeat, nil
	case toolspkg.SystemEventModeNow:
		return toolspkg.SystemEventModeNow, nil
	default:
		return "", fmt.Errorf("system event: --mode must be now or next-heartbeat")
	}
}

func writeSystemEventResult(cmd *cobra.Command, result toolspkg.SystemEventResult, jsonOut bool) {
	if jsonOut {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
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

func writeSystemPresenceResult(cmd *cobra.Command, result toolspkg.SystemPresenceResult, jsonOut bool) {
	if jsonOut {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s ok=%t component=%s status=%s\n", result.Code, result.OK, result.Entry.Component, result.Entry.Status)
}

func writeSystemPresenceSnapshot(cmd *cobra.Command, snapshot toolspkg.SystemEventsSnapshot, jsonOut bool) {
	if jsonOut {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(snapshot)
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

func firstSystemDegradedMessage(items []toolspkg.SystemDegradedStatus) string {
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
