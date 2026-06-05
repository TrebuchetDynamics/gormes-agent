package process

import (
	"slices"
	"testing"
)

func TestOrphanCleanupReapsOnlyOrphansAfterCronTick(t *testing.T) {
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
	assertProcessEvent(t, events, MCPOrphanReaped, orphanPID)
	assertProcessEvent(t, events, MCPActivePIDPreserved, activePID)
}

func TestOrphanCleanupShutdownIncludesActive(t *testing.T) {
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
	assertProcessEvent(t, events, MCPOrphanReaped, activePID)
	assertProcessEvent(t, events, MCPOrphanReaped, orphanPID)
}

func assertProcessEvent(t *testing.T, events []MCPStdioCleanupEvent, status MCPStdioCleanupStatus, pid int) {
	t.Helper()
	for _, event := range events {
		if event.Status == status && event.PID == pid {
			return
		}
	}
	t.Fatalf("missing event status=%q pid=%d in %#v", status, pid, events)
}
