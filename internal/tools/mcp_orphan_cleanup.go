package tools

import (
	"errors"
	"sort"
	"sync"
	"syscall"
)

// MCPStdioCleanupStatus is operator-visible evidence from MCP stdio cleanup.
type MCPStdioCleanupStatus string

const (
	MCPOrphanReaped       MCPStdioCleanupStatus = "mcp_orphan_reaped"
	MCPOrphanReapFailed   MCPStdioCleanupStatus = "mcp_orphan_reap_failed"
	MCPActivePIDPreserved MCPStdioCleanupStatus = "mcp_active_pid_preserved"
)

// MCPStdioCleanupEvent records one PID decision from a cleanup sweep.
type MCPStdioCleanupEvent struct {
	Status     MCPStdioCleanupStatus
	PID        int
	ServerName string
	Error      string
}

// MCPStdioProcessTrackerOptions injects process operations for tests.
type MCPStdioProcessTrackerOptions struct {
	Alive func(pid int) bool
	Kill  func(pid int) error
}

// MCPStdioProcessSnapshot is a copied read model of tracked PIDs.
type MCPStdioProcessSnapshot struct {
	Active   map[int]string
	Orphaned map[int]string
}

// MCPStdioProcessTracker tracks active stdio MCP server PIDs and the subset
// that survived session exit.
type MCPStdioProcessTracker struct {
	mu       sync.Mutex
	active   map[int]string
	orphaned map[int]string
	alive    func(pid int) bool
	kill     func(pid int) error
}

// DefaultMCPStdioProcessTracker is used by stdio clients and cron cleanup in
// normal runtime wiring.
var DefaultMCPStdioProcessTracker = NewMCPStdioProcessTracker(MCPStdioProcessTrackerOptions{})

func NewMCPStdioProcessTracker(opts MCPStdioProcessTrackerOptions) *MCPStdioProcessTracker {
	alive := opts.Alive
	if alive == nil {
		alive = defaultMCPStdioPIDAlive
	}
	kill := opts.Kill
	if kill == nil {
		kill = defaultMCPStdioKillPID
	}
	return &MCPStdioProcessTracker{
		active:   make(map[int]string),
		orphaned: make(map[int]string),
		alive:    alive,
		kill:     kill,
	}
}

// TrackActivePID marks a stdio server PID as in-flight.
func (t *MCPStdioProcessTracker) TrackActivePID(serverName string, pid int) {
	if t == nil || pid <= 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.active[pid] = serverName
	delete(t.orphaned, pid)
}

// MarkSessionExit removes a PID from active tracking and records it as an
// orphan only if it is still alive after the stdio session exits.
func (t *MCPStdioProcessTracker) MarkSessionExit(serverName string, pid int) {
	if t == nil || pid <= 0 {
		return
	}
	t.mu.Lock()
	if trackedName, ok := t.active[pid]; ok && serverName == "" {
		serverName = trackedName
	}
	delete(t.active, pid)
	t.mu.Unlock()

	if !t.alive(pid) {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if _, active := t.active[pid]; active {
		return
	}
	t.orphaned[pid] = serverName
}

// ReapOrphans kills only PIDs already proven orphaned. Active sessions are
// reported as preserved evidence and left untouched.
func (t *MCPStdioProcessTracker) ReapOrphans() []MCPStdioCleanupEvent {
	return t.reap(false)
}

// Shutdown kills both active and orphaned stdio PIDs.
func (t *MCPStdioProcessTracker) Shutdown() []MCPStdioCleanupEvent {
	return t.reap(true)
}

func (t *MCPStdioProcessTracker) Snapshot() MCPStdioProcessSnapshot {
	if t == nil {
		return MCPStdioProcessSnapshot{Active: map[int]string{}, Orphaned: map[int]string{}}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return MCPStdioProcessSnapshot{
		Active:   copyMCPStdioPIDMap(t.active),
		Orphaned: copyMCPStdioPIDMap(t.orphaned),
	}
}

func (t *MCPStdioProcessTracker) reap(includeActive bool) []MCPStdioCleanupEvent {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	orphaned := copyMCPStdioPIDMap(t.orphaned)
	active := copyMCPStdioPIDMap(t.active)
	for pid := range orphaned {
		delete(t.orphaned, pid)
	}
	if includeActive {
		for pid := range active {
			delete(t.active, pid)
		}
	}
	t.mu.Unlock()

	events := make([]MCPStdioCleanupEvent, 0, len(orphaned)+len(active))
	for _, pid := range sortedMCPStdioPIDs(orphaned) {
		events = append(events, t.killPID(pid, orphaned[pid]))
	}
	for _, pid := range sortedMCPStdioPIDs(active) {
		if includeActive {
			events = append(events, t.killPID(pid, active[pid]))
			continue
		}
		events = append(events, MCPStdioCleanupEvent{
			Status:     MCPActivePIDPreserved,
			PID:        pid,
			ServerName: active[pid],
		})
	}
	return events
}

func (t *MCPStdioProcessTracker) killPID(pid int, serverName string) MCPStdioCleanupEvent {
	if !t.alive(pid) {
		return MCPStdioCleanupEvent{Status: MCPOrphanReaped, PID: pid, ServerName: serverName}
	}
	if err := t.kill(pid); err != nil {
		t.mu.Lock()
		t.orphaned[pid] = serverName
		t.mu.Unlock()
		return MCPStdioCleanupEvent{
			Status:     MCPOrphanReapFailed,
			PID:        pid,
			ServerName: serverName,
			Error:      err.Error(),
		}
	}
	return MCPStdioCleanupEvent{Status: MCPOrphanReaped, PID: pid, ServerName: serverName}
}

// ReapMCPStdioOrphans runs the normal post-cron cleanup sweep.
func ReapMCPStdioOrphans() []MCPStdioCleanupEvent {
	return DefaultMCPStdioProcessTracker.ReapOrphans()
}

// ShutdownMCPStdioProcesses runs the final shutdown sweep, including active
// PIDs because no stdio sessions should remain in flight.
func ShutdownMCPStdioProcesses() []MCPStdioCleanupEvent {
	return DefaultMCPStdioProcessTracker.Shutdown()
}

func copyMCPStdioPIDMap(in map[int]string) map[int]string {
	out := make(map[int]string, len(in))
	for pid, name := range in {
		out[pid] = name
	}
	return out
}

func sortedMCPStdioPIDs(in map[int]string) []int {
	pids := make([]int, 0, len(in))
	for pid := range in {
		pids = append(pids, pid)
	}
	sort.Ints(pids)
	return pids
}

func defaultMCPStdioPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

func defaultMCPStdioKillPID(pid int) error {
	if pid <= 0 {
		return nil
	}
	return syscall.Kill(pid, syscall.SIGTERM)
}
