package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// TestGatewayCommand_ConstructorReturnsIndependentInstances proves
// each newGatewayCommand() returns a fresh tree. With 6 real
// subcommands, 4 mutating-unavailable placeholders, and 4 row-backed
// Hermes parity placeholders, the parent must own 14 children per call.
func TestGatewayCommand_ConstructorReturnsIndependentInstances(t *testing.T) {
	a := newGatewayCommand()
	b := newGatewayCommand()
	if a == b {
		t.Fatal("newGatewayCommand must return distinct instances")
	}
	want := 14
	if got := len(a.Commands()); got != want {
		t.Fatalf("gateway tree must have %d subcommands; got %d", want, got)
	}
	if got := len(b.Commands()); got != want {
		t.Fatalf("gateway tree must have %d subcommands; got %d", want, got)
	}
	for i := range a.Commands() {
		if a.Commands()[i] == b.Commands()[i] {
			t.Fatalf("subcommand[%d] %q shared between constructor calls", i, a.Commands()[i].Use)
		}
	}
}

func TestGatewayStatusCommand_NoChannelsSucceedsWithoutOpeningRuntimeClients(t *testing.T) {
	setupGatewayStatusTestEnv(t)

	stdout, stderr, err := executeGatewayStatusCommand(t)
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{
		"runtime: missing",
		"channels: none configured",
		"- pairing missing: pairing state is missing",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q\n%s", want, stdout)
		}
	}
	assertGatewayStatusDidNotOpenRuntimeStores(t)
}

func TestGatewayStatusCommand_RendersConfiguredChannelsFromReadModels(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	writeGatewayStatusConfig(t, []byte(`
[telegram]
bot_token = "12345:bogus"
allowed_chat_id = 42

[discord]
token = "bogus-discord-token"
allowed_channel_id = "D123"
`))

	now := time.Now().UTC()
	pairing := gateway.NewXDGPairingStore()
	if err := pairing.RecordPendingPairing(context.Background(), gateway.PairingPendingRecord{
		Platform:  "telegram",
		UserID:    "telegram-user",
		UserName:  "Ada",
		Code:      "TGREADY",
		CreatedAt: now.Add(-2 * time.Minute),
	}); err != nil {
		t.Fatalf("record pending pairing: %v", err)
	}
	if err := pairing.RecordApprovedPairing(context.Background(), gateway.PairingApprovedRecord{
		Platform:   "discord",
		UserID:     "discord-owner",
		UserName:   "Grace",
		ApprovedAt: now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("record approved pairing: %v", err)
	}

	runtimeStatus := gateway.NewRuntimeStatusStore(config.GatewayRuntimeStatusPath())
	if err := runtimeStatus.UpdateRuntimeStatus(context.Background(), gateway.RuntimeStatusUpdate{
		GatewayState: gateway.GatewayStateRunning,
	}); err != nil {
		t.Fatalf("write gateway runtime: %v", err)
	}
	if err := runtimeStatus.UpdateRuntimeStatus(context.Background(), gateway.RuntimeStatusUpdate{
		Platform:      "telegram",
		PlatformState: gateway.PlatformStateRunning,
	}); err != nil {
		t.Fatalf("write telegram runtime: %v", err)
	}
	if err := runtimeStatus.UpdateRuntimeStatus(context.Background(), gateway.RuntimeStatusUpdate{
		Platform:      "discord",
		PlatformState: gateway.PlatformStateFailed,
		ErrorMessage:  "discord: open session: denied",
	}); err != nil {
		t.Fatalf("write discord runtime: %v", err)
	}

	stdout, stderr, err := executeGatewayStatusCommand(t)
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	discordRow := "- discord: lifecycle=failed error=\"discord: open session: denied\"; pairing=paired pending=0 approved=1; target=allowed_channel_id=D123"
	telegramRow := "- telegram: lifecycle=running; pairing=unpaired pending=1 approved=0; target=allowed_chat_id=42"
	if !strings.Contains(stdout, discordRow) {
		t.Fatalf("stdout missing discord row\n%s", stdout)
	}
	if !strings.Contains(stdout, telegramRow) {
		t.Fatalf("stdout missing telegram row\n%s", stdout)
	}
	if strings.Index(stdout, discordRow) > strings.Index(stdout, telegramRow) {
		t.Fatalf("channel rows are not sorted\n%s", stdout)
	}
	for _, want := range []string{
		"- pending telegram user=telegram-user code=TGREADY",
		"- approved discord user=discord-owner name=Grace",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q\n%s", want, stdout)
		}
	}
	assertGatewayStatusDidNotOpenRuntimeStores(t)
}

func TestGatewayStatusCommand_RendersRuntimePIDValidationEvidence(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	restoreRuntimeStore := gatewayStatusRuntimeStoreForTest(t, fakeGatewayStatusRuntimeStore{
		snapshot: gateway.RuntimeStatusSnapshot{
			Status: gateway.RuntimeStatus{
				Kind:         "gormes-gateway",
				PID:          4242,
				StartTime:    100,
				Generation:   3,
				Command:      "gormes gateway",
				GatewayState: gateway.GatewayStateStopped,
			},
			Validation: gateway.RuntimeProcessValidation{
				Status:            gateway.RuntimeProcessValidationStalePID,
				Live:              false,
				PID:               4242,
				ExpectedStartTime: 100,
				Message:           "process is not running",
			},
		},
	})
	defer restoreRuntimeStore()

	stdout, stderr, err := executeGatewayStatusCommand(t)
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{
		"runtime: stopped (pid=4242 active_agents=0)",
		"runtime_validation: stale_pid live=false pid=4242 expected_start_time=100 message=\"process is not running\"",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q\n%s", want, stdout)
		}
	}
	assertGatewayStatusDidNotOpenRuntimeStores(t)
}

// TestGatewayStatusCommand_JSONIncludesBuildProvenance proves
// `gormes gateway status --json` carries the running binary's build
// version + SHA. Same contract as
// update --json / doctor --json / status --json / restore --list --json /
// auth status --json / secrets ... --json — captured gateway snapshots
// stay attributable to a specific binary, which matters when operators
// correlate gateway behavior with the binary build that produced it.
func TestGatewayStatusCommand_JSONIncludesBuildProvenance(t *testing.T) {
	setupGatewayStatusTestEnv(t)

	stdout, _, err := executeGatewayStatusCommand(t, "--json")
	if err != nil {
		t.Fatalf("gateway status --json: %v", err)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version {
		t.Fatalf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Build.GitCommit == "" {
		t.Fatalf("got.build.git_commit must be non-empty")
	}
}

func TestGatewayStatusCommand_JSONRendersStableRuntimeFields(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	restoreRuntimeStore := gatewayStatusRuntimeStoreForTest(t, fakeGatewayStatusRuntimeStore{
		snapshot: gateway.RuntimeStatusSnapshot{
			Status: gateway.RuntimeStatus{
				Kind:         "gormes-gateway",
				PID:          4242,
				StartTime:    100,
				Generation:   3,
				Command:      "/home/xel/.gormes/bin/gormes gateway",
				GatewayState: gateway.GatewayStateRunning,
				ActiveAgents: 0,
				Platforms: map[string]gateway.PlatformRuntimeStatus{
					"telegram": {State: gateway.PlatformStateRunning},
				},
			},
			Validation: gateway.RuntimeProcessValidation{
				Status:            gateway.RuntimeProcessValidationLive,
				Live:              true,
				PID:               4242,
				ExpectedStartTime: 100,
				ActualStartTime:   100,
				Command:           "/home/xel/.gormes/bin/gormes gateway",
			},
		},
	})
	defer restoreRuntimeStore()

	stdout, stderr, err := executeGatewayStatusCommand(t, "--json")
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	var got struct {
		Runtime struct {
			GatewayState string `json:"gateway_state"`
			PID          int    `json:"pid"`
			ActiveAgents int    `json:"active_agents"`
			Command      string `json:"command"`
		} `json:"runtime"`
		Validation gateway.RuntimeProcessValidation `json:"validation"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("gateway status --json returned invalid JSON: %v\n%s", err, stdout)
	}
	if got.Runtime.GatewayState != string(gateway.GatewayStateRunning) || got.Runtime.PID != 4242 || got.Runtime.ActiveAgents != 0 {
		t.Fatalf("json runtime = %+v, want running pid 4242 active_agents 0", got.Runtime)
	}
	if got.Runtime.Command != "/home/xel/.gormes/bin/gormes gateway" {
		t.Fatalf("json runtime command = %q", got.Runtime.Command)
	}
	if got.Validation.Status != gateway.RuntimeProcessValidationLive || !got.Validation.Live || got.Validation.PID != 4242 {
		t.Fatalf("json validation = %+v, want live pid 4242", got.Validation)
	}
	if strings.Contains(stdout, "Gateway status") || strings.Contains(stdout, "runtime_validation:") {
		t.Fatalf("json output mixed human text:\n%s", stdout)
	}
	assertGatewayStatusDidNotOpenRuntimeStores(t)
}

func TestGatewayStatusCommand_RendersStaleCodeRestartGuidance(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	restoreRuntimeStore := gatewayStatusRuntimeStoreForTest(t, fakeGatewayStatusRuntimeStore{
		snapshot: gateway.RuntimeStatusSnapshot{
			Status: gateway.RuntimeStatus{
				Kind:         "gormes-gateway",
				PID:          4242,
				GatewayState: gateway.GatewayStateRunning,
				BootGitSHA:   "1111111111111111111111111111111111111111",
				StaleCode: &gateway.RuntimeStaleCodeEvidence{
					Status:           gateway.RuntimeStaleCodeStale,
					BootGitSHA:       "1111111111111111111111111111111111111111",
					CurrentGitSHA:    "2222222222222222222222222222222222222222",
					Stale:            true,
					RestartSuggested: true,
					Evidence:         []string{"stale_code_head_changed", "stale_code_restart_gateway"},
					Message:          "gateway restart recommended to load current git HEAD",
				},
			},
			Validation: gateway.RuntimeProcessValidation{
				Status: gateway.RuntimeProcessValidationLive,
				Live:   true,
				PID:    4242,
			},
		},
	})
	defer restoreRuntimeStore()

	stdout, stderr, err := executeGatewayStatusCommand(t)
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{
		"stale_code: stale boot=111111111111 current=222222222222 restart_suggested=true evidence=stale_code_head_changed,stale_code_restart_gateway",
		"gateway restart recommended to load current git HEAD",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q\n%s", want, stdout)
		}
	}
}

func TestGatewayStatusCommand_JSONRendersStaleCodeEvidence(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	restoreRuntimeStore := gatewayStatusRuntimeStoreForTest(t, fakeGatewayStatusRuntimeStore{
		snapshot: gateway.RuntimeStatusSnapshot{
			Status: gateway.RuntimeStatus{
				Kind:         "gormes-gateway",
				PID:          4242,
				GatewayState: gateway.GatewayStateRunning,
				BootGitSHA:   "1111111111111111111111111111111111111111",
				StaleCode: &gateway.RuntimeStaleCodeEvidence{
					Status:           gateway.RuntimeStaleCodeStale,
					BootGitSHA:       "1111111111111111111111111111111111111111",
					CurrentGitSHA:    "2222222222222222222222222222222222222222",
					Stale:            true,
					RestartSuggested: true,
					Evidence:         []string{"stale_code_head_changed", "stale_code_restart_gateway"},
					Message:          "gateway restart recommended to load current git HEAD",
				},
			},
			Validation: gateway.RuntimeProcessValidation{
				Status: gateway.RuntimeProcessValidationLive,
				Live:   true,
				PID:    4242,
			},
		},
	})
	defer restoreRuntimeStore()

	stdout, stderr, err := executeGatewayStatusCommand(t, "--json")
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	var got struct {
		Runtime struct {
			StaleCode *gateway.RuntimeStaleCodeEvidence `json:"stale_code"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("gateway status --json returned invalid JSON: %v\n%s", err, stdout)
	}
	if got.Runtime.StaleCode == nil ||
		got.Runtime.StaleCode.Status != gateway.RuntimeStaleCodeStale ||
		!got.Runtime.StaleCode.RestartSuggested {
		t.Fatalf("json stale_code = %+v, want stale restart evidence", got.Runtime.StaleCode)
	}
	if got.Runtime.StaleCode.BootGitSHA != "1111111111111111111111111111111111111111" ||
		got.Runtime.StaleCode.CurrentGitSHA != "2222222222222222222222222222222222222222" {
		t.Fatalf("json stale_code SHAs = %+v", got.Runtime.StaleCode)
	}
}

func TestGatewayStatusCommand_JSONPairingPathHonorsGormesHome(t *testing.T) {
	root := t.TempDir()
	gormesHome := filepath.Join(root, "gormes-home")
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
	restoreRuntimeStore := gatewayStatusRuntimeStoreForTest(t, fakeGatewayStatusRuntimeStore{
		snapshot: gateway.RuntimeStatusSnapshot{Missing: true},
	})
	defer restoreRuntimeStore()

	stdout, stderr, err := executeGatewayStatusCommand(t, "--json")
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	var got struct {
		Pairing gateway.PairingStatus `json:"pairing"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("gateway status --json returned invalid JSON: %v\n%s", err, stdout)
	}
	if len(got.Pairing.Degraded) == 0 {
		t.Fatalf("pairing degraded evidence = none, want missing-path evidence")
	}
	wantPath := filepath.Join(gormesHome, "pairing.json")
	if got.Pairing.Degraded[0].Path != wantPath {
		t.Fatalf("pairing degraded path = %q, want %q", got.Pairing.Degraded[0].Path, wantPath)
	}
}

func setupGatewayStatusTestEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
}

func writeGatewayStatusConfig(t *testing.T, data []byte) {
	t.Helper()
	path := config.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func executeGatewayStatusCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	// Each newRootCommand() builds a fresh gateway tree via
	// newGatewayCommand(), so the JSON flag's default state is
	// natural — no explicit reset needed.
	var stdout, stderr bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(append([]string{"gateway", "status"}, args...))
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func assertGatewayStatusDidNotOpenRuntimeStores(t *testing.T) {
	t.Helper()
	for _, path := range []string{
		config.SessionDBPath(),
		config.MemoryDBPath(),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("gateway status opened runtime store %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat runtime store %s: %v", path, err)
		}
	}
}

type fakeGatewayStatusRuntimeStore struct {
	snapshot gateway.RuntimeStatusSnapshot
	err      error
}

func (s fakeGatewayStatusRuntimeStore) ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error) {
	return s.snapshot, s.err
}

func gatewayStatusRuntimeStoreForTest(t *testing.T, store fakeGatewayStatusRuntimeStore) func() {
	t.Helper()
	previous := newGatewayStatusRuntimeStore
	newGatewayStatusRuntimeStore = func(string) gatewayStatusRuntimeStore {
		return store
	}
	return func() {
		newGatewayStatusRuntimeStore = previous
	}
}
