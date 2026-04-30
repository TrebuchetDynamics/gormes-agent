package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
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
	cmd := newRootCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmdArgs := append([]string{"gateway", sub}, args...)
	cmd.SetArgs(cmdArgs)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
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
