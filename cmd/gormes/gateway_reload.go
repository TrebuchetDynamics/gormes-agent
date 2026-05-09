package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// gatewayReloadReportJSON is the wire shape for `gateway reload --json`.
// Fleet rollout automation triggering SIGHUP after config changes parses
// this to confirm the signal landed on the right pid. `action: "reloaded"`
// distinguishes a real reload from `action: "noop"` when no live runtime
// exists.
type gatewayReloadReportJSON struct {
	Build   buildProvenanceJSON `json:"build"`
	Action  string              `json:"action"`
	Live    bool                `json:"live"`
	PID     int                 `json:"pid,omitempty"`
	Signal  string              `json:"signal,omitempty"`
	Status  string              `json:"status,omitempty"`
	Message string              `json:"message,omitempty"`
}

type gatewayReloadRuntimeStore interface {
	ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error)
}

var (
	newGatewayReloadRuntimeStore = func(path string) gatewayReloadRuntimeStore {
		return gateway.NewRuntimeStatusStore(path)
	}
	signalGatewayReloadProcess = func(pid int, signal os.Signal) error {
		if pid <= 0 {
			return fmt.Errorf("invalid pid %d", pid)
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return proc.Signal(signal)
	}
)

func newGatewayReloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "reload",
		Short:        "Reload live Gormes gateway config without restarting",
		SilenceUsage: true,
		RunE:         runGatewayReload,
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action: 'reloaded'|'noop', live, pid, signal: 'SIGHUP', status, message}`")
	return cmd
}

func runGatewayReload(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	store := newGatewayReloadRuntimeStore(config.GatewayRuntimeStatusPath())
	snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("runtime status: %w", err)
	}
	pid := gatewayReloadPID(snapshot)
	validation := snapshot.Validation
	if !validation.Live {
		if asJSON {
			return writeGatewayReloadJSON(cmd.OutOrStdout(), gatewayReloadReportJSON{
				Build:   newBuildProvenance(),
				Action:  "noop",
				Live:    false,
				PID:     pid,
				Status:  string(validation.Status),
				Message: validation.Message,
			})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "gateway reload: no live gateway runtime (status=%s", validation.Status)
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
		return fmt.Errorf("gateway reload: live runtime validation did not include a pid")
	}
	if err := signalGatewayReloadProcess(pid, syscall.SIGHUP); err != nil {
		return fmt.Errorf("gateway reload: signal pid %d: %w", pid, err)
	}
	if asJSON {
		return writeGatewayReloadJSON(cmd.OutOrStdout(), gatewayReloadReportJSON{
			Build:  newBuildProvenance(),
			Action: "reloaded",
			Live:   true,
			PID:    pid,
			Signal: "SIGHUP",
			Status: string(validation.Status),
		})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "gateway reload: sent hangup to pid=%d\n", pid)
	return nil
}

func writeGatewayReloadJSON(out interface{ Write(p []byte) (int, error) }, report gatewayReloadReportJSON) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	return nil
}

func gatewayReloadPID(snapshot gateway.RuntimeStatusSnapshot) int {
	if snapshot.Validation.PID > 0 {
		return snapshot.Validation.PID
	}
	return snapshot.Status.PID
}
