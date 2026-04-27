package tools

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"
)

func TestMCPOrphanCleanup_MarksAlivePIDOrphanOnSessionExit(t *testing.T) {
	const pid = 4242
	tracker := NewMCPStdioProcessTracker(MCPStdioProcessTrackerOptions{
		Alive: func(got int) bool { return got == pid },
		Kill: func(int) error {
			t.Fatal("session exit should only mark an orphan, not kill it")
			return nil
		},
	})

	server := startFakeStdioServer(t, func(req fakeStdioRequest) []byte { return nil })
	defer server.Close()

	client, err := NewStdioClient(MCPServerDefinition{
		Name:      "fake",
		Enabled:   true,
		Transport: MCPTransportStdio,
		Command:   "fake-mcp",
	}, StdioClientOpts{
		Conn:           server.client,
		ProcessPID:     pid,
		ProcessTracker: tracker,
	})
	if err != nil {
		t.Fatalf("NewStdioClient: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.ListTools(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ListTools err = %v; want context.Canceled", err)
	}
	if err := client.Close(); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Close: %v", err)
	}

	snap := tracker.Snapshot()
	if _, ok := snap.Active[pid]; ok {
		t.Fatalf("pid %d still active after session exit: %#v", pid, snap.Active)
	}
	if got := snap.Orphaned[pid]; got != "fake" {
		t.Fatalf("orphaned[%d] = %q; want fake", pid, got)
	}
}

func TestMCPOrphanCleanup_ReapsOnlyOrphansAfterCronTick(t *testing.T) {
	const (
		activePID = 101
		orphanPID = 202
	)
	var killed []int
	tracker := NewMCPStdioProcessTracker(MCPStdioProcessTrackerOptions{
		Alive: func(pid int) bool { return pid == activePID || pid == orphanPID },
		Kill: func(pid int) error {
			killed = append(killed, pid)
			return nil
		},
	})
	tracker.TrackActivePID("live-session", activePID)
	tracker.TrackActivePID("cancelled-session", orphanPID)
	tracker.MarkSessionExit("cancelled-session", orphanPID)

	events := tracker.ReapOrphans()

	if !slices.Equal(killed, []int{orphanPID}) {
		t.Fatalf("killed = %#v; want only orphan pid %d", killed, orphanPID)
	}
	snap := tracker.Snapshot()
	if got := snap.Active[activePID]; got != "live-session" {
		t.Fatalf("active[%d] = %q; want live-session", activePID, got)
	}
	if _, ok := snap.Orphaned[orphanPID]; ok {
		t.Fatalf("orphan pid %d still tracked after reap: %#v", orphanPID, snap.Orphaned)
	}
	assertMCPProcessEvent(t, events, MCPOrphanReaped, orphanPID)
	assertMCPProcessEvent(t, events, MCPActivePIDPreserved, activePID)
}

func TestMCPOrphanCleanup_ShutdownIncludesActive(t *testing.T) {
	const (
		activePID = 303
		orphanPID = 404
	)
	var killed []int
	tracker := NewMCPStdioProcessTracker(MCPStdioProcessTrackerOptions{
		Alive: func(pid int) bool { return pid == activePID || pid == orphanPID },
		Kill: func(pid int) error {
			killed = append(killed, pid)
			return nil
		},
	})
	tracker.TrackActivePID("live-session", activePID)
	tracker.TrackActivePID("cancelled-session", orphanPID)
	tracker.MarkSessionExit("cancelled-session", orphanPID)

	events := tracker.Shutdown()

	slices.Sort(killed)
	if !slices.Equal(killed, []int{activePID, orphanPID}) {
		t.Fatalf("killed = %#v; want active and orphan pids", killed)
	}
	snap := tracker.Snapshot()
	if len(snap.Active) != 0 {
		t.Fatalf("active after shutdown = %#v; want empty", snap.Active)
	}
	if len(snap.Orphaned) != 0 {
		t.Fatalf("orphaned after shutdown = %#v; want empty", snap.Orphaned)
	}
	assertMCPProcessEvent(t, events, MCPOrphanReaped, activePID)
	assertMCPProcessEvent(t, events, MCPOrphanReaped, orphanPID)
}

func assertMCPProcessEvent(t *testing.T, events []MCPStdioCleanupEvent, status MCPStdioCleanupStatus, pid int) {
	t.Helper()
	for _, event := range events {
		if event.Status == status && event.PID == pid {
			return
		}
	}
	t.Fatalf("missing event status=%q pid=%d in %#v", status, pid, events)
}
