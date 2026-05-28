package kanban

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type FailureKind string

const (
	FailureKindSpawn   FailureKind = "spawn"
	FailureKindTimeout FailureKind = "timeout"
	FailureKindCrash   FailureKind = "crash"
)

const (
	RunOutcomeWorkerZombie RunOutcome = "worker_zombie"
)

func (s *Store) HeartbeatTask(ctx context.Context, id string, heartbeatTTL time.Duration, note string) (bool, error) {
	if heartbeatTTL <= 0 {
		heartbeatTTL = 60 * time.Second
	}
	now := s.now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin kanban heartbeat: %w", err)
	}
	defer tx.Rollback()

	expires := now.Add(heartbeatTTL)
	result, err := tx.ExecContext(ctx, `
UPDATE tasks
SET claim_expires = ?, heartbeat_at = ?
WHERE id = ? AND status = ?`,
		expires.UnixMilli(), now.UnixMilli(), id, string(StatusRunning),
	)
	if err != nil {
		return false, fmt.Errorf("heartbeat kanban task %q: %w", id, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("heartbeat kanban task rows: %w", err)
	}
	if changed == 0 {
		if _, err := getTask(ctx, tx, id); err != nil {
			return false, err
		}
		return false, nil
	}
	if err := insertEvent(ctx, tx, id, "heartbeat", strings.TrimSpace(note)); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit kanban heartbeat: %w", err)
	}
	return true, nil
}

func (s *Store) ReclaimStaleHeartbeats(ctx context.Context, heartbeatTimeout time.Duration) ([]string, error) {
	if heartbeatTimeout <= 0 {
		heartbeatTimeout = 120 * time.Second
	}
	now := s.now().UTC()
	cutoff := now.Add(-heartbeatTimeout).UnixMilli()

	rows, err := s.db.QueryContext(ctx, `
SELECT id, failure_count
FROM tasks
WHERE status = ? AND heartbeat_at > 0 AND heartbeat_at <= ?
ORDER BY heartbeat_at ASC, id ASC`,
		string(StatusRunning), cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("list stale heartbeat kanban tasks: %w", err)
	}
	defer rows.Close()

	type zombieEntry struct {
		ID           string
		FailureCount int
	}
	var entries []zombieEntry
	for rows.Next() {
		var entry zombieEntry
		if err := rows.Scan(&entry.ID, &entry.FailureCount); err != nil {
			return nil, fmt.Errorf("scan stale heartbeat kanban task: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan stale heartbeat kanban tasks: %w", err)
	}

	var reclaimed []string
	for _, entry := range entries {
		if err := s.reclaimZombie(ctx, entry.ID, entry.FailureCount+1, now); err != nil {
			return nil, err
		}
		reclaimed = append(reclaimed, entry.ID)
	}
	return reclaimed, nil
}

func (s *Store) reclaimZombie(ctx context.Context, taskID string, newFailures int, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin kanban zombie reclaim: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
UPDATE tasks
SET status = ?, claim_lock = '', claim_expires = 0, heartbeat_at = 0,
	failure_count = ?
WHERE id = ? AND status = ?`,
		string(StatusReady), newFailures, taskID, string(StatusRunning),
	); err != nil {
		return fmt.Errorf("reclaim kanban zombie %q: %w", taskID, err)
	}

	payload := fmt.Sprintf(`{"failure_count":%d}`, newFailures)
	if err := insertEvent(ctx, tx, taskID, "worker_zombie", payload); err != nil {
		return err
	}
	if err := insertRun(ctx, tx, taskID, RunOutcomeWorkerZombie, "worker heartbeat stale", now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit kanban zombie reclaim: %w", err)
	}
	return nil
}

func (s *Store) IncrementFailureCounter(ctx context.Context, taskID string, kind FailureKind) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin kanban failure counter: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
UPDATE tasks
SET failure_count = failure_count + 1
WHERE id = ?`,
		taskID,
	); err != nil {
		return fmt.Errorf("increment kanban failure counter %q: %w", taskID, err)
	}

	eventKind := "spawn_failed"
	switch kind {
	case FailureKindTimeout:
		eventKind = "worker_timed_out"
	case FailureKindCrash:
		eventKind = "worker_crashed"
	}
	if err := insertEvent(ctx, tx, taskID, eventKind, string(kind)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit kanban failure counter: %w", err)
	}
	return nil
}

func (s *Store) AutoBlockIncompleteExit(ctx context.Context, taskID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "worker exited without completing task"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin kanban auto-block: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
UPDATE tasks
SET status = ?, result = ?, claim_lock = '', claim_expires = 0, heartbeat_at = 0
WHERE id = ? AND status = ?`,
		string(StatusBlocked), reason, taskID, string(StatusRunning),
	); err != nil {
		return fmt.Errorf("auto-block kanban task %q: %w", taskID, err)
	}

	if err := insertEvent(ctx, tx, taskID, "auto_blocked", reason); err != nil {
		return err
	}
	if err := insertRun(ctx, tx, taskID, RunOutcomeWorkerCrashed, reason, time.Now().UTC()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit kanban auto-block: %w", err)
	}
	return nil
}

func (t Task) FailureCount() int { return t.failureCount }
