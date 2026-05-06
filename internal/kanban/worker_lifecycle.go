package kanban

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type ProcessSnapshot struct {
	PID       int
	Live      bool
	StartedAt time.Time
}

type WorkerProcessController interface {
	Snapshot(context.Context, int) (ProcessSnapshot, error)
	Stop(context.Context, int, time.Time) error
}

type WorkerLifecycleMonitor struct {
	Store        *Store
	Processes    WorkerProcessController
	Now          func() time.Time
	MaxRuntime   time.Duration
	FailureLimit int
}

type workerSpawnEvent struct {
	PID       int    `json:"pid"`
	StartedAt string `json:"started_at,omitempty"`
}

type workerSpawnRecord struct {
	PID       int
	StartedAt time.Time
}

func (m WorkerLifecycleMonitor) DetectCrashedWorkers(ctx context.Context) ([]string, error) {
	if err := m.valid(); err != nil {
		return nil, err
	}
	tasks, err := m.Store.ListTasks(ctx, ListFilter{Status: StatusRunning})
	if err != nil {
		return nil, err
	}
	var crashed []string
	for _, task := range tasks {
		spawn, ok, err := m.Store.latestWorkerSpawn(ctx, task.ID)
		if err != nil {
			return nil, err
		}
		if !ok || spawn.PID <= 0 {
			continue
		}
		snapshot, err := m.Processes.Snapshot(ctx, spawn.PID)
		if err != nil {
			return nil, fmt.Errorf("worker_crashed: inspect pid %d: %w", spawn.PID, err)
		}
		if snapshot.Live {
			continue
		}
		if err := m.Store.releaseWorkerLifecycleFailure(ctx, task.ID, RunOutcomeWorkerCrashed, fmt.Sprintf("worker_crashed: pid=%d", spawn.PID), m.failureLimit()); err != nil {
			return nil, err
		}
		crashed = append(crashed, task.ID)
	}
	return crashed, nil
}

func (m WorkerLifecycleMonitor) EnforceMaxRuntime(ctx context.Context) ([]string, error) {
	if err := m.valid(); err != nil {
		return nil, err
	}
	if m.MaxRuntime <= 0 {
		return nil, nil
	}
	tasks, err := m.Store.ListTasks(ctx, ListFilter{Status: StatusRunning})
	if err != nil {
		return nil, err
	}
	now := m.now()
	var timedOut []string
	for _, task := range tasks {
		spawn, ok, err := m.Store.latestWorkerSpawn(ctx, task.ID)
		if err != nil {
			return nil, err
		}
		if !ok || spawn.PID <= 0 {
			continue
		}
		startedAt := spawn.StartedAt
		if startedAt.IsZero() {
			startedAt = task.StartedAt
		}
		if startedAt.IsZero() || now.Sub(startedAt) < m.MaxRuntime {
			continue
		}
		snapshot, err := m.Processes.Snapshot(ctx, spawn.PID)
		if err != nil {
			return nil, fmt.Errorf("worker_timed_out: inspect pid %d: %w", spawn.PID, err)
		}
		if !snapshot.Live || !sameWorkerProcess(spawn, snapshot) {
			continue
		}
		if err := m.Processes.Stop(ctx, spawn.PID, startedAt); err != nil {
			return nil, fmt.Errorf("worker_timed_out: stop pid %d: %w", spawn.PID, err)
		}
		message := fmt.Sprintf("worker_timed_out: pid=%d elapsed=%s limit=%s", spawn.PID, now.Sub(startedAt).Round(time.Second), m.MaxRuntime)
		if err := m.Store.releaseWorkerLifecycleFailure(ctx, task.ID, RunOutcomeWorkerTimedOut, message, m.failureLimit()); err != nil {
			return nil, err
		}
		timedOut = append(timedOut, task.ID)
	}
	return timedOut, nil
}

func (m WorkerLifecycleMonitor) valid() error {
	if m.Store == nil {
		return errors.New("worker lifecycle store is required")
	}
	if m.Processes == nil {
		return errors.New("worker lifecycle process controller is required")
	}
	return nil
}

func (m WorkerLifecycleMonitor) now() time.Time {
	if m.Now != nil {
		return m.Now().UTC()
	}
	return time.Now().UTC()
}

func (m WorkerLifecycleMonitor) failureLimit() int {
	if m.FailureLimit > 0 {
		return m.FailureLimit
	}
	return 5
}

func sameWorkerProcess(spawn workerSpawnRecord, snapshot ProcessSnapshot) bool {
	if spawn.StartedAt.IsZero() || snapshot.StartedAt.IsZero() {
		return true
	}
	return spawn.StartedAt.Equal(snapshot.StartedAt.UTC())
}

func (s *Store) latestWorkerSpawn(ctx context.Context, taskID string) (workerSpawnRecord, bool, error) {
	var payload string
	err := s.db.QueryRowContext(ctx, `
SELECT payload
FROM task_events
WHERE task_id = ? AND kind = 'spawned'
ORDER BY id DESC
LIMIT 1`, taskID).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return workerSpawnRecord{}, false, nil
	}
	if err != nil {
		return workerSpawnRecord{}, false, fmt.Errorf("read kanban spawned event %q: %w", taskID, err)
	}
	var event workerSpawnEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return workerSpawnRecord{}, false, fmt.Errorf("decode kanban spawned event %q: %w", taskID, err)
	}
	record := workerSpawnRecord{PID: event.PID}
	if event.StartedAt != "" {
		startedAt, err := time.Parse(time.RFC3339Nano, event.StartedAt)
		if err != nil {
			return workerSpawnRecord{}, false, fmt.Errorf("decode kanban spawned event started_at %q: %w", taskID, err)
		}
		record.StartedAt = startedAt.UTC()
	}
	return record, record.PID > 0, nil
}

func (s *Store) releaseWorkerLifecycleFailure(ctx context.Context, taskID string, outcome RunOutcome, message string, failureLimit int) error {
	message = truncateFailure(message)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin kanban worker lifecycle release: %w", err)
	}
	defer tx.Rollback()

	var current int
	if err := tx.QueryRowContext(ctx, `SELECT spawn_failures FROM tasks WHERE id = ?`, taskID).Scan(&current); err != nil {
		return fmt.Errorf("read kanban worker failures %q: %w", taskID, err)
	}
	failures := current + 1
	status := StatusReady
	result := message
	if failures >= failureLimit {
		status = StatusBlocked
		result = fmt.Sprintf("task_circuit_open after %d worker failures: %s", failures, message)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE tasks
SET status = ?, result = ?, claim_lock = '', claim_expires = 0,
	spawn_failures = ?, last_spawn_error = ?
WHERE id = ? AND status = ?`, string(status), result, failures, message, taskID, string(StatusRunning)); err != nil {
		return fmt.Errorf("release kanban worker lifecycle %q: %w", taskID, err)
	}
	if err := insertRun(ctx, tx, taskID, outcome, message, s.now().UTC()); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, taskID, string(outcome), message); err != nil {
		return err
	}
	if status == StatusBlocked {
		if err := insertEvent(ctx, tx, taskID, "task_circuit_open", result); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit kanban worker lifecycle release: %w", err)
	}
	return nil
}
