package kanban

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Dispatcher struct {
	Store    *Store
	Spawner  SpawnFunc
	ClaimTTL time.Duration
	Worker   string
}

type DispatchOptions struct {
	MaxSpawn     int
	FailureLimit int
}

type DispatchResult struct {
	ReclaimedIDs       []string
	Spawned            []SpawnRecord
	SkippedUnassigned  []string
	SpawnFailedIDs     []string
	AutoBlockedTaskIDs []string
}

type SpawnRecord struct {
	TaskID        string `json:"task_id"`
	Assignee      string `json:"assignee"`
	WorkspacePath string `json:"workspace_path"`
	PID           int    `json:"pid,omitempty"`
}

type SpawnRequest struct {
	Task          Task
	WorkspacePath string
	Env           map[string]string
}

type SpawnResult struct {
	PID int
}

type SpawnFunc func(context.Context, SpawnRequest) (SpawnResult, error)

func (f SpawnFunc) SpawnKanbanWorker(ctx context.Context, req SpawnRequest) (SpawnResult, error) {
	return f(ctx, req)
}

func (d Dispatcher) RunOnce(ctx context.Context, opts DispatchOptions) (DispatchResult, error) {
	if d.Store == nil {
		return DispatchResult{}, errors.New("kanban dispatcher store is required")
	}
	result := DispatchResult{}
	reclaimed, err := d.Store.reclaimStaleClaims(ctx)
	if err != nil {
		return result, err
	}
	result.ReclaimedIDs = reclaimed

	ready, err := d.Store.readyTasks(ctx)
	if err != nil {
		return result, err
	}
	attempted := 0
	for _, task := range ready {
		if opts.MaxSpawn > 0 && attempted >= opts.MaxSpawn {
			break
		}
		if strings.TrimSpace(task.Assignee) == "" {
			result.SkippedUnassigned = append(result.SkippedUnassigned, task.ID)
			continue
		}
		claimed, ok, err := d.Store.ClaimTask(ctx, task.ID, ClaimTaskInput{
			Worker: d.workerName(),
			TTL:    d.claimTTL(),
		})
		if err != nil {
			return result, err
		}
		if !ok {
			continue
		}
		attempted++

		workspace, err := d.Store.resolveWorkspace(ctx, claimed)
		if err != nil {
			result.SpawnFailedIDs = append(result.SpawnFailedIDs, claimed.ID)
			blocked, err := d.Store.releaseFailedSpawn(ctx, claimed.ID, err.Error(), opts.failureLimit())
			if err != nil {
				return result, err
			}
			if blocked {
				result.AutoBlockedTaskIDs = append(result.AutoBlockedTaskIDs, claimed.ID)
			}
			continue
		}
		spawner := d.Spawner
		if spawner == nil {
			result.SpawnFailedIDs = append(result.SpawnFailedIDs, claimed.ID)
			blocked, err := d.Store.releaseFailedSpawn(ctx, claimed.ID, "kanban dispatcher spawner unavailable", opts.failureLimit())
			if err != nil {
				return result, err
			}
			if blocked {
				result.AutoBlockedTaskIDs = append(result.AutoBlockedTaskIDs, claimed.ID)
			}
			continue
		}
		spawned, err := spawner.SpawnKanbanWorker(ctx, SpawnRequest{
			Task:          claimed,
			WorkspacePath: workspace,
			Env:           d.Store.workerEnv(claimed, workspace),
		})
		if err != nil {
			result.SpawnFailedIDs = append(result.SpawnFailedIDs, claimed.ID)
			blocked, err := d.Store.releaseFailedSpawn(ctx, claimed.ID, err.Error(), opts.failureLimit())
			if err != nil {
				return result, err
			}
			if blocked {
				result.AutoBlockedTaskIDs = append(result.AutoBlockedTaskIDs, claimed.ID)
			}
			continue
		}
		result.Spawned = append(result.Spawned, SpawnRecord{
			TaskID:        claimed.ID,
			Assignee:      claimed.Assignee,
			WorkspacePath: workspace,
			PID:           spawned.PID,
		})
		if err := d.Store.recordSpawned(ctx, claimed.ID, spawned.PID); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (o DispatchOptions) failureLimit() int {
	if o.FailureLimit > 0 {
		return o.FailureLimit
	}
	return 5
}

func (d Dispatcher) claimTTL() time.Duration {
	if d.ClaimTTL > 0 {
		return d.ClaimTTL
	}
	return 15 * time.Minute
}

func (d Dispatcher) workerName() string {
	if strings.TrimSpace(d.Worker) != "" {
		return strings.TrimSpace(d.Worker)
	}
	return "kanban-dispatcher"
}

func (s *Store) reclaimStaleClaims(ctx context.Context) ([]string, error) {
	now := s.now().UTC().UnixMilli()
	rows, err := s.db.QueryContext(ctx, `
SELECT id
FROM tasks
WHERE status = ? AND claim_expires > 0 AND claim_expires <= ?
ORDER BY claim_expires ASC, id ASC`, string(StatusRunning), now)
	if err != nil {
		return nil, fmt.Errorf("list stale kanban claims: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan stale kanban claim: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan stale kanban claims: %w", err)
	}
	for _, id := range ids {
		if _, err := s.db.ExecContext(ctx, `
UPDATE tasks
SET status = ?, claim_lock = '', claim_expires = 0
WHERE id = ? AND status = ?`, string(StatusReady), id, string(StatusRunning)); err != nil {
			return nil, fmt.Errorf("reclaim stale kanban claim %q: %w", id, err)
		}
	}
	return ids, nil
}

func (s *Store) readyTasks(ctx context.Context) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, title, body, assignee, status, priority, workspace_kind, workspace_path,
	created_by, created_at, started_at, completed_at, result, claim_lock, claim_expires
FROM tasks
WHERE status = ?
ORDER BY priority DESC, created_at ASC, id ASC`, string(StatusReady))
	if err != nil {
		return nil, fmt.Errorf("list ready kanban tasks: %w", err)
	}
	defer rows.Close()
	var tasks []Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		if err := s.fillLinks(ctx, &task); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan ready kanban tasks: %w", err)
	}
	return tasks, nil
}

func (s *Store) resolveWorkspace(ctx context.Context, task Task) (string, error) {
	switch task.WorkspaceKind {
	case WorkspaceScratch:
		path := filepath.Join(filepath.Dir(s.dbPath), "kanban", "workspaces", task.ID)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return "", fmt.Errorf("prepare scratch workspace: %w", err)
		}
		return path, nil
	case WorkspaceDir, WorkspaceWorktree:
		if strings.TrimSpace(task.WorkspacePath) == "" {
			return "", fmt.Errorf("%s workspace path is required", task.WorkspaceKind)
		}
		return task.WorkspacePath, nil
	default:
		return "", fmt.Errorf("unsupported workspace kind %q", task.WorkspaceKind)
	}
}

func (s *Store) workerEnv(task Task, workspace string) map[string]string {
	return map[string]string{
		"GORMES_KANBAN_DB":        s.dbPath,
		"GORMES_KANBAN_TASK":      task.ID,
		"GORMES_KANBAN_WORKSPACE": workspace,
		"GORMES_PROFILE":          task.Assignee,
	}
}

func (s *Store) releaseFailedSpawn(ctx context.Context, taskID, message string, failureLimit int) (bool, error) {
	message = truncateFailure(message)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin failed kanban spawn release: %w", err)
	}
	defer tx.Rollback()

	var current int
	if err := tx.QueryRowContext(ctx, `SELECT spawn_failures FROM tasks WHERE id = ?`, taskID).Scan(&current); err != nil {
		return false, fmt.Errorf("read kanban spawn failures %q: %w", taskID, err)
	}
	failures := current + 1
	blocked := failures >= failureLimit
	status := StatusReady
	result := message
	outcome := RunOutcomeSpawnFailed
	if blocked {
		status = StatusBlocked
		result = fmt.Sprintf("spawn failure circuit open after %d attempts: %s", failures, message)
		outcome = RunOutcomeGaveUp
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE tasks
SET status = ?, result = ?, claim_lock = '', claim_expires = 0,
	spawn_failures = ?, last_spawn_error = ?
WHERE id = ? AND status = ?`, string(status), result, failures, message, taskID, string(StatusRunning)); err != nil {
		return false, fmt.Errorf("release failed kanban spawn %q: %w", taskID, err)
	}
	if err := insertRun(ctx, tx, taskID, outcome, message, s.now().UTC()); err != nil {
		return false, err
	}
	eventKind := "spawn_failed"
	if blocked {
		eventKind = "gave_up"
	}
	if err := insertEvent(ctx, tx, taskID, eventKind, result); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit failed kanban spawn release: %w", err)
	}
	return blocked, nil
}

func (s *Store) recordSpawned(ctx context.Context, taskID string, pid int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin kanban spawn record: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET spawn_failures = 0, last_spawn_error = '' WHERE id = ?`, taskID); err != nil {
		return fmt.Errorf("clear kanban spawn failures %q: %w", taskID, err)
	}
	if err := insertRun(ctx, tx, taskID, RunOutcomeSpawned, "", s.now().UTC()); err != nil {
		return err
	}
	if pid > 0 {
		if err := insertEvent(ctx, tx, taskID, "spawned", fmt.Sprintf(`{"pid":%d}`, pid)); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit kanban spawn record: %w", err)
	}
	return nil
}

func insertRun(ctx context.Context, tx *sql.Tx, taskID string, outcome RunOutcome, message string, at time.Time) error {
	if _, err := tx.ExecContext(ctx, `
INSERT INTO task_runs(task_id, outcome, error, started_at, ended_at)
VALUES (?, ?, ?, ?, ?)`, taskID, string(outcome), message, at.UnixMilli(), at.UnixMilli()); err != nil {
		return fmt.Errorf("insert kanban run %q: %w", outcome, err)
	}
	return nil
}

func truncateFailure(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 500 {
		return message[:500]
	}
	return message
}
