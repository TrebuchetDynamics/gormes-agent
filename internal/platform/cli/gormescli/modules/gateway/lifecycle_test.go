package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/spf13/cobra"
)

func TestGatewayMutatingSubcommandsAreUnavailable(t *testing.T) {
	if len(gatewayMutatingUnavailableSubcommands) == 0 {
		t.Fatalf("no mutating subcommands registered")
	}
	for _, sub := range gatewayMutatingUnavailableSubcommands {
		t.Run(sub, func(t *testing.T) {
			setupGatewayStatusTestEnv(t)

			stdout, stderr, err := executeGatewayMutatingCommand(t, sub)
			if err == nil {
				t.Fatalf("expected error from `gateway %s`; got nil\nstdout=%s\nstderr=%s", sub, stdout, stderr)
			}

			wantMsg := "gateway: " + sub + " is not available; use the service_restart helper"
			if !strings.Contains(err.Error(), wantMsg) {
				t.Fatalf("error %q does not contain %q", err.Error(), wantMsg)
			}

			if code := exitCodeFromError(err); code == 0 {
				t.Fatalf("expected non-zero exit code from `gateway %s`, got %d", sub, code)
			}

			for _, path := range []string{
				config.GatewayRuntimeStatusPath(),
				config.SessionDBPath(),
				config.MemoryDBPath(),
			} {
				if _, statErr := os.Stat(path); statErr == nil {
					t.Fatalf("`gateway %s` opened or created runtime artifact %s", sub, path)
				} else if !os.IsNotExist(statErr) {
					t.Fatalf("stat runtime artifact %s: %v", path, statErr)
				}
			}
		})
	}
}

func TestGatewayWindowsScheduledTaskLifecycleCommands(t *testing.T) {
	for _, tc := range []struct {
		sub       string
		wantCalls []string
	}{
		{sub: "install", wantCalls: []string{"install", "start"}},
		{sub: "start", wantCalls: []string{"start"}},
		{sub: "restart", wantCalls: []string{"restart"}},
		{sub: "uninstall", wantCalls: []string{"uninstall"}},
	} {
		t.Run(tc.sub, func(t *testing.T) {
			setupGatewayStatusTestEnv(t)
			restoreGOOS := gatewayRuntimeGOOSForTest(t, "windows")
			defer restoreGOOS()
			runner := &fakeGatewayWindowsScheduledTaskRunner{}
			restoreRunner := gatewayWindowsScheduledTaskRunnerForTest(t, runner)
			defer restoreRunner()

			stdout, stderr, err := executeGatewayMutatingCommand(t, tc.sub)
			if err != nil {
				t.Fatalf("gateway %s: %v\nstdout=%s\nstderr=%s", tc.sub, err, stdout, stderr)
			}
			if strings.Join(runner.calls, ",") != strings.Join(tc.wantCalls, ",") {
				t.Fatalf("calls = %v, want %v", runner.calls, tc.wantCalls)
			}
			for _, want := range []string{"Scheduled Task", "gateway " + tc.sub} {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout missing %q:\n%s", want, stdout)
				}
			}
			for _, cfg := range runner.configs {
				if cfg.TaskName == "" || cfg.Command == "" {
					t.Fatalf("config missing task name or command: %+v", cfg)
				}
				if len(cfg.Args) != 1 || cfg.Args[0] != "gateway" {
					t.Fatalf("config args = %#v, want [gateway]", cfg.Args)
				}
			}
			assertGatewayStopDidNotOpenDurableStores(t)
		})
	}
}

func TestGatewayRestartUsesServiceManagerOnLinux(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	restoreGOOS := gatewayRuntimeGOOSForTest(t, "linux")
	defer restoreGOOS()
	runner := &fakeGatewayRestartServiceManager{
		statuses: []cli.ServiceActiveStatusCheck{{
			Status: cli.ServiceActiveStatusActive,
			Raw:    "active\n",
		}},
	}
	restoreService := gatewayRestartServiceManagerForTest(t, runner)
	defer restoreService()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "restart", "--timeout=100ms", "--json")
	if err != nil {
		t.Fatalf("gateway restart --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(runner.restarts) != 1 || runner.restarts[0] != defaultGatewayServiceName {
		t.Fatalf("service restarts = %v, want [%s]", runner.restarts, defaultGatewayServiceName)
	}

	var got struct {
		Action  string `json:"action"`
		Manager string `json:"manager"`
		Service string `json:"service"`
		Outcome string `json:"outcome"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("gateway restart --json output must be valid JSON: %v\nstdout=%s", err, stdout)
	}
	if got.Action != "restarted" || got.Manager != "systemd" || got.Service != defaultGatewayServiceName || got.Outcome != string(cli.ServiceRestartPollRestarted) {
		t.Fatalf("restart json = %+v, want service-manager restart evidence", got)
	}
}

func TestGatewayRestartServiceManagerTakesOverRecordedRuntimeOwner(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	restoreGOOS := gatewayRuntimeGOOSForTest(t, "linux")
	defer restoreGOOS()
	runner := &fakeGatewayRestartServiceManager{
		statuses: []cli.ServiceActiveStatusCheck{
			{Status: cli.ServiceActiveStatusActive, Raw: "active\n"},
			{Status: cli.ServiceActiveStatusActive, Raw: "active\n"},
		},
	}
	restoreService := gatewayRestartServiceManagerForTest(t, runner)
	defer restoreService()
	store := &fakeGatewayRestartRuntimeStore{
		snapshots: []gateway.RuntimeStatusSnapshot{
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          4242,
					StartTime:    100,
					Generation:   7,
					GatewayState: gateway.GatewayStateRunning,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status:            gateway.RuntimeProcessValidationLive,
					Live:              true,
					PID:               4242,
					ExpectedStartTime: 100,
				},
			},
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          4242,
					StartTime:    100,
					Generation:   7,
					GatewayState: gateway.GatewayStateRunning,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status:            gateway.RuntimeProcessValidationLive,
					Live:              true,
					PID:               4242,
					ExpectedStartTime: 100,
				},
			},
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          4242,
					GatewayState: gateway.GatewayStateStopped,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status:  gateway.RuntimeProcessValidationStalePID,
					Live:    false,
					PID:     4242,
					Message: "process is not running",
				},
			},
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          5151,
					StartTime:    200,
					GatewayState: gateway.GatewayStateRunning,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status:            gateway.RuntimeProcessValidationLive,
					Live:              true,
					PID:               5151,
					ExpectedStartTime: 200,
				},
			},
		},
	}
	restoreStore := gatewayRestartRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayRestartSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()
	starts := 0
	restoreStarter := gatewayRestartStarterForTest(t, func(context.Context, gatewayRestartStartConfig) error {
		starts++
		return nil
	})
	defer restoreStarter()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "restart", "--timeout=100ms", "--json")
	if err != nil {
		t.Fatalf("gateway restart service takeover: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(runner.restarts) != 2 {
		t.Fatalf("service restarts = %v, want initial restart plus retry after recorded owner stop", runner.restarts)
	}
	if len(signals) != 1 || signals[0].pid != 4242 || signals[0].signal != os.Interrupt {
		t.Fatalf("signals = %+v, want one interrupt for recorded owner pid 4242", signals)
	}
	if starts != 0 {
		t.Fatalf("detached starts = %d, want service-manager takeover without detached fallback", starts)
	}

	var got gatewayRestartReportJSON
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("gateway restart takeover output must be valid JSON: %v\nstdout=%s", err, stdout)
	}
	if got.Action != "restarted" || got.Mode != "service" || got.Manager != "systemd" || got.Service != defaultGatewayServiceName {
		t.Fatalf("restart takeover json = %+v, want service restarted evidence", got)
	}
	if !strings.Contains(got.Message, "stopped previous live runtime pid=4242") {
		t.Fatalf("restart takeover message = %q, want recorded-owner cleanup evidence", got.Message)
	}
}

func TestGatewayRestartServiceManagerUsesTimeoutContext(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	restoreGOOS := gatewayRuntimeGOOSForTest(t, "linux")
	defer restoreGOOS()
	runner := &fakeGatewayRestartServiceManager{
		statuses: []cli.ServiceActiveStatusCheck{{
			Status: cli.ServiceActiveStatusActive,
			Raw:    "active\n",
		}},
	}
	restoreService := gatewayRestartServiceManagerForTest(t, runner)
	defer restoreService()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "restart", "--timeout=250ms", "--json")
	if err != nil {
		t.Fatalf("gateway restart --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(runner.restartHadDeadlines) != 1 || !runner.restartHadDeadlines[0] {
		t.Fatalf("restart deadline evidence = %v, want one restart call with context deadline", runner.restartHadDeadlines)
	}
}

func TestGatewayRestartFallsBackToRecordedLiveRuntime(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	restoreGOOS := gatewayRuntimeGOOSForTest(t, "linux")
	defer restoreGOOS()
	restoreService := gatewayRestartServiceManagerForTest(t, &fakeGatewayRestartServiceManager{
		restartErr: errors.New("unit not found"),
	})
	defer restoreService()
	store := &fakeGatewayRestartRuntimeStore{
		snapshots: []gateway.RuntimeStatusSnapshot{
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          4242,
					StartTime:    100,
					Generation:   7,
					GatewayState: gateway.GatewayStateRunning,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status:            gateway.RuntimeProcessValidationLive,
					Live:              true,
					PID:               4242,
					ExpectedStartTime: 100,
				},
			},
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          4242,
					GatewayState: gateway.GatewayStateStopped,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status:  gateway.RuntimeProcessValidationStalePID,
					Live:    false,
					PID:     4242,
					Message: "process is not running",
				},
			},
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          5151,
					GatewayState: gateway.GatewayStateRunning,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status: gateway.RuntimeProcessValidationLive,
					Live:   true,
					PID:    5151,
				},
			},
		},
	}
	restoreStore := gatewayRestartRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayRestartSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()
	starts := 0
	restoreStarter := gatewayRestartStarterForTest(t, func(context.Context, gatewayRestartStartConfig) error {
		starts++
		return nil
	})
	defer restoreStarter()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "restart", "--timeout=100ms", "--json")
	if err != nil {
		t.Fatalf("gateway restart fallback: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(signals) != 1 || signals[0].pid != 4242 || signals[0].signal != os.Interrupt {
		t.Fatalf("signals = %+v, want one interrupt for pid 4242", signals)
	}
	if starts != 1 {
		t.Fatalf("starts = %d, want 1", starts)
	}

	var got struct {
		Action string `json:"action"`
		Mode   string `json:"mode"`
		OldPID int    `json:"old_pid"`
		NewPID int    `json:"new_pid"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("gateway restart fallback output must be valid JSON: %v\nstdout=%s", err, stdout)
	}
	if got.Action != "restarted" || got.Mode != "runtime" || got.OldPID != 4242 || got.NewPID != 5151 {
		t.Fatalf("restart fallback json = %+v, want runtime restart evidence", got)
	}
}

func TestGatewayRestartStartsRuntimeWhenNoLiveRuntimeExists(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	restoreGOOS := gatewayRuntimeGOOSForTest(t, "linux")
	defer restoreGOOS()
	restoreService := gatewayRestartServiceManagerForTest(t, nil)
	defer restoreService()
	store := &fakeGatewayRestartRuntimeStore{
		snapshots: []gateway.RuntimeStatusSnapshot{
			{
				Missing: true,
				Validation: gateway.RuntimeProcessValidation{
					Status:  gateway.RuntimeProcessValidationMissingState,
					Live:    false,
					Message: "runtime status is missing",
				},
			},
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          5151,
					GatewayState: gateway.GatewayStateRunning,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status: gateway.RuntimeProcessValidationLive,
					Live:   true,
					PID:    5151,
				},
			},
		},
	}
	restoreStore := gatewayRestartRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayRestartSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()
	starts := 0
	restoreStarter := gatewayRestartStarterForTest(t, func(context.Context, gatewayRestartStartConfig) error {
		starts++
		return nil
	})
	defer restoreStarter()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "restart", "--timeout=100ms", "--json")
	if err != nil {
		t.Fatalf("gateway restart should start when no live runtime exists: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(signals) != 0 {
		t.Fatalf("signals = %+v, want none when no live runtime exists", signals)
	}
	if starts != 1 {
		t.Fatalf("starts = %d, want 1", starts)
	}

	var got struct {
		Action        string `json:"action"`
		Mode          string `json:"mode"`
		NewPID        int    `json:"new_pid"`
		InitialStatus string `json:"initial_status"`
		FinalStatus   string `json:"final_status"`
		Message       string `json:"message"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("gateway restart start output must be valid JSON: %v\nstdout=%s", err, stdout)
	}
	if got.Action != "started" || got.Mode != "runtime" || got.NewPID != 5151 {
		t.Fatalf("restart start json = %+v, want runtime start evidence for pid 5151", got)
	}
	if got.InitialStatus != string(gateway.RuntimeProcessValidationMissingState) || got.FinalStatus != string(gateway.RuntimeProcessValidationLive) {
		t.Fatalf("restart start statuses = %q -> %q, want missing_state -> live", got.InitialStatus, got.FinalStatus)
	}
	if !strings.Contains(got.Message, "no live gateway runtime") {
		t.Fatalf("restart start message = %q, want no-live-runtime evidence", got.Message)
	}
}

func TestGatewayWindowsScheduledTaskCommandLineMarksDetached(t *testing.T) {
	got := windowsScheduledTaskCommandLine(gatewayWindowsScheduledTaskConfig{
		Command: `C:\Program Files\Gormes\gormes.exe`,
		Args:    []string{"gateway"},
	})
	for _, want := range []string{
		`cmd.exe /d /c`,
		`set "GORMES_GATEWAY_DETACHED=1"&&`,
		`"C:\Program Files\Gormes\gormes.exe"`,
		`"gateway"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("scheduled task command line missing %q:\n%s", want, got)
		}
	}
}

func TestGatewayStopSignalsValidatedLiveRuntime(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	store := &fakeGatewayStopRuntimeStore{
		snapshots: []gateway.RuntimeStatusSnapshot{
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          4242,
					GatewayState: gateway.GatewayStateRunning,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status: gateway.RuntimeProcessValidationLive,
					Live:   true,
					PID:    4242,
				},
			},
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          4242,
					GatewayState: gateway.GatewayStateStopped,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status:  gateway.RuntimeProcessValidationStalePID,
					Live:    false,
					PID:     4242,
					Message: "process is not running",
				},
			},
		},
	}
	restoreStore := gatewayStopRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayStopSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "stop", "--timeout=100ms")
	if err != nil {
		t.Fatalf("gateway stop: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(signals) != 1 || signals[0].pid != 4242 || signals[0].signal != os.Interrupt {
		t.Fatalf("signals = %+v, want one interrupt for pid 4242", signals)
	}
	for _, want := range []string{
		"gateway stop: sent interrupt to pid=4242",
		"gateway stop: stopped",
		"stale_pid",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	assertGatewayStopDidNotOpenDurableStores(t)
}

func TestGatewayStopPlannedMarkerWrittenBeforeSignal(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	store := &fakeGatewayStopRuntimeStore{
		snapshots: []gateway.RuntimeStatusSnapshot{
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          4242,
					StartTime:    987654,
					Generation:   9,
					GatewayState: gateway.GatewayStateRunning,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status:            gateway.RuntimeProcessValidationLive,
					Live:              true,
					PID:               4242,
					ExpectedStartTime: 987654,
				},
			},
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          4242,
					GatewayState: gateway.GatewayStateStopped,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status:  gateway.RuntimeProcessValidationStalePID,
					Live:    false,
					PID:     4242,
					Message: "process is not running",
				},
			},
		},
	}
	restoreStore := gatewayStopRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayStopSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "stop", "--timeout=100ms")
	if err != nil {
		t.Fatalf("gateway stop: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(signals) != 1 {
		t.Fatalf("signals = %+v, want one signal after marker write", signals)
	}

	raw, err := os.ReadFile(gateway.DefaultPlannedStopMarkerPath(config.GatewayRuntimeStatusPath()))
	if err != nil {
		t.Fatalf("read planned stop marker: %v", err)
	}
	var marker gateway.PlannedStopMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		t.Fatalf("decode planned stop marker: %v\n%s", err, raw)
	}
	if marker.TargetPID != 4242 || marker.TargetStartTime != 987654 || marker.Generation != 9 {
		t.Fatalf("marker target = %+v, want pid/start/generation 4242/987654/9", marker)
	}
	if marker.Reason != "gateway stop" {
		t.Fatalf("marker reason = %q, want gateway stop", marker.Reason)
	}
	if marker.WrittenAt == "" || marker.StopperPID <= 0 {
		t.Fatalf("marker missing written_at or stopper_pid: %+v", marker)
	}
	if !strings.Contains(stdout, "planned_stop_marker_written") {
		t.Fatalf("stdout missing planned marker evidence:\n%s", stdout)
	}
	assertGatewayStopDidNotOpenDurableStores(t)
}

func TestGatewayStopRefusesActiveAgents(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	store := &fakeGatewayStopRuntimeStore{
		snapshots: []gateway.RuntimeStatusSnapshot{{
			Status: gateway.RuntimeStatus{
				Kind:         "gormes-gateway",
				PID:          4242,
				GatewayState: gateway.GatewayStateRunning,
				ActiveAgents: 1,
			},
			Validation: gateway.RuntimeProcessValidation{
				Status: gateway.RuntimeProcessValidationLive,
				Live:   true,
				PID:    4242,
			},
		}},
	}
	restoreStore := gatewayStopRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayStopSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "stop")
	if err == nil {
		t.Fatalf("gateway stop should reject active agents\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if len(signals) != 0 {
		t.Fatalf("signals = %+v, want none when active agents are running", signals)
	}
	if !strings.Contains(err.Error(), "active_agents=1") {
		t.Fatalf("error missing active agent evidence: %v", err)
	}
	assertGatewayStopDidNotOpenDurableStores(t)
}

func TestGatewayStopNoLiveRuntimeIsIdempotent(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	store := &fakeGatewayStopRuntimeStore{
		snapshots: []gateway.RuntimeStatusSnapshot{{
			Missing: true,
			Validation: gateway.RuntimeProcessValidation{
				Status:  gateway.RuntimeProcessValidationMissingState,
				Live:    false,
				Message: "runtime status is missing",
			},
		}},
	}
	restoreStore := gatewayStopRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayStopSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "stop")
	if err != nil {
		t.Fatalf("gateway stop should be idempotent: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(signals) != 0 {
		t.Fatalf("signals = %+v, want none for missing runtime", signals)
	}
	for _, want := range []string{
		"gateway stop: no live gateway runtime",
		"missing_state",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	assertGatewayStopDidNotOpenDurableStores(t)
}

func TestGatewayReloadSignalsValidatedLiveRuntimeWithHUP(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	store := &fakeGatewayReloadRuntimeStore{
		snapshot: gateway.RuntimeStatusSnapshot{
			Status: gateway.RuntimeStatus{
				Kind:         "gormes-gateway",
				PID:          4242,
				GatewayState: gateway.GatewayStateRunning,
				ActiveAgents: 2,
			},
			Validation: gateway.RuntimeProcessValidation{
				Status: gateway.RuntimeProcessValidationLive,
				Live:   true,
				PID:    4242,
			},
		},
	}
	restoreStore := gatewayReloadRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayReloadSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "reload")
	if err != nil {
		t.Fatalf("gateway reload: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(signals) != 1 || signals[0].pid != 4242 || signals[0].signal != syscall.SIGHUP {
		t.Fatalf("signals = %+v, want one SIGHUP for pid 4242", signals)
	}
	if !strings.Contains(stdout, "gateway reload: sent hangup to pid=4242") {
		t.Fatalf("stdout missing reload evidence:\n%s", stdout)
	}
	assertGatewayStopDidNotOpenDurableStores(t)
}

func TestGatewayReloadNoLiveRuntimeIsIdempotent(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	store := &fakeGatewayReloadRuntimeStore{
		snapshot: gateway.RuntimeStatusSnapshot{
			Missing: true,
			Validation: gateway.RuntimeProcessValidation{
				Status:  gateway.RuntimeProcessValidationMissingState,
				Live:    false,
				Message: "runtime status is missing",
			},
		},
	}
	restoreStore := gatewayReloadRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayReloadSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "reload")
	if err != nil {
		t.Fatalf("gateway reload should be idempotent: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(signals) != 0 {
		t.Fatalf("signals = %+v, want none for missing runtime", signals)
	}
	for _, want := range []string{
		"gateway reload: no live gateway runtime",
		"missing_state",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	assertGatewayStopDidNotOpenDurableStores(t)
}

// TestGatewayStop_JSONEmitsStructuredOutcome proves
// `gormes gateway stop --json --timeout=100ms` returns a parseable
// `{build, action: "stopped"|"noop", live, pid, signal: "SIGINT",
// initial_status, final_status, planned_stop_marker_written}` document
// so fleet automation orchestrating gateway lifecycle (deploy/restart
// cycles) can confirm the SIGINT landed on the right pid AND that the
// process actually exited within the timeout. `final_status` reflects
// the post-shutdown validation state (typically `stale_pid`).
func TestGatewayStop_JSONEmitsStructuredOutcome(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	store := &fakeGatewayStopRuntimeStore{
		snapshots: []gateway.RuntimeStatusSnapshot{
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          5555,
					GatewayState: gateway.GatewayStateRunning,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status: gateway.RuntimeProcessValidationLive,
					Live:   true,
					PID:    5555,
				},
			},
			{
				Status: gateway.RuntimeStatus{
					Kind:         "gormes-gateway",
					PID:          5555,
					GatewayState: gateway.GatewayStateStopped,
				},
				Validation: gateway.RuntimeProcessValidation{
					Status:  gateway.RuntimeProcessValidationStalePID,
					Live:    false,
					PID:     5555,
					Message: "process is not running",
				},
			},
		},
	}
	restoreStore := gatewayStopRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayStopSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "stop", "--timeout=100ms", "--json")
	if err != nil {
		t.Fatalf("gateway stop --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(signals) != 1 || signals[0].pid != 5555 || signals[0].signal != os.Interrupt {
		t.Fatalf("signals = %+v, want one interrupt for pid 5555", signals)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action                   string `json:"action"`
		Live                     bool   `json:"live"`
		PID                      int    `json:"pid"`
		Signal                   string `json:"signal"`
		InitialStatus            string `json:"initial_status"`
		FinalStatus              string `json:"final_status"`
		PlannedStopMarkerWritten bool   `json:"planned_stop_marker_written"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("gateway stop --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != testGatewayVersion {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, testGatewayVersion)
	}
	if got.Action != "stopped" {
		t.Errorf("action = %q, want %q", got.Action, "stopped")
	}
	if got.PID != 5555 {
		t.Errorf("pid = %d, want 5555", got.PID)
	}
	if got.Signal != "SIGINT" {
		t.Errorf("signal = %q, want %q", got.Signal, "SIGINT")
	}
	if got.FinalStatus == "" {
		t.Errorf("final_status must be populated")
	}
}

// TestGatewayStop_JSONNoopWhenNoLiveRuntime proves the JSON
// idempotent path when no gateway is running. Fleet automation
// can branch on `action: "noop"` and `live: false` instead of
// scraping "no live gateway runtime" prose.
func TestGatewayStop_JSONNoopWhenNoLiveRuntime(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	store := &fakeGatewayStopRuntimeStore{
		snapshots: []gateway.RuntimeStatusSnapshot{
			{
				Missing: true,
				Validation: gateway.RuntimeProcessValidation{
					Status:  gateway.RuntimeProcessValidationMissingState,
					Live:    false,
					Message: "runtime status is missing",
				},
			},
		},
	}
	restoreStore := gatewayStopRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayStopSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()

	stdout, _, err := executeGatewayMutatingCommand(t, "stop", "--json")
	if err != nil {
		t.Fatalf("gateway stop --json (no runtime): %v\nstdout=%s", err, stdout)
	}
	if len(signals) != 0 {
		t.Fatalf("signals = %+v, want none for missing runtime", signals)
	}
	var got struct {
		Action string `json:"action"`
		Live   bool   `json:"live"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("gateway stop --json (no runtime) must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Action != "noop" {
		t.Errorf("action = %q, want %q", got.Action, "noop")
	}
	if got.Live {
		t.Errorf("live must be false when no runtime")
	}
}

// TestGatewayReload_JSONEmitsStructuredOutcome proves
// `gormes gateway reload --json` returns a parseable
// `{build, action, live, pid, signal, status}` document so fleet
// rollout automation that triggers gateway reloads after config
// changes can confirm the SIGHUP landed on the right pid without
// scraping prose. `action` distinguishes "reloaded" from "noop".
func TestGatewayReload_JSONEmitsStructuredOutcome(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	store := &fakeGatewayReloadRuntimeStore{
		snapshot: gateway.RuntimeStatusSnapshot{
			Status: gateway.RuntimeStatus{
				Kind:         "gormes-gateway",
				PID:          7474,
				GatewayState: gateway.GatewayStateRunning,
			},
			Validation: gateway.RuntimeProcessValidation{
				Status: gateway.RuntimeProcessValidationLive,
				Live:   true,
				PID:    7474,
			},
		},
	}
	restoreStore := gatewayReloadRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayReloadSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()

	stdout, stderr, err := executeGatewayMutatingCommand(t, "reload", "--json")
	if err != nil {
		t.Fatalf("gateway reload --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(signals) != 1 || signals[0].pid != 7474 || signals[0].signal != syscall.SIGHUP {
		t.Fatalf("signals = %+v, want one SIGHUP for pid 7474", signals)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action string `json:"action"`
		Live   bool   `json:"live"`
		PID    int    `json:"pid"`
		Signal string `json:"signal"`
		Status string `json:"status"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("gateway reload --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != testGatewayVersion {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, testGatewayVersion)
	}
	if got.Action != "reloaded" {
		t.Errorf("action = %q, want %q", got.Action, "reloaded")
	}
	if !got.Live {
		t.Errorf("live must be true after reload")
	}
	if got.PID != 7474 {
		t.Errorf("pid = %d, want 7474", got.PID)
	}
	if got.Signal != "SIGHUP" {
		t.Errorf("signal = %q, want %q", got.Signal, "SIGHUP")
	}
}

// TestGatewayReload_JSONNoopWhenNoLiveRuntime proves that when the
// gateway isn't running, `--json` reports `action: "noop"` and
// `live: false` rather than returning an error. Fleet automation
// reloading config across many machines treats "no live runtime" as
// "nothing to do" and continues with the next host.
func TestGatewayReload_JSONNoopWhenNoLiveRuntime(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	store := &fakeGatewayReloadRuntimeStore{
		snapshot: gateway.RuntimeStatusSnapshot{
			Missing: true,
			Validation: gateway.RuntimeProcessValidation{
				Status:  gateway.RuntimeProcessValidationMissingState,
				Live:    false,
				Message: "runtime status is missing",
			},
		},
	}
	restoreStore := gatewayReloadRuntimeStoreForTest(t, store)
	defer restoreStore()
	var signals []gatewayStopSignal
	restoreSignal := gatewayReloadSignalForTest(t, func(pid int, signal os.Signal) error {
		signals = append(signals, gatewayStopSignal{pid: pid, signal: signal})
		return nil
	})
	defer restoreSignal()

	stdout, _, err := executeGatewayMutatingCommand(t, "reload", "--json")
	if err != nil {
		t.Fatalf("gateway reload --json (no runtime) should be idempotent: %v\nstdout=%s", err, stdout)
	}
	if len(signals) != 0 {
		t.Fatalf("signals = %+v, want none for missing runtime", signals)
	}
	var got struct {
		Action  string `json:"action"`
		Live    bool   `json:"live"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("gateway reload --json (no runtime) must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Action != "noop" {
		t.Errorf("action = %q, want %q", got.Action, "noop")
	}
	if got.Live {
		t.Errorf("live must be false when no runtime")
	}
}

func TestGatewayMutatingExitCodeIsStable(t *testing.T) {
	codes := make(map[string]int, len(gatewayMutatingUnavailableSubcommands))
	for _, sub := range gatewayMutatingUnavailableSubcommands {
		t.Run(sub, func(t *testing.T) {
			setupGatewayStatusTestEnv(t)
			_, _, err := executeGatewayMutatingCommand(t, sub)
			if err == nil {
				t.Fatalf("expected error from `gateway %s`", sub)
			}
			codes[sub] = exitCodeFromError(err)
		})
	}

	first := gatewayMutatingUnavailableSubcommands[0]
	want := codes[first]
	for _, sub := range gatewayMutatingUnavailableSubcommands[1:] {
		if got := codes[sub]; got != want {
			t.Fatalf("exit code drift: gateway %s = %d but gateway %s = %d", first, want, sub, got)
		}
	}
}

func TestGatewayMutatingDoesNotShadowGatewayStatus(t *testing.T) {
	setupGatewayStatusTestEnv(t)

	stdout, stderr, err := executeGatewayStatusCommand(t)
	if err != nil {
		t.Fatalf("gateway status broken after mutating subcommands registered: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"runtime: missing",
		"channels: none configured",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("gateway status stdout missing %q after mutating subcommands registered:\n%s", want, stdout)
		}
	}
	assertGatewayStatusDidNotOpenRuntimeStores(t)
}

func executeGatewayMutatingCommand(t *testing.T, sub string, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	stub := func(name string) func() *cobra.Command {
		return func() *cobra.Command {
			return &cobra.Command{Use: name, RunE: func(*cobra.Command, []string) error { return nil }}
		}
	}
	cmd := NewGatewayCommandWithSeams(GatewayCommandSeams{
		Run:            func(*cobra.Command, []string) error { return nil },
		StopCommand:    func() *cobra.Command { return NewStopCommand(testGatewayOptions()) },
		RestartCommand: func() *cobra.Command { return NewRestartCommand(testGatewayOptions()) },
		ReloadCommand:  func() *cobra.Command { return NewReloadCommand(testGatewayOptions()) },
		StatusCommand:  func() *cobra.Command { return NewStatusCommand(testGatewayOptions()) },
		FleetCommand:   stub("fleet"),
		DiscoverCommand: func() *cobra.Command {
			return NewDiscoverCommand(testGatewayOptions())
		},
		ProbeCommand: func() *cobra.Command { return NewProbeCommand(testGatewayOptions()) },
		UsageCostCommand: func() *cobra.Command {
			return NewUsageCostCommand(testGatewayOptions())
		},
		MutatingUnavailableCommand: func(name string) *cobra.Command {
			return NewMutatingUnavailableCommand(name, testGatewayOptions())
		},
		BootInstallCommand:   stub("boot-install"),
		BootUninstallCommand: stub("boot-uninstall"),
	}, testGatewayOptions())
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(append([]string{sub}, args...))
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}

func assertGatewayStopDidNotOpenDurableStores(t *testing.T) {
	t.Helper()
	for _, path := range []string{
		config.SessionDBPath(),
		config.MemoryDBPath(),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("gateway stop opened durable store %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat durable store %s: %v", path, err)
		}
	}
}

type fakeGatewayStopRuntimeStore struct {
	snapshots []gateway.RuntimeStatusSnapshot
	err       error
	reads     int
}

func (s *fakeGatewayStopRuntimeStore) ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error) {
	if s.err != nil {
		return gateway.RuntimeStatusSnapshot{}, s.err
	}
	if len(s.snapshots) == 0 {
		return gateway.RuntimeStatusSnapshot{}, nil
	}
	idx := s.reads
	if idx >= len(s.snapshots) {
		idx = len(s.snapshots) - 1
	}
	s.reads++
	return s.snapshots[idx], nil
}

type gatewayStopSignal struct {
	pid    int
	signal os.Signal
}

func gatewayStopRuntimeStoreForTest(t *testing.T, store gatewayStopRuntimeStore) func() {
	t.Helper()
	previous := newGatewayStopRuntimeStore
	newGatewayStopRuntimeStore = func(string) gatewayStopRuntimeStore {
		return store
	}
	return func() {
		newGatewayStopRuntimeStore = previous
	}
}

func gatewayStopSignalForTest(t *testing.T, signal func(int, os.Signal) error) func() {
	t.Helper()
	previous := signalGatewayStopProcess
	signalGatewayStopProcess = signal
	return func() {
		signalGatewayStopProcess = previous
	}
}

type fakeGatewayReloadRuntimeStore struct {
	snapshot gateway.RuntimeStatusSnapshot
	err      error
}

func (s *fakeGatewayReloadRuntimeStore) ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error) {
	if s.err != nil {
		return gateway.RuntimeStatusSnapshot{}, s.err
	}
	return s.snapshot, nil
}

func gatewayReloadRuntimeStoreForTest(t *testing.T, store gatewayReloadRuntimeStore) func() {
	t.Helper()
	previous := newGatewayReloadRuntimeStore
	newGatewayReloadRuntimeStore = func(string) gatewayReloadRuntimeStore {
		return store
	}
	return func() {
		newGatewayReloadRuntimeStore = previous
	}
}

func gatewayReloadSignalForTest(t *testing.T, signal func(int, os.Signal) error) func() {
	t.Helper()
	previous := signalGatewayReloadProcess
	signalGatewayReloadProcess = signal
	return func() {
		signalGatewayReloadProcess = previous
	}
}

func gatewayRuntimeGOOSForTest(t *testing.T, goos string) func() {
	t.Helper()
	previous := gatewayRuntimeGOOS
	gatewayRuntimeGOOS = goos
	return func() {
		gatewayRuntimeGOOS = previous
	}
}

func gatewayWindowsScheduledTaskRunnerForTest(t *testing.T, runner gatewayWindowsScheduledTaskRunner) func() {
	t.Helper()
	previous := gatewayWindowsTaskRunner
	gatewayWindowsTaskRunner = runner
	return func() {
		gatewayWindowsTaskRunner = previous
	}
}

type fakeGatewayWindowsScheduledTaskRunner struct {
	calls   []string
	configs []gatewayWindowsScheduledTaskConfig
}

func (r *fakeGatewayWindowsScheduledTaskRunner) Install(_ context.Context, cfg gatewayWindowsScheduledTaskConfig) error {
	r.calls = append(r.calls, "install")
	r.configs = append(r.configs, cfg)
	return nil
}

func (r *fakeGatewayWindowsScheduledTaskRunner) Start(_ context.Context, cfg gatewayWindowsScheduledTaskConfig) error {
	r.calls = append(r.calls, "start")
	r.configs = append(r.configs, cfg)
	return nil
}

func (r *fakeGatewayWindowsScheduledTaskRunner) Restart(_ context.Context, cfg gatewayWindowsScheduledTaskConfig) error {
	r.calls = append(r.calls, "restart")
	r.configs = append(r.configs, cfg)
	return nil
}

func (r *fakeGatewayWindowsScheduledTaskRunner) Uninstall(_ context.Context, cfg gatewayWindowsScheduledTaskConfig) error {
	r.calls = append(r.calls, "uninstall")
	r.configs = append(r.configs, cfg)
	return nil
}

type fakeGatewayRestartServiceManager struct {
	restarts            []string
	restartHadDeadlines []bool
	statuses            []cli.ServiceActiveStatusCheck
	restartErr          error
	statusErr           error
}

func (r *fakeGatewayRestartServiceManager) Restart(ctx context.Context, service string) error {
	r.restarts = append(r.restarts, service)
	_, hasDeadline := ctx.Deadline()
	r.restartHadDeadlines = append(r.restartHadDeadlines, hasDeadline)
	return r.restartErr
}

func (r *fakeGatewayRestartServiceManager) ServiceActiveStatus(service string) (cli.ServiceActiveStatusCheck, error) {
	if r.statusErr != nil {
		return cli.ServiceActiveStatusCheck{}, r.statusErr
	}
	idx := len(r.restarts) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(r.statuses) {
		idx = len(r.statuses) - 1
	}
	if idx < 0 {
		return cli.ServiceActiveStatusCheck{Status: cli.ServiceActiveStatusActive}, nil
	}
	return r.statuses[idx], nil
}

type fakeGatewayRestartRuntimeStore struct {
	snapshots []gateway.RuntimeStatusSnapshot
	err       error
	reads     int
}

func (s *fakeGatewayRestartRuntimeStore) ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error) {
	if s.err != nil {
		return gateway.RuntimeStatusSnapshot{}, s.err
	}
	if len(s.snapshots) == 0 {
		return gateway.RuntimeStatusSnapshot{}, nil
	}
	idx := s.reads
	if idx >= len(s.snapshots) {
		idx = len(s.snapshots) - 1
	}
	s.reads++
	return s.snapshots[idx], nil
}

func gatewayRestartServiceManagerForTest(t *testing.T, runner gatewayRestartServiceManager) func() {
	t.Helper()
	previous := newGatewayRestartServiceManager
	newGatewayRestartServiceManager = func() gatewayRestartServiceManager {
		return runner
	}
	return func() {
		newGatewayRestartServiceManager = previous
	}
}

func gatewayRestartRuntimeStoreForTest(t *testing.T, store gatewayRestartRuntimeStore) func() {
	t.Helper()
	previous := newGatewayRestartRuntimeStore
	newGatewayRestartRuntimeStore = func(string) gatewayRestartRuntimeStore {
		return store
	}
	return func() {
		newGatewayRestartRuntimeStore = previous
	}
}

func gatewayRestartSignalForTest(t *testing.T, signal func(int, os.Signal) error) func() {
	t.Helper()
	previous := signalGatewayRestartProcess
	signalGatewayRestartProcess = signal
	return func() {
		signalGatewayRestartProcess = previous
	}
}

func gatewayRestartStarterForTest(t *testing.T, start func(context.Context, gatewayRestartStartConfig) error) func() {
	t.Helper()
	previous := startGatewayRestartProcess
	startGatewayRestartProcess = start
	return func() {
		startGatewayRestartProcess = previous
	}
}

func TestGatewayRestartDetachedStartConfigUsesLogUnderGormesHome(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	cfg, err := defaultGatewayRestartStartConfig()
	if err != nil {
		t.Fatalf("defaultGatewayRestartStartConfig: %v", err)
	}
	if !strings.HasSuffix(cfg.Args[0], "gateway") {
		t.Fatalf("args = %#v, want gateway subcommand", cfg.Args)
	}
	if !strings.HasPrefix(cfg.LogPath, config.GormesHome()) {
		t.Fatalf("log path = %q, want under %q", cfg.LogPath, config.GormesHome())
	}
}
