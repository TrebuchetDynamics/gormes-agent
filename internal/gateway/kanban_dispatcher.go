package gateway

import (
	"context"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
)

type KanbanDispatcherRunner interface {
	RunOnce(context.Context) (kanban.DispatchResult, error)
}

type KanbanDispatcherConfig struct {
	Runner   KanbanDispatcherRunner
	Tick     <-chan time.Time
	Nudge    <-chan struct{}
	Interval time.Duration
}

func (m *Manager) startKanbanDispatcher(ctx context.Context, wg *sync.WaitGroup) {
	cfg := m.cfg.KanbanDispatcher
	if cfg.Runner == nil {
		return
	}
	if !m.claimKanbanDispatcherLoop() {
		return
	}
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		KanbanDispatcher: &KanbanDispatcherStatus{State: KanbanDispatcherStateRunning},
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer m.releaseKanbanDispatcherLoop()
		defer m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
			KanbanDispatcher: &KanbanDispatcherStatus{State: KanbanDispatcherStateStopped},
		})
		m.runKanbanDispatcherLoop(ctx, cfg)
	}()
}

func (m *Manager) claimKanbanDispatcherLoop() bool {
	m.kanbanDispatcherMu.Lock()
	defer m.kanbanDispatcherMu.Unlock()
	if m.kanbanDispatcherRunning {
		return false
	}
	m.kanbanDispatcherRunning = true
	return true
}

func (m *Manager) releaseKanbanDispatcherLoop() {
	m.kanbanDispatcherMu.Lock()
	defer m.kanbanDispatcherMu.Unlock()
	m.kanbanDispatcherRunning = false
}

func (m *Manager) runKanbanDispatcherLoop(ctx context.Context, cfg KanbanDispatcherConfig) {
	tick := cfg.Tick
	var ticker *time.Ticker
	if tick == nil {
		interval := cfg.Interval
		if interval <= 0 {
			interval = time.Minute
		}
		ticker = time.NewTicker(interval)
		defer ticker.Stop()
		tick = ticker.C
	}
	nudge := cfg.Nudge

	for {
		select {
		case <-ctx.Done():
			return
		case at, ok := <-tick:
			if !ok {
				return
			}
			m.runKanbanDispatcherOnce(ctx, cfg.Runner, at)
		case _, ok := <-nudge:
			if !ok {
				nudge = nil
				continue
			}
			m.runKanbanDispatcherOnce(ctx, cfg.Runner, m.now())
		}
	}
}

func (m *Manager) runKanbanDispatcherOnce(ctx context.Context, runner KanbanDispatcherRunner, at time.Time) {
	result, err := runner.RunOnce(ctx)
	if at.IsZero() {
		at = m.now()
	}
	update := KanbanDispatcherStatus{
		State:       KanbanDispatcherStateRunning,
		LastTickAt:  at.UTC().Format(time.RFC3339Nano),
		Spawned:     len(result.Spawned),
		SpawnFailed: len(result.SpawnFailedIDs),
		AutoBlocked: len(result.AutoBlockedTaskIDs),
	}
	if err != nil {
		update.State = KanbanDispatcherStateDegraded
		update.LastError = err.Error()
	}
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		KanbanDispatcher: &update,
	})
}
