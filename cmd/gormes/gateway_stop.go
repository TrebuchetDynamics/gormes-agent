package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const defaultGatewayStopTimeout = 10 * time.Second

func init() {
	gatewayStopCmd.Flags().Duration("timeout", defaultGatewayStopTimeout, "maximum time to wait for the gateway process to exit")
	gatewayCmd.AddCommand(gatewayStopCmd)
}

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

var gatewayStopCmd = &cobra.Command{
	Use:          "stop",
	Short:        "Stop the live Gormes gateway recorded in runtime status",
	SilenceUsage: true,
	RunE:         runGatewayStop,
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

	store := newGatewayStopRuntimeStore(config.GatewayRuntimeStatusPath())
	snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("runtime status: %w", err)
	}
	pid := gatewayStopPID(snapshot)
	validation := snapshot.Validation
	if !validation.Live {
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

	if err := signalGatewayStopProcess(pid, os.Interrupt); err != nil {
		return fmt.Errorf("gateway stop: signal pid %d: %w", pid, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "gateway stop: sent interrupt to pid=%d\n", pid)
	if timeout == 0 {
		return nil
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	finalValidation, err := waitForGatewayStop(waitCtx, store)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "gateway stop: stopped (status=%s", finalValidation.Status)
	if finalValidation.Message != "" {
		fmt.Fprintf(cmd.OutOrStdout(), " message=%q", finalValidation.Message)
	}
	fmt.Fprintln(cmd.OutOrStdout(), ")")
	return nil
}

func gatewayStopPID(snapshot gateway.RuntimeStatusSnapshot) int {
	if snapshot.Validation.PID > 0 {
		return snapshot.Validation.PID
	}
	return snapshot.Status.PID
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
