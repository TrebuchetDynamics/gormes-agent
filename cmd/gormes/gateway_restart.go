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

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
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
	var initial gatewayRestartInitialSnapshot
	if manager := newGatewayRestartServiceManager(); manager != nil {
		initial = readGatewayRestartInitialSnapshot(ctx)
		report, err := restartGatewayViaServiceManager(ctx, timeout, manager, service)
		if err == nil {
			report, err = ensureServiceRestartTookOverRecordedRuntime(ctx, timeout, manager, service, initial, report)
			if err != nil {
				return err
			}
			return writeGatewayRestartSuccess(cmd, asJSON, report)
		}
		serviceErr = err
	}

	report, err := restartRecordedGatewayRuntimeAfterServiceFailure(ctx, timeout, initial)
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

type gatewayRestartInitialSnapshot struct {
	store    gatewayRestartRuntimeStore
	snapshot gateway.RuntimeStatusSnapshot
	ok       bool
}

func readGatewayRestartInitialSnapshot(ctx context.Context) gatewayRestartInitialSnapshot {
	store := newGatewayRestartRuntimeStore(config.GatewayRuntimeStatusPath())
	snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return gatewayRestartInitialSnapshot{store: store}
	}
	return gatewayRestartInitialSnapshot{store: store, snapshot: snapshot, ok: true}
}

func restartGatewayViaServiceManager(ctx context.Context, timeout time.Duration, manager gatewayRestartServiceManager, service string) (gatewayRestartReportJSON, error) {
	restartCtx := ctx
	cancelRestart := func() {}
	if timeout > 0 {
		restartCtx, cancelRestart = context.WithTimeout(ctx, timeout)
	}
	if err := manager.Restart(restartCtx, service); err != nil {
		cancelRestart()
		return gatewayRestartReportJSON{}, err
	}
	cancelRestart()
	report := cli.PollServiceRestartActive(cli.ServiceRestartPollOptions{
		Service:      service,
		Runner:       manager,
		BaseTimeout:  timeout,
		PollInterval: gatewayRestartPollInterval,
	})
	if report.Outcome != cli.ServiceRestartPollRestarted {
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: service %s did not become active (outcome=%s)", service, report.Outcome)
	}
	return gatewayRestartReportJSON{
		Build:   newBuildProvenance(),
		Action:  "restarted",
		Mode:    "service",
		Manager: gatewayRestartServiceManagerName(),
		Service: service,
		Outcome: string(report.Outcome),
	}, nil
}

func restartRecordedGatewayRuntimeAfterServiceFailure(ctx context.Context, timeout time.Duration, initial gatewayRestartInitialSnapshot) (gatewayRestartReportJSON, error) {
	if initial.ok && initial.store != nil {
		return restartRecordedGatewayRuntimeFromSnapshot(ctx, timeout, initial.store, initial.snapshot)
	}
	return restartRecordedGatewayRuntime(ctx, timeout)
}

func restartRecordedGatewayRuntime(ctx context.Context, timeout time.Duration) (gatewayRestartReportJSON, error) {
	store := newGatewayRestartRuntimeStore(config.GatewayRuntimeStatusPath())
	snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return gatewayRestartReportJSON{}, fmt.Errorf("runtime status: %w", err)
	}
	return restartRecordedGatewayRuntimeFromSnapshot(ctx, timeout, store, snapshot)
}

func restartRecordedGatewayRuntimeFromSnapshot(ctx context.Context, timeout time.Duration, store gatewayRestartRuntimeStore, snapshot gateway.RuntimeStatusSnapshot) (gatewayRestartReportJSON, error) {
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

func ensureServiceRestartTookOverRecordedRuntime(ctx context.Context, timeout time.Duration, manager gatewayRestartServiceManager, service string, initial gatewayRestartInitialSnapshot, report gatewayRestartReportJSON) (gatewayRestartReportJSON, error) {
	if !initial.ok || initial.store == nil || !initial.snapshot.Validation.Live {
		return report, nil
	}
	current, err := initial.store.ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: verify service takeover runtime status: %w", err)
	}
	if !sameLiveGatewayRestartRuntime(initial.snapshot, current) {
		return report, nil
	}
	pid := gatewayStopPID(current)
	if pid <= 0 {
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: service reported active but recorded runtime owner did not include a pid")
	}
	if current.Status.ActiveAgents > 0 {
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: service reported active but previous runtime pid=%d still owns active_agents=%d", pid, current.Status.ActiveAgents)
	}

	markerStore := gateway.NewPlannedStopStore(gateway.DefaultPlannedStopMarkerPath(config.GatewayRuntimeStatusPath()))
	_ = markerStore.Write(ctx, gateway.PlannedStopMarker{
		TargetPID:       pid,
		TargetStartTime: gatewayStopStartTime(current),
		Generation:      current.Status.Generation,
		Reason:          "gateway service-manager restart takeover",
	})
	if err := signalGatewayRestartProcess(pid, os.Interrupt); err != nil {
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: stop previous recorded runtime pid %d: %w", pid, err)
	}

	waitCtx, cancelWait := context.WithTimeout(ctx, timeout)
	defer cancelWait()
	if _, err := waitForGatewayRestartStop(waitCtx, initial.store, pid); err != nil {
		return gatewayRestartReportJSON{}, err
	}

	retried, err := restartGatewayViaServiceManager(ctx, timeout, manager, service)
	if err != nil {
		return gatewayRestartReportJSON{}, fmt.Errorf("gateway restart: retry service-manager restart after stopping previous runtime: %w", err)
	}
	startCtx, cancelStart := context.WithTimeout(ctx, timeout)
	defer cancelStart()
	newSnapshot, err := waitForGatewayRestartStart(startCtx, initial.store, pid)
	if err != nil {
		return gatewayRestartReportJSON{}, err
	}
	retried.OldPID = pid
	retried.NewPID = gatewayStopPID(newSnapshot)
	retried.InitialStatus = string(current.Validation.Status)
	retried.FinalStatus = string(newSnapshot.Validation.Status)
	retried.Message = fmt.Sprintf("service-manager restart did not stop previous recorded runtime; stopped previous live runtime pid=%d and retried service restart", pid)
	return retried, nil
}

func sameLiveGatewayRestartRuntime(first, second gateway.RuntimeStatusSnapshot) bool {
	if !first.Validation.Live || !second.Validation.Live {
		return false
	}
	firstPID := gatewayStopPID(first)
	secondPID := gatewayStopPID(second)
	if firstPID <= 0 || secondPID != firstPID {
		return false
	}
	firstStart := gatewayStopStartTime(first)
	secondStart := gatewayStopStartTime(second)
	return firstStart == 0 || secondStart == 0 || firstStart == secondStart
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
