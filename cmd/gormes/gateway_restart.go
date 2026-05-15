package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const defaultGatewayServiceName = "gormes-gateway.service"

type gatewayRestartReportJSON struct {
	Build         buildProvenanceJSON `json:"build"`
	Action        string              `json:"action"`
	Mode          string              `json:"mode,omitempty"`
	Manager       string              `json:"manager,omitempty"`
	Service       string              `json:"service,omitempty"`
	Outcome       string              `json:"outcome,omitempty"`
	OldPID        int                 `json:"old_pid,omitempty"`
	NewPID        int                 `json:"new_pid,omitempty"`
	InitialStatus string              `json:"initial_status,omitempty"`
	FinalStatus   string              `json:"final_status,omitempty"`
	Message       string              `json:"message,omitempty"`
}

type gatewayRestartRuntimeStore interface {
	ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error)
}

type gatewayRestartServiceManager interface {
	Restart(context.Context, string) error
	ServiceActiveStatus(string) (cli.ServiceActiveStatusCheck, error)
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
	newGatewayRestartServiceManager = func() gatewayRestartServiceManager {
		if gatewayRuntimeGOOS == "linux" {
			if _, err := exec.LookPath("systemctl"); err == nil {
				return systemdGatewayRestartManager{}
			}
		}
		return nil
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

func newGatewayRestartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "restart",
		Short:        "Restart the live Gormes gateway",
		Long:         "Restart the live Gormes gateway through the platform service manager when available, falling back to the recorded runtime process.",
		SilenceUsage: true,
		RunE:         runGatewayRestart,
	}
	cmd.Flags().Duration("timeout", defaultGatewayStopTimeout, "maximum time to wait for the gateway to restart")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON with service-manager or runtime restart evidence")
	cmd.Flags().String("service", defaultGatewayServiceName, "user service name for service-manager restarts")
	return cmd
}

func runGatewayRestart(cmd *cobra.Command, _ []string) error {
	if gatewayRuntimeGOOS == "windows" {
		return runGatewayWindowsScheduledTaskCommand(cmd, "restart")
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	timeout, err := cmd.Flags().GetDuration("timeout")
	if err != nil {
		return err
	}
	if timeout < 0 {
		return fmt.Errorf("gateway restart: timeout must be non-negative")
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	service, _ := cmd.Flags().GetString("service")
	service = strings.TrimSpace(service)
	if service == "" {
		service = defaultGatewayServiceName
	}

	var serviceErr error
	if manager := newGatewayRestartServiceManager(); manager != nil {
		restartCtx := ctx
		cancelRestart := func() {}
		if timeout > 0 {
			restartCtx, cancelRestart = context.WithTimeout(ctx, timeout)
		}
		if err := manager.Restart(restartCtx, service); err == nil {
			cancelRestart()
			report := cli.PollServiceRestartActive(cli.ServiceRestartPollOptions{
				Service:      service,
				Runner:       manager,
				BaseTimeout:  timeout,
				PollInterval: gatewayRestartPollInterval,
			})
			if report.Outcome == cli.ServiceRestartPollRestarted {
				return writeGatewayRestartSuccess(cmd, asJSON, gatewayRestartReportJSON{
					Build:   newBuildProvenance(),
					Action:  "restarted",
					Mode:    "service",
					Manager: gatewayRestartServiceManagerName(),
					Service: service,
					Outcome: string(report.Outcome),
				})
			}
			return fmt.Errorf("gateway restart: service %s did not become active (outcome=%s)", service, report.Outcome)
		} else {
			cancelRestart()
			serviceErr = err
		}
	}

	report, err := restartRecordedGatewayRuntime(ctx, timeout)
	if err != nil {
		if serviceErr != nil {
			return fmt.Errorf("gateway restart: service-manager restart failed: %v; runtime fallback failed: %w", serviceErr, err)
		}
		return err
	}
	return writeGatewayRestartSuccess(cmd, asJSON, report)
}

func writeGatewayRestartSuccess(cmd *cobra.Command, asJSON bool, report gatewayRestartReportJSON) error {
	if asJSON {
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return nil
	}
	switch report.Mode {
	case "service":
		fmt.Fprintf(cmd.OutOrStdout(), "gateway restart: service %s restarted via %s\n", report.Service, report.Manager)
	case "runtime":
		if report.Action == "started" {
			fmt.Fprintf(cmd.OutOrStdout(), "gateway restart: no live gateway runtime; started pid=%d\n", report.NewPID)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "gateway restart: restarted pid=%d -> %d\n", report.OldPID, report.NewPID)
		}
	default:
		fmt.Fprintln(cmd.OutOrStdout(), "gateway restart: restarted")
	}
	return nil
}

func restartRecordedGatewayRuntime(ctx context.Context, timeout time.Duration) (gatewayRestartReportJSON, error) {
	store := newGatewayRestartRuntimeStore(config.GatewayRuntimeStatusPath())
	snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return gatewayRestartReportJSON{}, fmt.Errorf("runtime status: %w", err)
	}
	pid := gatewayStopPID(snapshot)
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
		TargetStartTime: gatewayStopStartTime(snapshot),
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
		Build:         newBuildProvenance(),
		Action:        "restarted",
		Mode:          "runtime",
		OldPID:        pid,
		NewPID:        gatewayStopPID(newSnapshot),
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
	newSnapshot, err := waitForGatewayRestartStart(startCtx, store, gatewayStopPID(initial))
	if err != nil {
		return gatewayRestartReportJSON{}, gatewayRestartStartWaitError(startCfg.LogPath, err)
	}
	return gatewayRestartReportJSON{
		Build:         newBuildProvenance(),
		Action:        "started",
		Mode:          "runtime",
		NewPID:        gatewayStopPID(newSnapshot),
		InitialStatus: string(initial.Validation.Status),
		FinalStatus:   string(newSnapshot.Validation.Status),
		Message:       "no live gateway runtime; started new gateway",
	}, nil
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

func waitForGatewayRestartStop(ctx context.Context, store gatewayRestartRuntimeStore, oldPID int) (gateway.RuntimeProcessValidation, error) {
	ticker := time.NewTicker(gatewayRestartPollInterval)
	defer ticker.Stop()
	for {
		snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
		if err != nil {
			return gateway.RuntimeProcessValidation{}, fmt.Errorf("runtime status: %w", err)
		}
		if !snapshot.Validation.Live || gatewayStopPID(snapshot) != oldPID {
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
		pid := gatewayStopPID(snapshot)
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
		LogPath: filepath.Join(home, "gateway.log"),
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

type systemdGatewayRestartManager struct{}

func (systemdGatewayRestartManager) Restart(ctx context.Context, service string) error {
	command := exec.CommandContext(ctx, "systemctl", "--user", "restart", service)
	out, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl --user restart %s: %w: %s", service, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (systemdGatewayRestartManager) ServiceActiveStatus(service string) (cli.ServiceActiveStatusCheck, error) {
	command := exec.Command("systemctl", "--user", "is-active", service)
	out, err := command.CombinedOutput()
	raw := strings.TrimSpace(string(out))
	status := parseSystemdGatewayActiveStatus(raw)
	check := cli.ServiceActiveStatusCheck{
		Status: status,
		Raw:    raw,
	}
	if err != nil && status == cli.ServiceActiveStatusUnknown {
		check.Unavailable = true
		check.Detail = err.Error()
	}
	return check, nil
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

func gatewayRestartServiceManagerName() string {
	if gatewayRuntimeGOOS == "linux" {
		return "systemd"
	}
	return "service"
}
