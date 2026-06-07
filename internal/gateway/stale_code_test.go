package gateway

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/stalecodetest"
)

const (
	staleCodeSHA1 = "1111111111111111111111111111111111111111"
	staleCodeSHA2 = "2222222222222222222222222222222222222222"
)

func TestRuntimeStatusStorePersistsBootGitSHAAndAnnotatesStaleCode(t *testing.T) {
	root := t.TempDir()
	stalecodetest.WriteNormalGitHEAD(t, root, "development", staleCodeSHA2)
	statusPath := filepath.Join(t.TempDir(), "gateway_state.json")
	store := NewRuntimeStatusStore(statusPath)
	store.bootGitSHA = staleCodeSHA1
	store.staleCodeChecker = NewStaleCodeChecker(root)
	store.processes = fakeRuntimeProcessTable{
		4242: {startTime: 123, command: "gormes gateway"},
	}
	store.pid = func() int { return 4242 }
	store.startTime = func(int) (int64, bool) { return 123, true }
	store.argv = func() []string { return []string{"gormes", "gateway"} }

	if err := store.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{GatewayState: GatewayStateRunning}); err != nil {
		t.Fatalf("UpdateRuntimeStatus: %v", err)
	}
	snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(context.Background())
	if err != nil {
		t.Fatalf("ReadValidatedRuntimeStatusSnapshot: %v", err)
	}
	if snapshot.Status.BootGitSHA != staleCodeSHA1 {
		t.Fatalf("BootGitSHA = %q, want %q", snapshot.Status.BootGitSHA, staleCodeSHA1)
	}
	if snapshot.Status.StaleCode == nil ||
		snapshot.Status.StaleCode.Status != RuntimeStaleCodeStale ||
		!snapshot.Status.StaleCode.RestartSuggested {
		t.Fatalf("StaleCode = %+v, want stale restart evidence", snapshot.Status.StaleCode)
	}
}

func TestRuntimeStatusStoreUsesPersistedSourceRootForValidatedReads(t *testing.T) {
	liveRoot := t.TempDir()
	stalecodetest.WriteNormalGitHEAD(t, liveRoot, "development", staleCodeSHA1)
	otherRoot := t.TempDir()
	stalecodetest.WriteNormalGitHEAD(t, otherRoot, "development", staleCodeSHA2)
	statusPath := filepath.Join(t.TempDir(), "gateway_state.json")

	writer := NewRuntimeStatusStore(statusPath)
	writer.bootGitSHA = staleCodeSHA1
	writer.staleCodeChecker = NewStaleCodeChecker(liveRoot)
	writer.processes = fakeRuntimeProcessTable{
		4242: {startTime: 123, command: "gormes gateway"},
	}
	writer.pid = func() int { return 4242 }
	writer.startTime = func(int) (int64, bool) { return 123, true }
	writer.argv = func() []string { return []string{"gormes", "gateway"} }

	if err := writer.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{GatewayState: GatewayStateRunning}); err != nil {
		t.Fatalf("UpdateRuntimeStatus: %v", err)
	}

	reader := NewRuntimeStatusStore(statusPath)
	reader.staleCodeChecker = NewStaleCodeChecker(otherRoot)
	reader.processes = fakeRuntimeProcessTable{
		4242: {startTime: 123, command: "gormes gateway"},
	}
	snapshot, err := reader.ReadValidatedRuntimeStatusSnapshot(context.Background())
	if err != nil {
		t.Fatalf("ReadValidatedRuntimeStatusSnapshot: %v", err)
	}
	if snapshot.Status.SourceRoot != liveRoot {
		t.Fatalf("SourceRoot = %q, want %q", snapshot.Status.SourceRoot, liveRoot)
	}
	if snapshot.Status.StaleCode == nil || snapshot.Status.StaleCode.Status != RuntimeStaleCodeFresh {
		t.Fatalf("StaleCode = %+v, want fresh from persisted source root", snapshot.Status.StaleCode)
	}
}
