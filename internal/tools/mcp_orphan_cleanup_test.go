package tools

import (
	"context"
	"errors"
	"io"
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
