package gormescli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

const (
	defaultGatewayServiceName = "gormes-gateway.service"
	defaultGatewayStopTimeout = 10 * time.Second
)

type gatewayRestartReportJSON struct {
	Action        string `json:"action"`
	Mode          string `json:"mode,omitempty"`
	OldPID        int    `json:"old_pid,omitempty"`
	NewPID        int    `json:"new_pid,omitempty"`
	InitialStatus string `json:"initial_status,omitempty"`
	FinalStatus   string `json:"final_status,omitempty"`
	Message       string `json:"message,omitempty"`
}

type gatewayRestartRuntimeStore interface {
	ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error)
}

type gatewayRestartStartConfig struct {
	Command string
	Args    []string
	Env     []string
	LogPath string
}

var (
	newGatewayRestartRuntimeStore = func(path string) gatewayRestartRuntimeStore {
		return gateway.NewRuntimeStatusStore(path)
	}
	signalGatewayRestartProcess = func(pid int, signal os.Signal) error {
		if pid <= 0 {
			return fmt.Errorf("invalid pid %d", pid)
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return proc.Signal(signal)
	}
	startGatewayRestartProcess = startGatewayRestartProcessDetached
	gatewayRestartPollInterval = 50 * time.Millisecond
)

func restartRecordedGatewayRuntime(ctx context.Context, timeout time.Duration) (gatewayRestartReportJSON, error) {
	store := newGatewayRestartRuntimeStore(config.GatewayRuntimeStatusPath())
	snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return gatewayRestartReportJSON{}, fmt.Errorf("runtime status: %w", err)
	}
	return restartRecordedGatewayRuntimeFromSnapshot(ctx, timeout, store, snapshot)
}

func restartRecordedGatewayRuntimeFromSnapshot(ctx context.Context, timeout time.Duration, store gatewayRestartRuntimeStore, snapshot gateway.RuntimeStatusSnapshot) (gatewayRestartReportJSON, error) {
	pid := gatewayRestartPID(snapshot)
	validation := snapshot.Validation
	if !validation.Live {
		if canStartGatewayRuntimeWithoutLiveProcess(snapshot) {
			return startGatewayRuntimeWithoutLiveProcess(ctx, timeout, store, snapshot)
		}
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: no live gateway runtime (status=%s message=%q)", validation.Status, validation.Message)
	}
	if pid <= 0 {
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: live runtime validation did not include a pid")
	}
	if snapshot.Status.ActiveAgents > 0 {
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: refusing to restart live gateway with active_agents=%d", snapshot.Status.ActiveAgents)
	}

	markerStore := gateway.NewPlannedStopStore(gateway.DefaultPlannedStopMarkerPath(config.GatewayRuntimeStatusPath()))
	_ = markerStore.Write(ctx, gateway.PlannedStopMarker{
		TargetPID:       pid,
		TargetStartTime: gatewayRestartStartTime(snapshot),
		Generation:      snapshot.Status.Generation,
		Reason:          "gateway restart",
	})
	if err := signalGatewayRestartProcess(pid, os.Interrupt); err != nil {
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: signal pid %d: %w", pid, err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	finalStop, err := waitForGatewayRestartStop(waitCtx, store, pid)
	if err != nil {
		return gatewayRestartReportJSON{}, err
	}

	startCfg, err := defaultGatewayRestartStartConfig()
	if err != nil {
		return gatewayRestartReportJSON{}, err
	}
	if err := startGatewayRestartProcess(ctx, startCfg); err != nil {
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: start gateway: %w", err)
	}

	startCtx, cancelStart := context.WithTimeout(ctx, timeout)
	defer cancelStart()
	newSnapshot, err := waitForGatewayRestartStart(startCtx, store, pid)
	if err != nil {
		return gatewayRestartReportJSON{}, gatewayRestartStartWaitError(startCfg.LogPath, err)
	}
	return gatewayRestartReportJSON{
		Action:        "restarted",
		Mode:          "runtime",
		OldPID:        pid,
		NewPID:        gatewayRestartPID(newSnapshot),
		InitialStatus: string(validation.Status),
		FinalStatus:   string(finalStop.Status),
	}, nil
}

func canStartGatewayRuntimeWithoutLiveProcess(snapshot gateway.RuntimeStatusSnapshot) bool {
	if snapshot.Missing {
		return true
	}
	switch snapshot.Validation.Status {
	case gateway.RuntimeProcessValidationStalePID,
		gateway.RuntimeProcessValidationStopped:
		return true
	default:
		return false
	}
}

func startGatewayRuntimeWithoutLiveProcess(ctx context.Context, timeout time.Duration, store gatewayRestartRuntimeStore, initial gateway.RuntimeStatusSnapshot) (gatewayRestartReportJSON, error) {
	startCfg, err := defaultGatewayRestartStartConfig()
	if err != nil {
		return gatewayRestartReportJSON{}, err
	}
	if err := startGatewayRestartProcess(ctx, startCfg); err != nil {
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: start gateway: %w", err)
	}

	startCtx, cancelStart := context.WithTimeout(ctx, timeout)
	defer cancelStart()
	newSnapshot, err := waitForGatewayRestartStart(startCtx, store, gatewayRestartPID(initial))
	if err != nil {
		return gatewayRestartReportJSON{}, gatewayRestartStartWaitError(startCfg.LogPath, err)
	}
	return gatewayRestartReportJSON{
		Action:        "started",
		Mode:          "runtime",
		NewPID:        gatewayRestartPID(newSnapshot),
		InitialStatus: string(initial.Validation.Status),
		FinalStatus:   string(newSnapshot.Validation.Status),
		Message:       "no live gateway runtime; started new gateway",
	}, nil
}

func waitForGatewayRestartStop(ctx context.Context, store gatewayRestartRuntimeStore, oldPID int) (gateway.RuntimeProcessValidation, error) {
	ticker := time.NewTicker(gatewayRestartPollInterval)
	defer ticker.Stop()
	for {
		snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
		if err != nil {
			return gateway.RuntimeProcessValidation{}, fmt.Errorf("runtime status: %w", err)
		}
		if !snapshot.Validation.Live || gatewayRestartPID(snapshot) != oldPID {
			return snapshot.Validation, nil
		}
		select {
		case <-ctx.Done():
			return gateway.RuntimeProcessValidation{}, fmt.Errorf("gateway restart: timed out waiting for pid=%d to exit", oldPID)
		case <-ticker.C:
		}
	}
}

func waitForGatewayRestartStart(ctx context.Context, store gatewayRestartRuntimeStore, oldPID int) (gateway.RuntimeStatusSnapshot, error) {
	ticker := time.NewTicker(gatewayRestartPollInterval)
	defer ticker.Stop()
	for {
		snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
		if err != nil {
			return gateway.RuntimeStatusSnapshot{}, fmt.Errorf("runtime status: %w", err)
		}
		pid := gatewayRestartPID(snapshot)
		if snapshot.Validation.Live && pid > 0 && pid != oldPID {
			return snapshot, nil
		}
		select {
		case <-ctx.Done():
			return gateway.RuntimeStatusSnapshot{}, fmt.Errorf("gateway restart: timed out waiting for new gateway runtime")
		case <-ticker.C:
		}
	}
}

func defaultGatewayRestartStartConfig() (gatewayRestartStartConfig, error) {
	command, err := os.Executable()
	if err != nil || strings.TrimSpace(command) == "" {
		command = "gormes"
	}
	home := config.GormesHome()
	if strings.TrimSpace(home) == "" {
		userHome, userErr := os.UserHomeDir()
		if userErr != nil {
			return gatewayRestartStartConfig{}, fmt.Errorf("gateway restart: resolve home: %w", userErr)
		}
		home = filepath.Join(userHome, ".gormes")
	}
	env := append([]string{}, os.Environ()...)
	env = append(env, "GORMES_HOME="+home)
	return gatewayRestartStartConfig{
		Command: command,
		Args:    []string{"gateway"},
		Env:     env,
		LogPath: filepath.Join(home, "runtime", "gateway.log"),
	}, nil
}

func startGatewayRestartProcessDetached(_ context.Context, cfg gatewayRestartStartConfig) error {
	if strings.TrimSpace(cfg.Command) == "" {
		return fmt.Errorf("missing command")
	}
	if err := os.MkdirAll(filepath.Dir(cfg.LogPath), 0o755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer logFile.Close()

	start := exec.Command(cfg.Command, cfg.Args...)
	start.Env = cfg.Env
	start.Stdin = nil
	start.Stdout = logFile
	start.Stderr = logFile
	if err := start.Start(); err != nil {
		return err
	}
	if start.Process != nil {
		_ = start.Process.Release()
	}
	return nil
}

func gatewayRestartStartWaitError(logPath string, err error) error {
	if evidence := gatewayRestartStartupEvidence(logPath); evidence != "" {
		return fmt.Errorf("gateway restart: start gateway: %w: %s", err, evidence)
	}
	return fmt.Errorf("gateway restart: start gateway: %w", err)
}

func gatewayRestartStartupEvidence(logPath string) string {
	if strings.TrimSpace(logPath) == "" {
		return ""
	}
	raw, err := os.ReadFile(logPath)
	if err != nil {
		return ""
	}
	text := string(raw)
	if strings.Contains(text, "no channels configured") {
		return "no channels configured"
	}
	return ""
}

func gatewayRestartPID(snapshot gateway.RuntimeStatusSnapshot) int {
	if snapshot.Validation.PID > 0 {
		return snapshot.Validation.PID
	}
	return snapshot.Status.PID
}

func gatewayRestartStartTime(snapshot gateway.RuntimeStatusSnapshot) int64 {
	if snapshot.Validation.ExpectedStartTime > 0 {
		return snapshot.Validation.ExpectedStartTime
	}
	if snapshot.Status.StartTime > 0 {
		return snapshot.Status.StartTime
	}
	return snapshot.Validation.ActualStartTime
}

func parseSystemdGatewayActiveStatus(raw string) cli.ServiceActiveStatus {
	switch strings.TrimSpace(raw) {
	case "active":
		return cli.ServiceActiveStatusActive
	case "inactive":
		return cli.ServiceActiveStatusInactive
	case "activating", "deactivating", "reloading":
		return cli.ServiceActiveStatusActivating
	case "failed":
		return cli.ServiceActiveStatusFailed
	default:
		return cli.ServiceActiveStatusUnknown
	}
}
