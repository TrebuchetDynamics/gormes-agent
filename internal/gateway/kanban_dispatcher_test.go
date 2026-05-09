package gateway

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
)

type fakeKanbanGatewayDispatcher struct {
	mu      sync.Mutex
	results []kanban.DispatchResult
	errors  []error
	calls   int
}

func (f *fakeKanbanGatewayDispatcher) RunOnce(context.Context) (kanban.DispatchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if len(f.results) == 0 {
		if len(f.errors) == 0 {
			return kanban.DispatchResult{}, nil
		}
		err := f.errors[0]
		f.errors = f.errors[1:]
		return kanban.DispatchResult{}, err
	}
	result := f.results[0]
	f.results = f.results[1:]
	if len(f.errors) == 0 {
		return result, nil
	}
	err := f.errors[0]
	f.errors = f.errors[1:]
	return result, err
}

func (f *fakeKanbanGatewayDispatcher) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func waitForKanbanManagerStop(t *testing.T, label string, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s Run returned error: %v", label, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("%s Manager.Run did not stop after context cancellation", label)
	}
}

func TestManagerKanbanDispatcherLifecycleRunsTicksNudgesAndStops(t *testing.T) {
	ctx := context.Background()
	ticks := make(chan time.Time, 2)
	nudges := make(chan struct{}, 2)
	dispatcher := &fakeKanbanGatewayDispatcher{
		results: []kanban.DispatchResult{
			{
				Spawned:        []kanban.SpawnRecord{{TaskID: "task-1"}, {TaskID: "task-2"}},
				SpawnFailedIDs: []string{"task-3"},
			},
			{
				AutoBlockedTaskIDs: []string{"task-3"},
			},
		},
	}
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	m := NewManagerWithSubmitter(ManagerConfig{
		RuntimeStatus: statusStore,
		KanbanDispatcher: KanbanDispatcherConfig{
			Runner: dispatcher,
			Tick:   ticks,
			Nudge:  nudges,
		},
	}, &fakeKernel{}, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- m.Run(runCtx) }()
	<-ch.started

	waitFor(t, 200*time.Millisecond, func() bool {
		status, err := statusStore.ReadRuntimeStatus(ctx)
		return err == nil && status.KanbanDispatcher.State == KanbanDispatcherStateRunning
	})

	ticks <- time.Date(2026, 5, 4, 17, 0, 0, 0, time.UTC)
	waitFor(t, 200*time.Millisecond, func() bool {
		return dispatcher.callCount() == 1
	})
	nudges <- struct{}{}
	waitFor(t, 200*time.Millisecond, func() bool {
		return dispatcher.callCount() == 2
	})

	status, err := statusStore.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if got := status.KanbanDispatcher.Spawned; got != 2 {
		t.Fatalf("kanban spawned = %d, want 2", got)
	}
	if got := status.KanbanDispatcher.SpawnFailed; got != 1 {
		t.Fatalf("kanban spawn_failed = %d, want 1", got)
	}
	if got := status.KanbanDispatcher.AutoBlocked; got != 1 {
		t.Fatalf("kanban auto_blocked = %d, want 1", got)
	}

	cancel()
	waitForKanbanManagerStop(t, "lifecycle", done)
	status, err = statusStore.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus after stop: %v", err)
	}
	if status.KanbanDispatcher.State != KanbanDispatcherStateStopped {
		t.Fatalf("kanban dispatcher state = %q, want %q", status.KanbanDispatcher.State, KanbanDispatcherStateStopped)
	}
}

func TestManagerKanbanDispatcherUsesProductionProcessSpawnerRunner(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 6, 20, 0, 0, 0, time.UTC)
	store, err := kanban.Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open kanban store: %v", err)
	}
	defer store.Close()
	task, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: "Spawn from gateway", Assignee: "coder"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	starter := &recordingGatewayProcessStarter{result: kanban.ProcessStartResult{PID: 5150, StartedAt: now}}
	runner := kanban.Runner{
		Dispatcher: kanban.Dispatcher{
			Store: store,
			Spawner: kanban.ProcessSpawner{
				Binary:  "gormes",
				LogRoot: filepath.Join(t.TempDir(), "logs"),
				Starter: starter,
			},
		},
		Options: kanban.DispatchOptions{MaxSpawn: 1},
	}
	var _ KanbanDispatcherRunner = runner

	ticks := make(chan time.Time, 1)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	m := NewManagerWithSubmitter(ManagerConfig{
		RuntimeStatus: statusStore,
		KanbanDispatcher: KanbanDispatcherConfig{
			Runner: runner,
			Tick:   ticks,
		},
	}, &fakeKernel{}, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- m.Run(runCtx) }()
	<-ch.started

	ticks <- now
	waitFor(t, 200*time.Millisecond, func() bool {
		return starter.callCount() == 1
	})
	req := starter.requests[0]
	if req.Env["GORMES_KANBAN_TASK"] != task.ID {
		t.Fatalf("GORMES_KANBAN_TASK = %q, want %q", req.Env["GORMES_KANBAN_TASK"], task.ID)
	}
	if req.Binary != "gormes" {
		t.Fatalf("Binary = %q, want gormes", req.Binary)
	}
	var status RuntimeStatus
	waitFor(t, 200*time.Millisecond, func() bool {
		status, err = statusStore.ReadRuntimeStatus(ctx)
		return err == nil && status.KanbanDispatcher.Spawned == 1
	})
	if status.KanbanDispatcher.Spawned != 1 {
		t.Fatalf("kanban spawned = %d, want 1", status.KanbanDispatcher.Spawned)
	}

	cancel()
	waitForKanbanManagerStop(t, "production runner", done)
}

type recordingGatewayProcessStarter struct {
	mu       sync.Mutex
	requests []kanban.ProcessStartRequest
	result   kanban.ProcessStartResult
}

func (s *recordingGatewayProcessStarter) StartKanbanProcess(_ context.Context, req kanban.ProcessStartRequest) (kanban.ProcessStartResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req)
	return s.result, nil
}

func (s *recordingGatewayProcessStarter) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

func TestManagerKanbanDispatcherRecordsTriggerTimestamp(t *testing.T) {
	ctx := context.Background()
	ticks := make(chan time.Time, 1)
	triggeredAt := time.Date(2026, 5, 4, 18, 15, 0, 0, time.UTC)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	m := NewManagerWithSubmitter(ManagerConfig{
		RuntimeStatus: statusStore,
		Now:           func() time.Time { return triggeredAt.Add(time.Hour) },
		KanbanDispatcher: KanbanDispatcherConfig{
			Runner: &fakeKanbanGatewayDispatcher{},
			Tick:   ticks,
		},
	}, &fakeKernel{}, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- m.Run(runCtx) }()
	<-ch.started

	ticks <- triggeredAt
	waitFor(t, 200*time.Millisecond, func() bool {
		status, err := statusStore.ReadRuntimeStatus(ctx)
		return err == nil && status.KanbanDispatcher.LastTickAt != ""
	})

	status, err := statusStore.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if got, want := status.KanbanDispatcher.LastTickAt, triggeredAt.Format(time.RFC3339Nano); got != want {
		t.Fatalf("LastTickAt = %q, want tick timestamp %q", got, want)
	}
	cancel()
	waitForKanbanManagerStop(t, "timestamp", done)
}

func TestManagerKanbanDispatcherErrorRecordsStatusWithoutStoppingGateway(t *testing.T) {
	ctx := context.Background()
	ticks := make(chan time.Time, 1)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	dispatcher := &fakeKanbanGatewayDispatcher{
		errors: []error{errors.New("worker_spawn_failed: missing profile")},
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: statusStore,
		KanbanDispatcher: KanbanDispatcherConfig{
			Runner: dispatcher,
			Tick:   ticks,
		},
	}, &fakeKernel{}, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- m.Run(runCtx) }()
	<-ch.started

	ticks <- time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC)
	waitFor(t, 200*time.Millisecond, func() bool {
		status, err := statusStore.ReadRuntimeStatus(ctx)
		return err == nil && status.KanbanDispatcher.LastError != ""
	})
	status, err := statusStore.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if status.KanbanDispatcher.State != KanbanDispatcherStateDegraded {
		t.Fatalf("kanban dispatcher state = %q, want %q", status.KanbanDispatcher.State, KanbanDispatcherStateDegraded)
	}
	if got := status.KanbanDispatcher.LastError; got != "worker_spawn_failed: missing profile" {
		t.Fatalf("kanban dispatcher LastError = %q", got)
	}

	ch.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m", Kind: EventStart})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(ch.sentSnapshot()) == 1
	})

	cancel()
	waitForKanbanManagerStop(t, "degraded dispatcher", done)
}

func TestManagerKanbanDispatcherCanRestartAfterGatewayStop(t *testing.T) {
	ctx := context.Background()
	ticks := make(chan time.Time, 2)
	dispatcher := &fakeKanbanGatewayDispatcher{}
	m := NewManagerWithSubmitter(ManagerConfig{
		KanbanDispatcher: KanbanDispatcherConfig{
			Runner: dispatcher,
			Tick:   ticks,
		},
	}, &fakeKernel{}, slog.Default())

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- m.Run(runCtx) }()
	ticks <- time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)
	waitFor(t, 200*time.Millisecond, func() bool {
		return dispatcher.callCount() == 1
	})
	cancel()
	waitForKanbanManagerStop(t, "first restart run", done)

	runCtx, cancel = context.WithCancel(ctx)
	defer cancel()
	done = make(chan error, 1)
	go func() { done <- m.Run(runCtx) }()
	ticks <- time.Date(2026, 5, 4, 20, 1, 0, 0, time.UTC)
	waitFor(t, 200*time.Millisecond, func() bool {
		return dispatcher.callCount() == 2
	})
	cancel()
	waitForKanbanManagerStop(t, "second restart run", done)
}

func TestManagerKanbanDispatcherClosedTickChannelStopsDispatcherLoop(t *testing.T) {
	ctx := context.Background()
	ticks := make(chan time.Time)
	close(ticks)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	dispatcher := &fakeKanbanGatewayDispatcher{}
	m := NewManagerWithSubmitter(ManagerConfig{
		RuntimeStatus: statusStore,
		KanbanDispatcher: KanbanDispatcherConfig{
			Runner: dispatcher,
			Tick:   ticks,
		},
	}, &fakeKernel{}, slog.Default())

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- m.Run(runCtx) }()

	waitFor(t, 200*time.Millisecond, func() bool {
		status, err := statusStore.ReadRuntimeStatus(ctx)
		return err == nil && status.KanbanDispatcher.State == KanbanDispatcherStateStopped
	})
	if got := dispatcher.callCount(); got != 0 {
		t.Fatalf("dispatcher calls after closed tick channel = %d, want 0", got)
	}

	cancel()
	waitForKanbanManagerStop(t, "closed tick channel", done)
}
