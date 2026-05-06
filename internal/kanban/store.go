package kanban

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
)

// Status is the Hermes Kanban task lifecycle state stored in kanban.db.
type Status string

const (
	StatusTriage   Status = "triage"
	StatusTodo     Status = "todo"
	StatusReady    Status = "ready"
	StatusRunning  Status = "running"
	StatusBlocked  Status = "blocked"
	StatusDone     Status = "done"
	StatusArchived Status = "archived"
)

// WorkspaceKind describes where a worker should perform task work.
type WorkspaceKind string

const (
	WorkspaceScratch  WorkspaceKind = "scratch"
	WorkspaceWorktree WorkspaceKind = "worktree"
	WorkspaceDir      WorkspaceKind = "dir"
)

// Task is the durable Kanban task record.
type Task struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Body          string        `json:"body,omitempty"`
	Assignee      string        `json:"assignee,omitempty"`
	Status        Status        `json:"status"`
	Priority      int           `json:"priority,omitempty"`
	WorkspaceKind WorkspaceKind `json:"workspace_kind"`
	WorkspacePath string        `json:"workspace_path,omitempty"`
	CreatedBy     string        `json:"created_by,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	StartedAt     time.Time     `json:"started_at,omitempty"`
	CompletedAt   time.Time     `json:"completed_at,omitempty"`
	Result        string        `json:"result,omitempty"`
	ClaimLock     string        `json:"claim_lock,omitempty"`
	ClaimExpires  time.Time     `json:"claim_expires,omitempty"`
	ParentIDs     []string      `json:"parent_ids,omitempty"`
	ChildIDs      []string      `json:"child_ids,omitempty"`
}

// CreateTaskInput is the public creation contract for CLI and later tools.
type CreateTaskInput struct {
	Title         string
	Body          string
	Assignee      string
	ParentIDs     []string
	Priority      int
	WorkspaceKind WorkspaceKind
	WorkspacePath string
	CreatedBy     string
}

type ListFilter struct {
	Status   Status
	Assignee string
}

type CompleteTaskInput struct {
	Result string
}

type ClaimTaskInput struct {
	Worker string
	TTL    time.Duration
}

type BlockTaskInput struct {
	Reason string
}

type RunOutcome string

const (
	RunOutcomeSpawned        RunOutcome = "spawned"
	RunOutcomeSpawnFailed    RunOutcome = "spawn_failed"
	RunOutcomeGaveUp         RunOutcome = "gave_up"
	RunOutcomeWorkerCrashed  RunOutcome = "worker_crashed"
	RunOutcomeWorkerTimedOut RunOutcome = "worker_timed_out"
)

type TaskRun struct {
	ID        int64      `json:"id"`
	TaskID    string     `json:"task_id"`
	Outcome   RunOutcome `json:"outcome"`
	Error     string     `json:"error,omitempty"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   time.Time  `json:"ended_at"`
}

type Store struct {
	db     *sql.DB
	dbPath string
	now    func() time.Time
}

func Open(ctx context.Context, path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("kanban db path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("prepare kanban db dir: %w", err)
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open kanban db: %w", err)
	}
	store := &Store{db: db, dbPath: path, now: time.Now}
	if err := store.Init(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Init(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `PRAGMA journal_mode=WAL;`); err != nil {
		return fmt.Errorf("enable kanban wal: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA busy_timeout=5000;`); err != nil {
		return fmt.Errorf("set kanban busy timeout: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys=ON;`); err != nil {
		return fmt.Errorf("enable kanban foreign keys: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, kanbanSchema); err != nil {
		return fmt.Errorf("init kanban schema: %w", err)
	}
	if err := s.migrateSchema(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Store) CreateTask(ctx context.Context, input CreateTaskInput) (Task, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Task{}, errors.New("task title is required")
	}
	workspaceKind := input.WorkspaceKind
	if workspaceKind == "" {
		workspaceKind = WorkspaceScratch
	}
	if !validWorkspaceKind(workspaceKind) {
		return Task{}, fmt.Errorf("unsupported workspace kind %q", workspaceKind)
	}
	now := s.now().UTC()
	id, err := newTaskID()
	if err != nil {
		return Task{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, fmt.Errorf("begin kanban create: %w", err)
	}
	defer tx.Rollback()

	parentsDone, err := taskParentsDone(ctx, tx, input.ParentIDs)
	if err != nil {
		return Task{}, err
	}
	status := StatusReady
	if !parentsDone {
		status = StatusTodo
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO tasks (
	id, title, body, assignee, status, priority, workspace_kind,
	workspace_path, created_by, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, title, input.Body, strings.TrimSpace(input.Assignee), string(status),
		input.Priority, string(workspaceKind), input.WorkspacePath, input.CreatedBy, now.UnixMilli(),
	); err != nil {
		return Task{}, fmt.Errorf("insert kanban task: %w", err)
	}
	for _, parentID := range input.ParentIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_links(parent_id, child_id) VALUES (?, ?)`, parentID, id); err != nil {
			return Task{}, fmt.Errorf("link kanban parent %q: %w", parentID, err)
		}
	}
	if err := insertEvent(ctx, tx, id, "created", ""); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("commit kanban create: %w", err)
	}
	return s.GetTask(ctx, id)
}

func (s *Store) GetTask(ctx context.Context, id string) (Task, error) {
	task, err := getTask(ctx, s.db, id)
	if err != nil {
		return Task{}, err
	}
	if err := s.fillLinks(ctx, &task); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Store) ListTasks(ctx context.Context, filter ListFilter) ([]Task, error) {
	query := `
SELECT id, title, body, assignee, status, priority, workspace_kind, workspace_path,
	created_by, created_at, started_at, completed_at, result, claim_lock, claim_expires
FROM tasks`
	var clauses []string
	var args []any
	if filter.Status != "" {
		if !validStatus(filter.Status) {
			return nil, fmt.Errorf("unsupported kanban status %q", filter.Status)
		}
		clauses = append(clauses, "status = ?")
		args = append(args, string(filter.Status))
	}
	if strings.TrimSpace(filter.Assignee) != "" {
		clauses = append(clauses, "assignee = ?")
		args = append(args, strings.TrimSpace(filter.Assignee))
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_at ASC, id ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list kanban tasks: %w", err)
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
		return nil, fmt.Errorf("list kanban tasks rows: %w", err)
	}
	return tasks, nil
}

func (s *Store) ListRuns(ctx context.Context, taskID string) ([]TaskRun, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, task_id, outcome, error, started_at, ended_at
FROM task_runs
WHERE task_id = ?
ORDER BY id ASC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list kanban task runs: %w", err)
	}
	defer rows.Close()
	var runs []TaskRun
	for rows.Next() {
		var run TaskRun
		var outcome string
		var startedAt, endedAt int64
		if err := rows.Scan(&run.ID, &run.TaskID, &outcome, &run.Error, &startedAt, &endedAt); err != nil {
			return nil, fmt.Errorf("scan kanban task run: %w", err)
		}
		run.Outcome = RunOutcome(outcome)
		run.StartedAt = millisToTime(startedAt)
		run.EndedAt = millisToTime(endedAt)
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan kanban task runs: %w", err)
	}
	return runs, nil
}

func (s *Store) CompleteTask(ctx context.Context, id string, input CompleteTaskInput) error {
	now := s.now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin kanban complete: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
UPDATE tasks
SET status = ?, result = ?, completed_at = ?, claim_lock = '', claim_expires = 0
WHERE id = ? AND status != ? AND status != ?`,
		string(StatusDone), input.Result, now.UnixMilli(), id, string(StatusDone), string(StatusArchived),
	)
	if err != nil {
		return fmt.Errorf("complete kanban task: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("complete kanban task rows: %w", err)
	}
	if changed == 0 {
		if _, err := getTask(ctx, tx, id); err != nil {
			return err
		}
	}
	if err := insertEvent(ctx, tx, id, "completed", input.Result); err != nil {
		return err
	}
	if err := promoteReadyChildren(ctx, tx, id); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit kanban complete: %w", err)
	}
	return nil
}

func (s *Store) ClaimTask(ctx context.Context, id string, input ClaimTaskInput) (Task, bool, error) {
	worker := strings.TrimSpace(input.Worker)
	if worker == "" {
		worker = "worker"
	}
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	now := s.now().UTC()
	expires := now.Add(ttl)
	result, err := s.db.ExecContext(ctx, `
UPDATE tasks
SET status = ?, claim_lock = ?, started_at = ?, claim_expires = ?
WHERE id = ? AND status = ?`,
		string(StatusRunning), worker, now.UnixMilli(), expires.UnixMilli(), id, string(StatusReady),
	)
	if err != nil {
		return Task{}, false, fmt.Errorf("claim kanban task %q: %w", id, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return Task{}, false, fmt.Errorf("claim kanban task rows: %w", err)
	}
	task, err := s.GetTask(ctx, id)
	if err != nil {
		return Task{}, false, err
	}
	return task, changed == 1, nil
}

func (s *Store) LinkTasks(ctx context.Context, parentID, childID string) error {
	parentID = strings.TrimSpace(parentID)
	childID = strings.TrimSpace(childID)
	if parentID == "" || childID == "" {
		return errors.New("parent and child task ids are required")
	}
	if parentID == childID {
		return fmt.Errorf("kanban task %q cannot depend on itself", parentID)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin kanban link: %w", err)
	}
	defer tx.Rollback()

	if _, err := getTask(ctx, tx, parentID); err != nil {
		return err
	}
	if _, err := getTask(ctx, tx, childID); err != nil {
		return err
	}
	cycle, err := hasDependencyPath(ctx, tx, childID, parentID)
	if err != nil {
		return err
	}
	if cycle {
		return fmt.Errorf("kanban link %s -> %s would create a cycle", parentID, childID)
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO task_links(parent_id, child_id) VALUES (?, ?)`, parentID, childID); err != nil {
		return fmt.Errorf("link kanban tasks: %w", err)
	}
	if err := recomputeTaskReadiness(ctx, tx, childID); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, childID, "linked", parentID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit kanban link: %w", err)
	}
	return nil
}

func (s *Store) BlockTask(ctx context.Context, id string, input BlockTaskInput) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE tasks
SET status = ?, result = ?, claim_lock = '', claim_expires = 0
WHERE id = ? AND status != ? AND status != ?`,
		string(StatusBlocked), input.Reason, id, string(StatusDone), string(StatusArchived),
	)
	if err != nil {
		return fmt.Errorf("block kanban task %q: %w", id, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("block kanban task rows: %w", err)
	}
	if changed == 0 {
		if _, err := s.GetTask(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UnblockTask(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin kanban unblock: %w", err)
	}
	defer tx.Rollback()
	task, err := getTask(ctx, tx, id)
	if err != nil {
		return err
	}
	if task.Status != StatusBlocked {
		return nil
	}
	if err := recomputeTaskReadiness(ctx, tx, id); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit kanban unblock: %w", err)
	}
	return nil
}

func (s *Store) fillLinks(ctx context.Context, task *Task) error {
	parents, err := queryIDs(ctx, s.db, `SELECT parent_id FROM task_links WHERE child_id = ? ORDER BY parent_id`, task.ID)
	if err != nil {
		return err
	}
	children, err := queryIDs(ctx, s.db, `SELECT child_id FROM task_links WHERE parent_id = ? ORDER BY child_id`, task.ID)
	if err != nil {
		return err
	}
	task.ParentIDs = parents
	task.ChildIDs = children
	return nil
}

func getTask(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id string) (Task, error) {
	var task Task
	var status, workspaceKind string
	var createdAt, startedAt, completedAt, claimExpires int64
	err := q.QueryRowContext(ctx, `
SELECT id, title, body, assignee, status, priority, workspace_kind, workspace_path,
	created_by, created_at, started_at, completed_at, result, claim_lock, claim_expires
FROM tasks
WHERE id = ?`, id).Scan(
		&task.ID, &task.Title, &task.Body, &task.Assignee, &status, &task.Priority,
		&workspaceKind, &task.WorkspacePath, &task.CreatedBy, &createdAt, &startedAt,
		&completedAt, &task.Result, &task.ClaimLock, &claimExpires,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, fmt.Errorf("kanban task %q not found", id)
	}
	if err != nil {
		return Task{}, fmt.Errorf("read kanban task %q: %w", id, err)
	}
	task.Status = Status(status)
	task.WorkspaceKind = WorkspaceKind(workspaceKind)
	task.CreatedAt = millisToTime(createdAt)
	task.StartedAt = millisToTime(startedAt)
	task.CompletedAt = millisToTime(completedAt)
	task.ClaimExpires = millisToTime(claimExpires)
	return task, nil
}

func scanTask(scanner interface {
	Scan(dest ...any) error
}) (Task, error) {
	var task Task
	var status, workspaceKind string
	var createdAt, startedAt, completedAt, claimExpires int64
	if err := scanner.Scan(
		&task.ID, &task.Title, &task.Body, &task.Assignee, &status, &task.Priority,
		&workspaceKind, &task.WorkspacePath, &task.CreatedBy, &createdAt, &startedAt,
		&completedAt, &task.Result, &task.ClaimLock, &claimExpires,
	); err != nil {
		return Task{}, fmt.Errorf("scan kanban task: %w", err)
	}
	task.Status = Status(status)
	task.WorkspaceKind = WorkspaceKind(workspaceKind)
	task.CreatedAt = millisToTime(createdAt)
	task.StartedAt = millisToTime(startedAt)
	task.CompletedAt = millisToTime(completedAt)
	task.ClaimExpires = millisToTime(claimExpires)
	return task, nil
}

func taskParentsDone(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, parentIDs []string) (bool, error) {
	allDone := true
	for _, parentID := range parentIDs {
		parentID = strings.TrimSpace(parentID)
		if parentID == "" {
			return false, errors.New("parent task id is required")
		}
		var status string
		err := q.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, parentID).Scan(&status)
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("parent task %q not found", parentID)
		}
		if err != nil {
			return false, fmt.Errorf("read parent task %q: %w", parentID, err)
		}
		if Status(status) != StatusDone {
			allDone = false
		}
	}
	return allDone, nil
}

func promoteReadyChildren(ctx context.Context, tx *sql.Tx, parentID string) error {
	rows, err := tx.QueryContext(ctx, `SELECT child_id FROM task_links WHERE parent_id = ?`, parentID)
	if err != nil {
		return fmt.Errorf("list kanban children for %q: %w", parentID, err)
	}
	defer rows.Close()
	var childIDs []string
	for rows.Next() {
		var childID string
		if err := rows.Scan(&childID); err != nil {
			return fmt.Errorf("scan kanban child: %w", err)
		}
		childIDs = append(childIDs, childID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("scan kanban children: %w", err)
	}
	for _, childID := range childIDs {
		parentIDs, err := parentsForTask(ctx, tx, childID)
		if err != nil {
			return err
		}
		parentsDone, err := taskParentsDone(ctx, tx, parentIDs)
		if err != nil {
			return err
		}
		if !parentsDone {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET status = ? WHERE id = ? AND status = ?`, string(StatusReady), childID, string(StatusTodo)); err != nil {
			return fmt.Errorf("promote kanban child %q: %w", childID, err)
		}
	}
	return nil
}

func recomputeTaskReadiness(ctx context.Context, tx *sql.Tx, taskID string) error {
	parents, err := parentsForTask(ctx, tx, taskID)
	if err != nil {
		return err
	}
	parentsDone, err := taskParentsDone(ctx, tx, parents)
	if err != nil {
		return err
	}
	status := StatusTodo
	if parentsDone {
		status = StatusReady
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE tasks
SET status = ?, result = '', claim_lock = '', claim_expires = 0
WHERE id = ? AND status IN (?, ?)`,
		string(status), taskID, string(StatusReady), string(StatusTodo),
	); err != nil {
		return fmt.Errorf("recompute kanban task %q readiness: %w", taskID, err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE tasks
SET status = ?, result = '', claim_lock = '', claim_expires = 0
WHERE id = ? AND status = ?`,
		string(status), taskID, string(StatusBlocked),
	); err != nil {
		return fmt.Errorf("unblock kanban task %q: %w", taskID, err)
	}
	return nil
}

func hasDependencyPath(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, fromID, toID string) (bool, error) {
	var found int
	err := q.QueryRowContext(ctx, `
WITH RECURSIVE reach(id) AS (
	SELECT child_id FROM task_links WHERE parent_id = ?
	UNION
	SELECT task_links.child_id
	FROM task_links
	JOIN reach ON task_links.parent_id = reach.id
)
SELECT 1 FROM reach WHERE id = ? LIMIT 1`, fromID, toID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check kanban dependency cycle: %w", err)
	}
	return found == 1, nil
}

func parentsForTask(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, childID string) ([]string, error) {
	ids, err := queryIDs(ctx, q, `SELECT parent_id FROM task_links WHERE child_id = ?`, childID)
	if err != nil {
		return nil, fmt.Errorf("list kanban parents for %q: %w", childID, err)
	}
	return ids, nil
}

func queryIDs(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, query string, args ...any) ([]string, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func insertEvent(ctx context.Context, tx *sql.Tx, taskID, kind, payload string) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO task_events(task_id, kind, payload, created_at) VALUES (?, ?, ?, ?)`, taskID, kind, payload, time.Now().UTC().UnixMilli()); err != nil {
		return fmt.Errorf("insert kanban event %q: %w", kind, err)
	}
	return nil
}

func (s *Store) migrateSchema(ctx context.Context) error {
	if err := s.ensureTaskColumn(ctx, "spawn_failures", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureTaskColumn(ctx, "last_spawn_error", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureTaskColumn(ctx context.Context, name, definition string) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(tasks)`)
	if err != nil {
		return fmt.Errorf("inspect kanban tasks schema: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var columnName, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan kanban tasks schema: %w", err)
		}
		if columnName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("scan kanban tasks schema: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE tasks ADD COLUMN %s %s", name, definition)); err != nil {
		return fmt.Errorf("migrate kanban tasks.%s: %w", name, err)
	}
	return nil
}

func validWorkspaceKind(kind WorkspaceKind) bool {
	switch kind {
	case WorkspaceScratch, WorkspaceWorktree, WorkspaceDir:
		return true
	default:
		return false
	}
}

func validStatus(status Status) bool {
	switch status {
	case StatusTriage, StatusTodo, StatusReady, StatusRunning, StatusBlocked, StatusDone, StatusArchived:
		return true
	default:
		return false
	}
}

func newTaskID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("create kanban task id: %w", err)
	}
	return "t_" + hex.EncodeToString(b[:]), nil
}

func millisToTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

const kanbanSchema = `
CREATE TABLE IF NOT EXISTS tasks (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	body TEXT NOT NULL DEFAULT '',
	assignee TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	priority INTEGER NOT NULL DEFAULT 0,
	workspace_kind TEXT NOT NULL DEFAULT 'scratch',
	workspace_path TEXT NOT NULL DEFAULT '',
	created_by TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	started_at INTEGER NOT NULL DEFAULT 0,
	completed_at INTEGER NOT NULL DEFAULT 0,
	result TEXT NOT NULL DEFAULT '',
	claim_lock TEXT NOT NULL DEFAULT '',
	claim_expires INTEGER NOT NULL DEFAULT 0,
	spawn_failures INTEGER NOT NULL DEFAULT 0,
	last_spawn_error TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS task_links (
	parent_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	child_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	PRIMARY KEY(parent_id, child_id)
);

CREATE TABLE IF NOT EXISTS task_comments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	author TEXT NOT NULL DEFAULT '',
	body TEXT NOT NULL,
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS task_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	kind TEXT NOT NULL,
	payload TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS task_runs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	outcome TEXT NOT NULL,
	error TEXT NOT NULL DEFAULT '',
	started_at INTEGER NOT NULL,
	ended_at INTEGER NOT NULL
);
`
