package kanban

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
	HeartbeatAt   time.Time     `json:"heartbeat_at,omitempty"`
	ParentIDs     []string      `json:"parent_ids,omitempty"`
	ChildIDs      []string      `json:"child_ids,omitempty"`

	failureCount int `json:"-"`
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
	Triage        bool
}

type ListFilter struct {
	Status   Status
	Assignee string
}

type CompleteTaskInput struct {
	Result   string
	Summary  string
	Metadata map[string]any
}

type ClaimTaskInput struct {
	Worker string
	TTL    time.Duration
}

type BlockTaskInput struct {
	Reason string
}

type PromoteTaskInput struct {
	Actor  string
	Reason string
	Force  bool
	DryRun bool
}

type PromoteTaskResult struct {
	TaskID   string
	Promoted bool
	DryRun   bool
	Forced   bool
	Reason   string
	Error    string
}

type RunOutcome string

const (
	RunOutcomeSpawned        RunOutcome = "spawned"
	RunOutcomeSpawnFailed    RunOutcome = "spawn_failed"
	RunOutcomeGaveUp         RunOutcome = "gave_up"
	RunOutcomeWorkerCrashed  RunOutcome = "worker_crashed"
	RunOutcomeWorkerTimedOut RunOutcome = "worker_timed_out"
	RunOutcomeCompleted      RunOutcome = "completed"
)

type TaskRun struct {
	ID        int64           `json:"id"`
	TaskID    string          `json:"task_id"`
	Outcome   RunOutcome      `json:"outcome"`
	Error     string          `json:"error,omitempty"`
	Summary   string          `json:"summary,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	StartedAt time.Time       `json:"started_at"`
	EndedAt   time.Time       `json:"ended_at"`
}

type Comment struct {
	ID        int64     `json:"id"`
	TaskID    string    `json:"task_id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type Event struct {
	ID        int64     `json:"id"`
	TaskID    string    `json:"task_id"`
	Kind      string    `json:"kind"`
	Payload   string    `json:"payload,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type NotifySubscriptionInput struct {
	Platform string
	ChatID   string
	ThreadID string
	UserID   string
}

type NotifySubscription struct {
	TaskID      string    `json:"task_id"`
	Platform    string    `json:"platform"`
	ChatID      string    `json:"chat_id"`
	ThreadID    string    `json:"thread_id,omitempty"`
	UserID      string    `json:"user_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastEventID int64     `json:"last_event_id"`
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

func (s *Store) DBPath() string {
	if s == nil {
		return ""
	}
	return s.dbPath
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
	if input.Triage {
		status = StatusTriage
	} else if !parentsDone {
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
	created_by, created_at, started_at, completed_at, result, claim_lock, claim_expires,
	heartbeat_at, failure_count
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
SELECT id, task_id, outcome, error, summary, metadata, started_at, ended_at
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
		var metadata string
		var startedAt, endedAt int64
		if err := rows.Scan(&run.ID, &run.TaskID, &outcome, &run.Error, &run.Summary, &metadata, &startedAt, &endedAt); err != nil {
			return nil, fmt.Errorf("scan kanban task run: %w", err)
		}
		run.Outcome = RunOutcome(outcome)
		if strings.TrimSpace(metadata) != "" {
			run.Metadata = json.RawMessage(metadata)
		}
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
	summary := strings.TrimSpace(input.Summary)
	if summary == "" {
		summary = strings.TrimSpace(input.Result)
	}
	metadata, err := marshalTaskRunMetadata(input.Metadata)
	if err != nil {
		return err
	}
	if err := insertRunWithDetails(ctx, tx, id, RunOutcomeCompleted, "", summary, metadata, now); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, id, "completed", summary); err != nil {
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

func (s *Store) PromoteTask(ctx context.Context, id string, input PromoteTaskInput) (PromoteTaskResult, error) {
	id = strings.TrimSpace(id)
	result := PromoteTaskResult{
		TaskID: id,
		DryRun: input.DryRun,
		Forced: input.Force,
		Reason: strings.TrimSpace(input.Reason),
	}
	if id == "" {
		result.Error = "task id is required"
		return result, nil
	}
	actor := strings.TrimSpace(input.Actor)
	if actor == "" {
		actor = "gormes"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin kanban promote: %w", err)
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, id).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		result.Error = fmt.Sprintf("task %s not found", id)
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("read kanban task %q for promotion: %w", id, err)
	}
	current := Status(status)
	if current != StatusTodo && current != StatusBlocked {
		result.Error = fmt.Sprintf("task %s is '%s'; promote only applies to 'todo' or 'blocked'", id, current)
		return result, nil
	}

	if !input.Force {
		unsatisfied, err := unsatisfiedParentIDs(ctx, tx, id)
		if err != nil {
			return result, err
		}
		if len(unsatisfied) > 0 {
			result.Error = fmt.Sprintf("unsatisfied parent dependencies: %s (use --force to override)", strings.Join(unsatisfied, ", "))
			return result, nil
		}
	}

	result.Promoted = true
	if input.DryRun {
		return result, nil
	}

	updated, err := tx.ExecContext(ctx, `UPDATE tasks SET status = ? WHERE id = ? AND status IN (?, ?)`, string(StatusReady), id, string(StatusTodo), string(StatusBlocked))
	if err != nil {
		return result, fmt.Errorf("promote kanban task %q: %w", id, err)
	}
	changed, err := updated.RowsAffected()
	if err != nil {
		return result, fmt.Errorf("promote kanban task rows: %w", err)
	}
	if changed != 1 {
		result.Promoted = false
		result.Error = fmt.Sprintf("task %s status changed during promotion", id)
		return result, nil
	}
	payload, err := json.Marshal(struct {
		Actor  string `json:"actor"`
		Reason string `json:"reason"`
		Forced bool   `json:"forced"`
	}{
		Actor:  actor,
		Reason: result.Reason,
		Forced: input.Force,
	})
	if err != nil {
		return result, fmt.Errorf("marshal kanban promote event: %w", err)
	}
	if err := insertEvent(ctx, tx, id, "promoted_manual", string(payload)); err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit kanban promote: %w", err)
	}
	return result, nil
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin kanban block event: %w", err)
	}
	defer tx.Rollback()
	if err := insertEvent(ctx, tx, id, "blocked", input.Reason); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit kanban block event: %w", err)
	}
	return nil
}

func (s *Store) AddComment(ctx context.Context, taskID, author, body string) (int64, error) {
	author = strings.TrimSpace(author)
	body = strings.TrimSpace(body)
	if author == "" {
		return 0, errors.New("comment author is required")
	}
	if body == "" {
		return 0, errors.New("comment body is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin kanban comment: %w", err)
	}
	defer tx.Rollback()
	if _, err := getTask(ctx, tx, taskID); err != nil {
		return 0, err
	}
	now := s.now().UTC()
	result, err := tx.ExecContext(ctx, `INSERT INTO task_comments(task_id, author, body, created_at) VALUES (?, ?, ?, ?)`, taskID, author, body, now.UnixMilli())
	if err != nil {
		return 0, fmt.Errorf("insert kanban comment: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read kanban comment id: %w", err)
	}
	if err := insertEvent(ctx, tx, taskID, "commented", author); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit kanban comment: %w", err)
	}
	return id, nil
}

func (s *Store) ListComments(ctx context.Context, taskID string) ([]Comment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, author, body, created_at FROM task_comments WHERE task_id = ? ORDER BY created_at ASC, id ASC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list kanban comments: %w", err)
	}
	defer rows.Close()
	var comments []Comment
	for rows.Next() {
		var comment Comment
		var createdAt int64
		if err := rows.Scan(&comment.ID, &comment.TaskID, &comment.Author, &comment.Body, &createdAt); err != nil {
			return nil, fmt.Errorf("scan kanban comment: %w", err)
		}
		comment.CreatedAt = millisToTime(createdAt)
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan kanban comments: %w", err)
	}
	return comments, nil
}

func (s *Store) ListEvents(ctx context.Context, taskID string) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, kind, payload, created_at FROM task_events WHERE task_id = ? ORDER BY created_at ASC, id ASC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list kanban events: %w", err)
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var event Event
		var createdAt int64
		if err := rows.Scan(&event.ID, &event.TaskID, &event.Kind, &event.Payload, &createdAt); err != nil {
			return nil, fmt.Errorf("scan kanban event: %w", err)
		}
		event.CreatedAt = millisToTime(createdAt)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan kanban events: %w", err)
	}
	return events, nil
}

func (s *Store) AddNotifySubscription(ctx context.Context, taskID string, input NotifySubscriptionInput) (NotifySubscription, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return NotifySubscription{}, errors.New("kanban task id is required")
	}
	normalized, err := normalizeNotifySubscriptionInput(input)
	if err != nil {
		return NotifySubscription{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return NotifySubscription{}, fmt.Errorf("begin kanban notify subscription: %w", err)
	}
	defer tx.Rollback()
	if _, err := getTask(ctx, tx, taskID); err != nil {
		return NotifySubscription{}, err
	}
	now := s.now().UTC()
	if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO kanban_notify_subs (
	task_id, platform, chat_id, thread_id, user_id, created_at, last_event_id
) VALUES (?, ?, ?, ?, ?, ?, 0)`,
		taskID, normalized.Platform, normalized.ChatID, normalized.ThreadID, normalized.UserID, now.UnixMilli(),
	); err != nil {
		return NotifySubscription{}, fmt.Errorf("insert kanban notify subscription: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return NotifySubscription{}, fmt.Errorf("commit kanban notify subscription: %w", err)
	}
	return s.getNotifySubscription(ctx, taskID, normalized)
}

func (s *Store) ListNotifySubscriptions(ctx context.Context, taskID string) ([]NotifySubscription, error) {
	query := `SELECT task_id, platform, chat_id, thread_id, user_id, created_at, last_event_id FROM kanban_notify_subs`
	var args []any
	taskID = strings.TrimSpace(taskID)
	if taskID != "" {
		query += ` WHERE task_id = ?`
		args = append(args, taskID)
	}
	query += ` ORDER BY task_id ASC, platform ASC, chat_id ASC, thread_id ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list kanban notify subscriptions: %w", err)
	}
	defer rows.Close()
	var subscriptions []NotifySubscription
	for rows.Next() {
		sub, err := scanNotifySubscription(rows)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan kanban notify subscriptions: %w", err)
	}
	return subscriptions, nil
}

func (s *Store) RemoveNotifySubscription(ctx context.Context, taskID string, input NotifySubscriptionInput) (bool, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false, errors.New("kanban task id is required")
	}
	normalized, err := normalizeNotifySubscriptionInput(input)
	if err != nil {
		return false, err
	}
	result, err := s.db.ExecContext(ctx, `
DELETE FROM kanban_notify_subs
WHERE task_id = ? AND platform = ? AND chat_id = ? AND thread_id = ?`,
		taskID, normalized.Platform, normalized.ChatID, normalized.ThreadID,
	)
	if err != nil {
		return false, fmt.Errorf("remove kanban notify subscription: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read kanban notify subscription removal count: %w", err)
	}
	return changed > 0, nil
}

func (s *Store) UnseenEventsForSubscription(ctx context.Context, taskID string, input NotifySubscriptionInput, kinds []string) (int64, []Event, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return 0, nil, errors.New("kanban task id is required")
	}
	normalized, err := normalizeNotifySubscriptionInput(input)
	if err != nil {
		return 0, nil, err
	}
	sub, ok, err := s.findNotifySubscription(ctx, taskID, normalized)
	if err != nil {
		return 0, nil, err
	}
	if !ok {
		return 0, []Event{}, nil
	}

	query := `SELECT id, task_id, kind, payload, created_at FROM task_events WHERE task_id = ? AND id > ?`
	args := []any{taskID, sub.LastEventID}
	kinds = normalizeNotifyKinds(kinds)
	if len(kinds) > 0 {
		placeholders := make([]string, 0, len(kinds))
		for _, kind := range kinds {
			placeholders = append(placeholders, "?")
			args = append(args, kind)
		}
		query += ` AND kind IN (` + strings.Join(placeholders, ",") + `)`
	}
	query += ` ORDER BY id ASC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, nil, fmt.Errorf("list kanban notify events: %w", err)
	}
	defer rows.Close()

	cursor := sub.LastEventID
	var events []Event
	for rows.Next() {
		var event Event
		var createdAt int64
		if err := rows.Scan(&event.ID, &event.TaskID, &event.Kind, &event.Payload, &createdAt); err != nil {
			return 0, nil, fmt.Errorf("scan kanban notify event: %w", err)
		}
		event.CreatedAt = millisToTime(createdAt)
		if event.ID > cursor {
			cursor = event.ID
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("scan kanban notify events: %w", err)
	}
	return cursor, events, nil
}

func (s *Store) AdvanceNotifyCursor(ctx context.Context, taskID string, input NotifySubscriptionInput, cursor int64) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return errors.New("kanban task id is required")
	}
	normalized, err := normalizeNotifySubscriptionInput(input)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE kanban_notify_subs
SET last_event_id = ?
WHERE task_id = ? AND platform = ? AND chat_id = ? AND thread_id = ?`,
		cursor, taskID, normalized.Platform, normalized.ChatID, normalized.ThreadID,
	)
	if err != nil {
		return fmt.Errorf("advance kanban notify cursor: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read kanban notify cursor update count: %w", err)
	}
	if changed == 0 {
		return fmt.Errorf("kanban notify subscription for %s %s:%s not found", taskID, normalized.Platform, normalized.ChatID)
	}
	return nil
}

func (s *Store) PruneTerminalEvents(ctx context.Context, olderThan time.Duration) (int64, error) {
	if olderThan < 0 {
		return 0, errors.New("event retention must be >= 0")
	}
	cutoff := s.now().UTC().Add(-olderThan).UnixMilli()
	result, err := s.db.ExecContext(ctx, `
DELETE FROM task_events
WHERE created_at < ?
  AND task_id IN (
    SELECT id FROM tasks WHERE status IN (?, ?)
  )`,
		cutoff, string(StatusDone), string(StatusArchived),
	)
	if err != nil {
		return 0, fmt.Errorf("prune kanban terminal events: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count pruned kanban terminal events: %w", err)
	}
	return deleted, nil
}

func (s *Store) PruneWorkerLogs(olderThan time.Duration) (int, error) {
	if olderThan < 0 {
		return 0, errors.New("log retention must be >= 0")
	}
	logRoot := kanbanWorkerLogRootForDBPath(s.dbPath)
	entries, err := os.ReadDir(logRoot)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read kanban worker log root: %w", err)
	}

	cutoff := s.now().UTC().Add(-olderThan)
	deleted := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}
		path := filepath.Join(logRoot, entry.Name())
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return deleted, fmt.Errorf("remove kanban worker log %s: %w", path, err)
		}
		deleted++
	}
	return deleted, nil
}

func (s *Store) ReadWorkerLog(taskID string, tailBytes int64) (string, bool, error) {
	if tailBytes < 0 {
		return "", false, errors.New("tail bytes must be >= 0")
	}
	path, err := s.workerLogPath(taskID)
	if err != nil {
		return "", false, err
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read kanban worker log: %w", err)
	}
	defer file.Close()

	if tailBytes <= 0 {
		data, err := io.ReadAll(file)
		if err != nil {
			return "", false, fmt.Errorf("read kanban worker log: %w", err)
		}
		return string(data), true, nil
	}

	info, err := file.Stat()
	if err != nil {
		return "", false, fmt.Errorf("stat kanban worker log: %w", err)
	}
	size := info.Size()
	if size > tailBytes {
		start := size - tailBytes
		if _, err := file.Seek(start, io.SeekStart); err != nil {
			return "", false, fmt.Errorf("seek kanban worker log: %w", err)
		}
		probe, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			return "", false, fmt.Errorf("probe kanban worker log: %w", err)
		}
		var skipped []byte
		for {
			buf := make([]byte, 1)
			n, readErr := file.Read(buf)
			if n == 1 {
				skipped = append(skipped, buf[0])
				if buf[0] == '\n' {
					break
				}
			}
			if readErr != nil {
				if errors.Is(readErr, io.EOF) {
					break
				}
				return "", false, fmt.Errorf("read kanban worker log tail: %w", readErr)
			}
		}
		pos, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			return "", false, fmt.Errorf("probe kanban worker log: %w", err)
		}
		if len(skipped) > 0 && skipped[len(skipped)-1] != '\n' && pos >= size {
			if _, err := file.Seek(probe, io.SeekStart); err != nil {
				return "", false, fmt.Errorf("seek kanban worker log: %w", err)
			}
		}
	} else {
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "", false, fmt.Errorf("seek kanban worker log: %w", err)
		}
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return "", false, fmt.Errorf("read kanban worker log tail: %w", err)
	}
	return string(data), true, nil
}

func (s *Store) workerLogPath(taskID string) (string, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return "", errors.New("task id is required")
	}
	if strings.ContainsAny(taskID, `/\`) || taskID == "." || taskID == ".." || filepath.Base(taskID) != taskID {
		return "", fmt.Errorf("unsafe kanban task id %q", taskID)
	}
	return filepath.Join(kanbanWorkerLogRootForDBPath(s.dbPath), taskID+".log"), nil
}

func (s *Store) BuildWorkerContext(ctx context.Context, taskID string) (string, error) {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Kanban task %s: %s\n\n", task.ID, task.Title)
	fmt.Fprintf(&b, "Assignee: %s\nStatus: %s\nWorkspace: %s @ %s\n\n", emptyAs(task.Assignee, "(unassigned)"), task.Status, task.WorkspaceKind, emptyAs(task.WorkspacePath, "(unresolved)"))
	if strings.TrimSpace(task.Body) != "" {
		fmt.Fprintf(&b, "## Body\n%s\n\n", strings.TrimSpace(task.Body))
	}
	if len(task.ParentIDs) > 0 {
		b.WriteString("## Parent handoffs\n")
		for _, parentID := range task.ParentIDs {
			parent, err := s.GetTask(ctx, parentID)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "- %s: %s (%s)\n", parent.ID, parent.Title, parent.Status)
			runs, err := s.ListRuns(ctx, parent.ID)
			if err != nil {
				return "", err
			}
			if run, ok := latestCompletedRun(runs); ok {
				if strings.TrimSpace(run.Summary) != "" {
					fmt.Fprintf(&b, "  Summary: %s\n", strings.TrimSpace(run.Summary))
				}
				if len(run.Metadata) > 0 {
					fmt.Fprintf(&b, "  Metadata: %s\n", strings.TrimSpace(string(run.Metadata)))
				}
			} else if strings.TrimSpace(parent.Result) != "" {
				fmt.Fprintf(&b, "  Result: %s\n", strings.TrimSpace(parent.Result))
			}
			comments, err := s.ListComments(ctx, parent.ID)
			if err != nil {
				return "", err
			}
			for _, comment := range comments {
				fmt.Fprintf(&b, "  Comment from %s: %s\n", comment.Author, comment.Body)
			}
		}
		b.WriteString("\n")
	}
	comments, err := s.ListComments(ctx, task.ID)
	if err != nil {
		return "", err
	}
	if len(comments) > 0 {
		b.WriteString("## Comment thread\n")
		for _, comment := range comments {
			fmt.Fprintf(&b, "**%s**: %s\n", comment.Author, comment.Body)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n", nil
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
	var createdAt, startedAt, completedAt, claimExpires, heartbeatAt safeMillis
	err := q.QueryRowContext(ctx, `
SELECT id, title, body, assignee, status, priority, workspace_kind, workspace_path,
	created_by, created_at, started_at, completed_at, result, claim_lock, claim_expires,
	heartbeat_at, failure_count
FROM tasks
WHERE id = ?`, id).Scan(
		&task.ID, &task.Title, &task.Body, &task.Assignee, &status, &task.Priority,
		&workspaceKind, &task.WorkspacePath, &task.CreatedBy, &createdAt, &startedAt,
		&completedAt, &task.Result, &task.ClaimLock, &claimExpires,
		&heartbeatAt, &task.failureCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, fmt.Errorf("kanban task %q not found", id)
	}
	if err != nil {
		return Task{}, fmt.Errorf("read kanban task %q: %w", id, err)
	}
	task.Status = Status(status)
	task.WorkspaceKind = WorkspaceKind(workspaceKind)
	task.CreatedAt = createdAt.Time()
	task.StartedAt = startedAt.Time()
	task.CompletedAt = completedAt.Time()
	task.ClaimExpires = claimExpires.Time()
	task.HeartbeatAt = heartbeatAt.Time()
	return task, nil
}

func scanTask(scanner interface {
	Scan(dest ...any) error
}) (Task, error) {
	var task Task
	var status, workspaceKind string
	var createdAt, startedAt, completedAt, claimExpires, heartbeatAt safeMillis
	if err := scanner.Scan(
		&task.ID, &task.Title, &task.Body, &task.Assignee, &status, &task.Priority,
		&workspaceKind, &task.WorkspacePath, &task.CreatedBy, &createdAt, &startedAt,
		&completedAt, &task.Result, &task.ClaimLock, &claimExpires,
		&heartbeatAt, &task.failureCount,
	); err != nil {
		return Task{}, fmt.Errorf("scan kanban task: %w", err)
	}
	task.Status = Status(status)
	task.WorkspaceKind = WorkspaceKind(workspaceKind)
	task.CreatedAt = createdAt.Time()
	task.StartedAt = startedAt.Time()
	task.CompletedAt = completedAt.Time()
	task.ClaimExpires = claimExpires.Time()
	task.HeartbeatAt = heartbeatAt.Time()
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
		if Status(status) != StatusDone && Status(status) != StatusArchived {
			allDone = false
		}
	}
	return allDone, nil
}

func unsatisfiedParentIDs(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, childID string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `
SELECT t.id
FROM tasks t
JOIN task_links l ON l.parent_id = t.id
WHERE l.child_id = ? AND t.status NOT IN (?, ?)
ORDER BY t.id`, childID, string(StatusDone), string(StatusArchived))
	if err != nil {
		return nil, fmt.Errorf("list unsatisfied kanban parents for %q: %w", childID, err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan unsatisfied kanban parent: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan unsatisfied kanban parents: %w", err)
	}
	return ids, nil
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

func normalizeNotifySubscriptionInput(input NotifySubscriptionInput) (NotifySubscriptionInput, error) {
	normalized := NotifySubscriptionInput{
		Platform: strings.TrimSpace(input.Platform),
		ChatID:   strings.TrimSpace(input.ChatID),
		ThreadID: strings.TrimSpace(input.ThreadID),
		UserID:   strings.TrimSpace(input.UserID),
	}
	if normalized.Platform == "" {
		return NotifySubscriptionInput{}, errors.New("notify platform is required")
	}
	if normalized.ChatID == "" {
		return NotifySubscriptionInput{}, errors.New("notify chat id is required")
	}
	return normalized, nil
}

func normalizeNotifyKinds(kinds []string) []string {
	if len(kinds) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(kinds))
	out := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		kind = strings.TrimSpace(kind)
		if kind == "" {
			continue
		}
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		out = append(out, kind)
	}
	return out
}

func (s *Store) getNotifySubscription(ctx context.Context, taskID string, input NotifySubscriptionInput) (NotifySubscription, error) {
	sub, ok, err := s.findNotifySubscription(ctx, taskID, input)
	if err != nil {
		return NotifySubscription{}, err
	}
	if !ok {
		return NotifySubscription{}, fmt.Errorf("kanban notify subscription for %s %s:%s not found", taskID, input.Platform, input.ChatID)
	}
	return sub, nil
}

func (s *Store) findNotifySubscription(ctx context.Context, taskID string, input NotifySubscriptionInput) (NotifySubscription, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT task_id, platform, chat_id, thread_id, user_id, created_at, last_event_id
FROM kanban_notify_subs
WHERE task_id = ? AND platform = ? AND chat_id = ? AND thread_id = ?`,
		taskID, input.Platform, input.ChatID, input.ThreadID,
	)
	sub, err := scanNotifySubscription(row)
	if errors.Is(err, sql.ErrNoRows) {
		return NotifySubscription{}, false, nil
	}
	if err != nil {
		return NotifySubscription{}, false, err
	}
	return sub, true, nil
}

func scanNotifySubscription(scanner interface {
	Scan(dest ...any) error
}) (NotifySubscription, error) {
	var sub NotifySubscription
	var createdAt int64
	if err := scanner.Scan(&sub.TaskID, &sub.Platform, &sub.ChatID, &sub.ThreadID, &sub.UserID, &createdAt, &sub.LastEventID); err != nil {
		return NotifySubscription{}, err
	}
	sub.CreatedAt = millisToTime(createdAt)
	return sub, nil
}

func (s *Store) migrateSchema(ctx context.Context) error {
	if err := s.ensureTaskColumn(ctx, "spawn_failures", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureTaskColumn(ctx, "last_spawn_error", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureTaskRunColumn(ctx, "summary", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureTaskRunColumn(ctx, "metadata", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureTaskColumn(ctx, "failure_count", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureTaskColumn(ctx, "heartbeat_at", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureTaskColumn(ctx context.Context, name, definition string) error {
	return s.ensureColumn(ctx, "tasks", name, definition)
}

func (s *Store) ensureTaskRunColumn(ctx context.Context, name, definition string) error {
	return s.ensureColumn(ctx, "task_runs", name, definition)
}

func (s *Store) ensureColumn(ctx context.Context, table, name, definition string) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return fmt.Errorf("inspect kanban %s schema: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var columnName, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan kanban %s schema: %w", table, err)
		}
		if columnName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("scan kanban %s schema: %w", table, err)
	}
	if err := s.addColumnIfMissing(ctx, table, name, definition); err != nil {
		return fmt.Errorf("migrate kanban %s.%s: %w", table, name, err)
	}
	return nil
}

func (s *Store) addColumnIfMissing(ctx context.Context, table, name, definition string) error {
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, name, definition)); err != nil {
		if isDuplicateColumnMigrationError(err) {
			return nil
		}
		return err
	}
	return nil
}

func isDuplicateColumnMigrationError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate column name")
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

type safeMillis struct {
	value int64
}

func (m *safeMillis) Scan(src any) error {
	if m == nil {
		return nil
	}
	switch value := src.(type) {
	case nil:
		m.value = 0
	case int64:
		m.value = value
	case int:
		m.value = int64(value)
	case int32:
		m.value = int64(value)
	case int16:
		m.value = int64(value)
	case int8:
		m.value = int64(value)
	case uint:
		m.value = int64(value)
	case uint32:
		m.value = int64(value)
	case uint16:
		m.value = int64(value)
	case uint8:
		m.value = int64(value)
	case []byte:
		m.value = parseSafeMillisString(string(value))
	case string:
		m.value = parseSafeMillisString(value)
	default:
		m.value = 0
	}
	return nil
}

func (m safeMillis) Time() time.Time {
	return millisToTime(m.value)
}

func parseSafeMillisString(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
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
	summary TEXT NOT NULL DEFAULT '',
	metadata TEXT NOT NULL DEFAULT '',
	started_at INTEGER NOT NULL,
	ended_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS kanban_notify_subs (
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	platform TEXT NOT NULL,
	chat_id TEXT NOT NULL,
	thread_id TEXT NOT NULL DEFAULT '',
	user_id TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	last_event_id INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY(task_id, platform, chat_id, thread_id)
);

CREATE INDEX IF NOT EXISTS idx_notify_task ON kanban_notify_subs(task_id);
`

func marshalTaskRunMetadata(metadata map[string]any) (json.RawMessage, error) {
	if len(metadata) == 0 {
		return nil, nil
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("encode kanban run metadata: %w", err)
	}
	return raw, nil
}

func latestCompletedRun(runs []TaskRun) (TaskRun, bool) {
	for i := len(runs) - 1; i >= 0; i-- {
		if runs[i].Outcome == RunOutcomeCompleted {
			return runs[i], true
		}
	}
	return TaskRun{}, false
}

func emptyAs(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
