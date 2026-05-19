package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// gatewayStopReportJSON is the wire shape for `gateway stop --json`.
// Fleet automation orchestrating gateway lifecycle parses this to
// confirm the SIGINT landed on the right pid AND that the process
// actually exited within the timeout. `final_status` reflects the
// post-shutdown validation state (typically `stale_pid`).
type gatewayStopReportJSON struct {
	Build                    buildProvenanceJSON `json:"build"`
	Action                   string              `json:"action"`
	Live                     bool                `json:"live"`
	PID                      int                 `json:"pid,omitempty"`
	Signal                   string              `json:"signal,omitempty"`
	InitialStatus            string              `json:"initial_status,omitempty"`
	FinalStatus              string              `json:"final_status,omitempty"`
	Message                  string              `json:"message,omitempty"`
	PlannedStopMarkerWritten bool                `json:"planned_stop_marker_written"`
}

const defaultGatewayStopTimeout = 10 * time.Second

type gatewayStopRuntimeStore interface {
	ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error)
}

var (
	newGatewayStopRuntimeStore = func(path string) gatewayStopRuntimeStore {
		return gateway.NewRuntimeStatusStore(path)
	}
	signalGatewayStopProcess = func(pid int, signal os.Signal) error {
		if pid <= 0 {
			return fmt.Errorf("invalid pid %d", pid)
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return proc.Signal(signal)
	}
	gatewayStopPollInterval = 50 * time.Millisecond
)

func newGatewayStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "stop",
		Short:        "Stop the live Gormes gateway recorded in runtime status",
		SilenceUsage: true,
		RunE:         runGatewayStop,
	}
	cmd.Flags().Duration("timeout", defaultGatewayStopTimeout, "maximum time to wait for the gateway process to exit")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action: 'stopped'|'noop', live, pid, signal: 'SIGINT', initial_status, final_status, planned_stop_marker_written}`")
	return cmd
}

func runGatewayStop(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	timeout, err := cmd.Flags().GetDuration("timeout")
	if err != nil {
		return err
	}
	if timeout < 0 {
		return fmt.Errorf("gateway stop: timeout must be non-negative")
	}
	asJSON, _ := cmd.Flags().GetBool("json")

	store := newGatewayStopRuntimeStore(config.GatewayRuntimeStatusPath())
	snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("runtime status: %w", err)
	}
	pid := gatewayStopPID(snapshot)
	validation := snapshot.Validation
	if !validation.Live {
		if asJSON {
			return writeGatewayStopJSON(cmd.OutOrStdout(), gatewayStopReportJSON{
				Build:         newBuildProvenance(),
				Action:        "noop",
				Live:          false,
				PID:           pid,
				InitialStatus: string(validation.Status),
				Message:       validation.Message,
			})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "gateway stop: no live gateway runtime (status=%s", validation.Status)
		if pid > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), " pid=%d", pid)
		}
		if validation.Message != "" {
			fmt.Fprintf(cmd.OutOrStdout(), " message=%q", validation.Message)
		}
		fmt.Fprintln(cmd.OutOrStdout(), ")")
		return nil
	}
	if pid <= 0 {
		return fmt.Errorf("gateway stop: live runtime validation did not include a pid")
	}
	if snapshot.Status.ActiveAgents > 0 {
		return fmt.Errorf("gateway stop: refusing to stop live gateway with active_agents=%d", snapshot.Status.ActiveAgents)
	}

	markerStore := gateway.NewPlannedStopStore(gateway.DefaultPlannedStopMarkerPath(config.GatewayRuntimeStatusPath()))
	var plannedStopMarkerWritten bool
	if err := markerStore.Write(ctx, gateway.PlannedStopMarker{
		TargetPID:       pid,
		TargetStartTime: gatewayStopStartTime(snapshot),
		Generation:      snapshot.Status.Generation,
		Reason:          "gateway stop",
	}); err != nil {
		if !asJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "gateway stop: planned_stop_marker_unavailable pid=%d error=%q\n", pid, err.Error())
		}
	} else {
		plannedStopMarkerWritten = true
		if !asJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "gateway stop: planned_stop_marker_written pid=%d\n", pid)
		}
	}

	if err := signalGatewayStopProcess(pid, os.Interrupt); err != nil {
		return fmt.Errorf("gateway stop: signal pid %d: %w", pid, err)
	}
	if !asJSON {
		fmt.Fprintf(cmd.OutOrStdout(), "gateway stop: sent interrupt to pid=%d\n", pid)
	}
	if timeout == 0 {
		if asJSON {
			return writeGatewayStopJSON(cmd.OutOrStdout(), gatewayStopReportJSON{
				Build:                    newBuildProvenance(),
				Action:                   "stopped",
				Live:                     true,
				PID:                      pid,
				Signal:                   "SIGINT",
				InitialStatus:            string(validation.Status),
				PlannedStopMarkerWritten: plannedStopMarkerWritten,
			})
		}
		return nil
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	finalValidation, err := waitForGatewayStop(waitCtx, store)
	if err != nil {
		return err
	}

	wakeLockMgr := tools.TermuxWakeLockManager{}
	if err := wakeLockMgr.Release(ctx); err != nil {
		slog.Warn("gateway stop: termux-wake-lock release failed", "err", err)
	}

	if asJSON {
		return writeGatewayStopJSON(cmd.OutOrStdout(), gatewayStopReportJSON{
			Build:                    newBuildProvenance(),
			Action:                   "stopped",
			Live:                     true,
			PID:                      pid,
			Signal:                   "SIGINT",
			InitialStatus:            string(validation.Status),
			FinalStatus:              string(finalValidation.Status),
			Message:                  finalValidation.Message,
			PlannedStopMarkerWritten: plannedStopMarkerWritten,
		})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "gateway stop: stopped (status=%s", finalValidation.Status)
	if finalValidation.Message != "" {
		fmt.Fprintf(cmd.OutOrStdout(), " message=%q", finalValidation.Message)
	}
	fmt.Fprintln(cmd.OutOrStdout(), ")")
	return nil
}

func writeGatewayStopJSON(out interface{ Write(p []byte) (int, error) }, report gatewayStopReportJSON) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	return nil
}

func gatewayStopPID(snapshot gateway.RuntimeStatusSnapshot) int {
	if snapshot.Validation.PID > 0 {
		return snapshot.Validation.PID
	}
	return snapshot.Status.PID
}

func gatewayStopStartTime(snapshot gateway.RuntimeStatusSnapshot) int64 {
	if snapshot.Validation.ExpectedStartTime > 0 {
		return snapshot.Validation.ExpectedStartTime
	}
	if snapshot.Status.StartTime > 0 {
		return snapshot.Status.StartTime
	}
	return snapshot.Validation.ActualStartTime
}

func waitForGatewayStop(ctx context.Context, store gatewayStopRuntimeStore) (gateway.RuntimeProcessValidation, error) {
	ticker := time.NewTicker(gatewayStopPollInterval)
	defer ticker.Stop()
	for {
		snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
		if err != nil {
			return gateway.RuntimeProcessValidation{}, fmt.Errorf("runtime status: %w", err)
		}
		if !snapshot.Validation.Live {
			return snapshot.Validation, nil
		}
		select {
		case <-ctx.Done():
			return gateway.RuntimeProcessValidation{}, fmt.Errorf("gateway stop: timed out waiting for pid=%d to exit", gatewayStopPID(snapshot))
		case <-ticker.C:
		}
	}
}
