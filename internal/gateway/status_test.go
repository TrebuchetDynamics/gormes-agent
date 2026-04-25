package gateway

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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

	pidRecord := readRuntimeStatusFixture(t, pidPath)
	if pidRecord.PID != status.PID ||
		pidRecord.StartTime != status.StartTime ||
		pidRecord.Generation != status.Generation ||
		pidRecord.Command != status.Command ||
		!reflect.DeepEqual(pidRecord.Argv, status.Argv) {
		t.Fatalf("pid record = %+v, want same identity as status %+v", pidRecord, status)
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
