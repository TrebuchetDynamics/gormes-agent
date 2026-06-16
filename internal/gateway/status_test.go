package gateway

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRuntimeStatusStoreAllowsNilContext(t *testing.T) {
	store := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("runtime status store panicked with nil context: %v", r)
		}
	}()

	if err := store.UpdateRuntimeStatus(nil, RuntimeStatusUpdate{GatewayState: GatewayStateRunning}); err != nil {
		t.Fatalf("UpdateRuntimeStatus nil context: %v", err)
	}
	if status, err := store.ReadRuntimeStatus(nil); err != nil || status.GatewayState != GatewayStateRunning {
		t.Fatalf("ReadRuntimeStatus nil context status=%+v err=%v, want running status", status, err)
	}
	if snapshot, err := store.ReadRuntimeStatusSnapshot(nil); err != nil || snapshot.Missing {
		t.Fatalf("ReadRuntimeStatusSnapshot nil context snapshot=%+v err=%v, want present snapshot", snapshot, err)
	}
	if snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(nil); err != nil || snapshot.Missing {
		t.Fatalf("ReadValidatedRuntimeStatusSnapshot nil context snapshot=%+v err=%v, want present snapshot", snapshot, err)
	}
}

func TestRuntimeStatusStore_MergesChannelLifecycleIntoReadModel(t *testing.T) {
	store := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))

	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		GatewayState: GatewayStateStarting,
	}); err != nil {
		t.Fatalf("write gateway starting: %v", err)
	}
	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      "telegram",
		PlatformState: PlatformStateStarting,
	}); err != nil {
		t.Fatalf("write telegram starting: %v", err)
	}
	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      "discord",
		PlatformState: PlatformStateFailed,
		ErrorMessage:  "discord: open session: denied",
	}); err != nil {
		t.Fatalf("write discord failed: %v", err)
	}
	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      "telegram",
		PlatformState: PlatformStateRunning,
		ErrorMessage:  "",
	}); err != nil {
		t.Fatalf("write telegram running: %v", err)
	}

	status, err := store.ReadRuntimeStatus(context.Background())
	if err != nil {
		t.Fatalf("read status: %v", err)
	}

	if status.Kind != "gormes-gateway" {
		t.Fatalf("Kind = %q, want gormes-gateway", status.Kind)
	}
	if status.GatewayState != GatewayStateStarting {
		t.Fatalf("GatewayState = %q, want %q", status.GatewayState, GatewayStateStarting)
	}
	if got := status.Platforms["telegram"].State; got != PlatformStateRunning {
		t.Fatalf("telegram state = %q, want %q", got, PlatformStateRunning)
	}
	if got := status.Platforms["telegram"].ErrorMessage; got != "" {
		t.Fatalf("telegram error = %q, want cleared empty error", got)
	}
	if got := status.Platforms["discord"].State; got != PlatformStateFailed {
		t.Fatalf("discord state = %q, want %q", got, PlatformStateFailed)
	}
	if got := status.Platforms["discord"].ErrorMessage; got != "discord: open session: denied" {
		t.Fatalf("discord error = %q, want startup failure", got)
	}
}

func TestRuntimeStatusStore_ClearsProxyURLOnStoppedUpdate(t *testing.T) {
	store := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))

	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		ProxyState: "running",
		ProxyURL:   "http://127.0.0.1:4321",
	}); err != nil {
		t.Fatalf("write running proxy status: %v", err)
	}
	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{ProxyState: "stopped"}); err != nil {
		t.Fatalf("write stopped proxy status: %v", err)
	}

	status, err := store.ReadRuntimeStatus(context.Background())
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status.Proxy.State != "stopped" {
		t.Fatalf("proxy state = %q, want stopped", status.Proxy.State)
	}
	if status.Proxy.URL != "" {
		t.Fatalf("proxy URL = %q, want cleared after stopped update", status.Proxy.URL)
	}
}

func TestRuntimeStatusStore_ClearsKanbanLastErrorOnStoppedUpdate(t *testing.T) {
	store := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))

	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		KanbanDispatcher: &KanbanDispatcherStatus{
			State:       KanbanDispatcherStateDegraded,
			LastError:   "worker_spawn_failed: missing profile",
			SpawnFailed: 1,
		},
	}); err != nil {
		t.Fatalf("write degraded kanban status: %v", err)
	}
	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		KanbanDispatcher: &KanbanDispatcherStatus{State: KanbanDispatcherStateStopped},
	}); err != nil {
		t.Fatalf("write stopped kanban status: %v", err)
	}

	status, err := store.ReadRuntimeStatus(context.Background())
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status.KanbanDispatcher.State != KanbanDispatcherStateStopped {
		t.Fatalf("kanban state = %q, want stopped", status.KanbanDispatcher.State)
	}
	if status.KanbanDispatcher.LastError != "" {
		t.Fatalf("kanban last error = %q, want cleared after stopped update", status.KanbanDispatcher.LastError)
	}
}

func TestRuntimeStatusStore_ClearsStaleExitReasonOnFreshStart(t *testing.T) {
	store := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))

	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		GatewayState: GatewayStateStartupFailed,
		ExitReason:   "telegram polling conflict",
	}); err != nil {
		t.Fatalf("write startup failure: %v", err)
	}
	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		GatewayState: GatewayStateStarting,
	}); err != nil {
		t.Fatalf("write fresh start: %v", err)
	}

	status, err := store.ReadRuntimeStatus(context.Background())
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status.GatewayState != GatewayStateStarting {
		t.Fatalf("GatewayState = %q, want %q", status.GatewayState, GatewayStateStarting)
	}
	if status.ExitReason != "" {
		t.Fatalf("ExitReason = %q, want cleared stale failure", status.ExitReason)
	}
}

func TestRuntimeStatusStore_RedactsArgvSecretsInStatusFiles(t *testing.T) {
	root := t.TempDir()
	statusPath := filepath.Join(root, "gateway_state.json")
	pidPath := filepath.Join(root, "gateway.pid")
	store := NewRuntimeStatusStore(statusPath)
	store.pidPath = pidPath
	store.pid = func() int { return 4242 }
	store.startTime = func(int) (int64, bool) { return 87654321, true }
	store.argv = func() []string { return []string{"gormes", "gateway", "--api-key=plain-secret-token"} }

	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{GatewayState: GatewayStateStarting}); err != nil {
		t.Fatalf("write gateway starting: %v", err)
	}

	for _, path := range []string{statusPath, pidPath} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, forbidden := range []string{"plain-secret-token", "api-key"} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("runtime status file %s leaked argv secret %q:\n%s", path, forbidden, raw)
			}
		}
		if !strings.Contains(string(raw), "[redacted]") {
			t.Fatalf("runtime status file %s missing redacted argv evidence:\n%s", path, raw)
		}
	}
}

func TestRuntimeStatusStore_RedactsSplitAuthorizationArgvInStatusFiles(t *testing.T) {
	root := t.TempDir()
	statusPath := filepath.Join(root, "gateway_state.json")
	pidPath := filepath.Join(root, "gateway.pid")
	store := NewRuntimeStatusStore(statusPath)
	store.pidPath = pidPath
	store.pid = func() int { return 4242 }
	store.startTime = func(int) (int64, bool) { return 87654321, true }
	store.argv = func() []string { return []string{"gormes", "gateway", "--authorization", "Bearer plain-secret-token"} }

	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{GatewayState: GatewayStateStarting}); err != nil {
		t.Fatalf("write gateway starting: %v", err)
	}

	for _, path := range []string{statusPath, pidPath} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, forbidden := range []string{"plain-secret-token", "authorization", "Bearer"} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("runtime status file %s leaked split authorization argv %q:\n%s", path, forbidden, raw)
			}
		}
		if !strings.Contains(string(raw), "[redacted]") {
			t.Fatalf("runtime status file %s missing redacted argv evidence:\n%s", path, raw)
		}
	}
}

func TestRuntimeStatusStore_RedactsAuthorizationArgvInStatusFiles(t *testing.T) {
	root := t.TempDir()
	statusPath := filepath.Join(root, "gateway_state.json")
	pidPath := filepath.Join(root, "gateway.pid")
	store := NewRuntimeStatusStore(statusPath)
	store.pidPath = pidPath
	store.pid = func() int { return 4242 }
	store.startTime = func(int) (int64, bool) { return 87654321, true }
	store.argv = func() []string {
		return []string{"gormes", "gateway", "--header", "Authorization: Bearer plain-secret-token"}
	}

	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{GatewayState: GatewayStateStarting}); err != nil {
		t.Fatalf("write gateway starting: %v", err)
	}

	for _, path := range []string{statusPath, pidPath} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, forbidden := range []string{"plain-secret-token", "Authorization", "authorization", "Bearer"} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("runtime status file %s leaked authorization argv %q:\n%s", path, forbidden, raw)
			}
		}
		if !strings.Contains(string(raw), "[redacted]") {
			t.Fatalf("runtime status file %s missing redacted argv evidence:\n%s", path, raw)
		}
	}
}

func TestRuntimeStatusStore_PrimaryStatusWriteFailurePreservesPIDRecord(t *testing.T) {
	root := t.TempDir()
	statusRoot := filepath.Join(root, "status")
	pidRoot := filepath.Join(root, "pid")
	if err := os.Mkdir(statusRoot, 0o755); err != nil {
		t.Fatalf("mkdir status root: %v", err)
	}
	if err := os.Mkdir(pidRoot, 0o755); err != nil {
		t.Fatalf("mkdir pid root: %v", err)
	}
	statusPath := filepath.Join(statusRoot, "gateway_state.json")
	pidPath := filepath.Join(pidRoot, "gateway.pid")
	store := NewRuntimeStatusStore(statusPath)
	store.pidPath = pidPath
	store.pid = func() int { return 4242 }
	store.startTime = func(int) (int64, bool) { return 87654321, true }
	store.argv = func() []string { return []string{"gormes", "gateway"} }

	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{GatewayState: GatewayStateStarting}); err != nil {
		t.Fatalf("write initial status: %v", err)
	}
	beforePID := readRuntimeStatusFixture(t, pidPath)

	if err := os.Chmod(statusRoot, 0o500); err != nil {
		t.Fatalf("chmod status root read-only: %v", err)
	}
	defer func() { _ = os.Chmod(statusRoot, 0o700) }()
	err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{Platform: "telegram", PlatformState: PlatformStateRunning})
	if err == nil {
		t.Fatal("UpdateRuntimeStatus with unwritable primary status succeeded, want error")
	}
	afterPID := readRuntimeStatusFixture(t, pidPath)
	if !reflect.DeepEqual(afterPID, beforePID) {
		t.Fatalf("pid record changed after primary status write failure\nbefore: %+v\nafter:  %+v", beforePID, afterPID)
	}
}

func TestRuntimeStatusStore_PIDRecordWriteFailurePreservesPrimaryStatus(t *testing.T) {
	root := t.TempDir()
	statusPath := filepath.Join(root, "gateway_state.json")
	pidPath := filepath.Join(root, "gateway.pid")
	store := NewRuntimeStatusStore(statusPath)
	store.pidPath = pidPath
	store.pid = func() int { return 4242 }
	store.startTime = func(int) (int64, bool) { return 87654321, true }
	store.argv = func() []string { return []string{"gormes", "gateway"} }

	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{GatewayState: GatewayStateStarting}); err != nil {
		t.Fatalf("write initial status: %v", err)
	}
	before := readRuntimeStatusFixture(t, statusPath)

	badPIDPath := filepath.Join(root, "pid-record-dir")
	if err := os.Mkdir(badPIDPath, 0o755); err != nil {
		t.Fatalf("mkdir bad pid path: %v", err)
	}
	store.pidPath = badPIDPath
	err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{Platform: "telegram", PlatformState: PlatformStateRunning})
	if err == nil {
		t.Fatal("UpdateRuntimeStatus with unwritable pid record succeeded, want error")
	}
	after := readRuntimeStatusFixture(t, statusPath)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("primary runtime status changed after pid record write failure\nbefore: %+v\nafter:  %+v", before, after)
	}
}

func TestRuntimeStatusStore_WritesPIDStartTimeGenerationAndCommandIdentity(t *testing.T) {
	root := t.TempDir()
	statusPath := filepath.Join(root, "gateway_state.json")
	pidPath := filepath.Join(root, "gateway.pid")
	store := NewRuntimeStatusStore(statusPath)
	store.pidPath = pidPath
	store.pid = func() int { return 4242 }
	store.startTime = func(pid int) (int64, bool) {
		if pid != 4242 {
			t.Fatalf("startTime pid = %d, want 4242", pid)
		}
		return 87654321, true
	}
	store.argv = func() []string { return []string{"gormes", "gateway"} }

	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		GatewayState: GatewayStateStarting,
	}); err != nil {
		t.Fatalf("write gateway starting: %v", err)
	}
	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      "telegram",
		PlatformState: PlatformStateRunning,
	}); err != nil {
		t.Fatalf("write telegram running: %v", err)
	}

	status := readRuntimeStatusFixture(t, statusPath)
	if status.PID != 4242 {
		t.Fatalf("status PID = %d, want 4242", status.PID)
	}
	if status.StartTime != 87654321 {
		t.Fatalf("status StartTime = %d, want 87654321", status.StartTime)
	}
	if status.Generation != 2 {
		t.Fatalf("status Generation = %d, want 2", status.Generation)
	}
	if status.Command != "gormes gateway" {
		t.Fatalf("status Command = %q, want command identity", status.Command)
	}
	if !reflect.DeepEqual(status.Argv, []string{"gormes", "gateway"}) {
		t.Fatalf("status Argv = %#v, want gormes gateway argv", status.Argv)
	}
	if status.ProcessValidation.Status != RuntimeProcessValidationLive || !status.ProcessValidation.Live {
		t.Fatalf("status ProcessValidation = %+v, want persisted live self-validation", status.ProcessValidation)
	}

	pidRecord := readRuntimeStatusFixture(t, pidPath)
	if pidRecord.PID != status.PID ||
		pidRecord.StartTime != status.StartTime ||
		pidRecord.Generation != status.Generation ||
		pidRecord.Command != status.Command ||
		!reflect.DeepEqual(pidRecord.Argv, status.Argv) ||
		pidRecord.ProcessValidation.Status != RuntimeProcessValidationLive ||
		!pidRecord.ProcessValidation.Live {
		t.Fatalf("pid record = %+v, want same live identity as status %+v", pidRecord, status)
	}
}

func readRuntimeStatusFixture(t *testing.T, path string) RuntimeStatus {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var status RuntimeStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return status
}
