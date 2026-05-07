package kanbantools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const (
	EvidenceInvalidArgs      = "kanban_invalid_args"
	EvidenceStoreUnavailable = "kanban_store_unavailable"
	EvidenceOwnershipDenied  = "kanban_task_ownership_denied"
)

type Config struct {
	DBPath  string
	TaskID  string
	Profile string
	Enabled bool
}

type kanbanTool struct {
	cfg         Config
	name        string
	description string
	schema      json.RawMessage
}

func ConfigFromEnv() Config {
	taskID := firstEnv("GORMES_KANBAN_TASK", "HERMES_KANBAN_TASK")
	return Config{
		DBPath:  config.KanbanDBPath(),
		TaskID:  taskID,
		Profile: firstEnv("GORMES_PROFILE", "HERMES_PROFILE"),
		Enabled: taskID != "" || envHasKanbanToolset(),
	}
}

func NewTools(cfg Config) []tools.Tool {
	cfg = normalizeConfig(cfg)
	if cfg.TaskID == "" && !cfg.Enabled {
		return nil
	}
	names := []string{
		"kanban_show",
		"kanban_complete",
		"kanban_block",
		"kanban_heartbeat",
		"kanban_comment",
		"kanban_create",
		"kanban_link",
	}
	out := make([]tools.Tool, 0, len(names))
	for _, name := range names {
		out = append(out, &kanbanTool{
			cfg:         cfg,
			name:        name,
			description: kanbanDescription(name),
			schema:      kanbanSchema(name),
		})
	}
	return out
}

func normalizeConfig(cfg Config) Config {
	if strings.TrimSpace(cfg.DBPath) == "" {
		cfg.DBPath = config.KanbanDBPath()
	}
	cfg.TaskID = strings.TrimSpace(cfg.TaskID)
	cfg.Profile = strings.TrimSpace(cfg.Profile)
	if cfg.Profile == "" {
		cfg.Profile = firstEnv("GORMES_PROFILE", "HERMES_PROFILE")
	}
	return cfg
}

func (t *kanbanTool) Name() string { return t.name }

func (t *kanbanTool) Description() string { return t.description }

func (t *kanbanTool) Schema() json.RawMessage {
	return append(json.RawMessage(nil), t.schema...)
}

func (*kanbanTool) Timeout() time.Duration { return 0 }

func (t *kanbanTool) Spec() tools.OperationSpec {
	return tools.OperationSpec{
		ToolDescriptor: tools.ToolDescriptor{Name: t.Name(), Description: t.Description(), Schema: t.Schema()},
		Mutating:       t.name != "kanban_show",
		Idempotent:     t.name == "kanban_show",
		PromptSafe:     true,
		TrustClass:     []string{"operator", "child-agent", "system"},
		AuditKind:      "kanban",
	}
}

func (t *kanbanTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	in, err := parseArgs(args)
	if err != nil {
		return kanbanError(EvidenceInvalidArgs, "invalid kanban args: "+err.Error()), nil
	}
	switch t.name {
	case "kanban_show":
		return t.show(ctx, in), nil
	case "kanban_complete":
		return t.complete(ctx, in), nil
	case "kanban_block":
		return t.block(ctx, in), nil
	case "kanban_heartbeat":
		return t.heartbeat(ctx, in), nil
	case "kanban_comment":
		return t.comment(ctx, in), nil
	case "kanban_create":
		return t.create(ctx, in), nil
	case "kanban_link":
		return t.link(ctx, in), nil
	default:
		return kanbanError(EvidenceStoreUnavailable, "unknown kanban tool"), nil
	}
}

func (t *kanbanTool) show(ctx context.Context, in map[string]any) json.RawMessage {
	taskID := t.defaultTaskID(stringValue(in["task_id"]))
	if taskID == "" {
		return kanbanError(EvidenceInvalidArgs, "task_id is required")
	}
	store, done, err := t.openStore(ctx)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	defer done()

	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	comments, err := store.ListComments(ctx, taskID)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	events, err := store.ListEvents(ctx, taskID)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	runs, err := store.ListRuns(ctx, taskID)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	contextBlock, err := store.BuildWorkerContext(ctx, taskID)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	return kanbanOK(map[string]any{
		"task":           task,
		"parents":        task.ParentIDs,
		"children":       task.ChildIDs,
		"comments":       comments,
		"events":         lastEvents(events, 50),
		"runs":           runs,
		"worker_context": contextBlock,
	})
}

func (t *kanbanTool) complete(ctx context.Context, in map[string]any) json.RawMessage {
	taskID := t.defaultTaskID(stringValue(in["task_id"]))
	if taskID == "" {
		return kanbanError(EvidenceInvalidArgs, "task_id is required")
	}
	if err := t.enforceWorkerTaskOwnership(taskID); err != nil {
		return kanbanError(EvidenceOwnershipDenied, err.Error())
	}
	summary := strings.TrimSpace(stringValue(in["summary"]))
	result := strings.TrimSpace(stringValue(in["result"]))
	if summary == "" && result == "" {
		return kanbanError(EvidenceInvalidArgs, "provide at least one of summary or result")
	}
	metadata, ok := metadataValue(in["metadata"])
	if !ok {
		return kanbanError(EvidenceInvalidArgs, "metadata must be an object")
	}
	store, done, err := t.openStore(ctx)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	defer done()

	if err := store.CompleteTask(ctx, taskID, kanban.CompleteTaskInput{Result: result, Summary: summary, Metadata: metadata}); err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	runID := latestRunID(ctx, store, taskID)
	return kanbanOK(map[string]any{"task_id": taskID, "run_id": runID})
}

func (t *kanbanTool) block(ctx context.Context, in map[string]any) json.RawMessage {
	taskID := t.defaultTaskID(stringValue(in["task_id"]))
	if taskID == "" {
		return kanbanError(EvidenceInvalidArgs, "task_id is required")
	}
	if err := t.enforceWorkerTaskOwnership(taskID); err != nil {
		return kanbanError(EvidenceOwnershipDenied, err.Error())
	}
	reason := strings.TrimSpace(stringValue(in["reason"]))
	if reason == "" {
		return kanbanError(EvidenceInvalidArgs, "reason is required")
	}
	store, done, err := t.openStore(ctx)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	defer done()
	if err := store.BlockTask(ctx, taskID, kanban.BlockTaskInput{Reason: reason}); err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	return kanbanOK(map[string]any{"task_id": taskID})
}

func (t *kanbanTool) heartbeat(ctx context.Context, in map[string]any) json.RawMessage {
	taskID := t.defaultTaskID(stringValue(in["task_id"]))
	if taskID == "" {
		return kanbanError(EvidenceInvalidArgs, "task_id is required")
	}
	if err := t.enforceWorkerTaskOwnership(taskID); err != nil {
		return kanbanError(EvidenceOwnershipDenied, err.Error())
	}
	store, done, err := t.openStore(ctx)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	defer done()
	ok, err := store.HeartbeatTask(ctx, taskID, 60*time.Second, stringValue(in["note"]))
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	if !ok {
		return kanbanError(EvidenceInvalidArgs, "task is not running")
	}
	return kanbanOK(map[string]any{"task_id": taskID})
}

func (t *kanbanTool) comment(ctx context.Context, in map[string]any) json.RawMessage {
	taskID := t.defaultTaskID(stringValue(in["task_id"]))
	if taskID == "" {
		return kanbanError(EvidenceInvalidArgs, "task_id is required")
	}
	if err := t.enforceWorkerTaskOwnership(taskID); err != nil {
		return kanbanError(EvidenceOwnershipDenied, err.Error())
	}
	body := strings.TrimSpace(stringValue(in["body"]))
	if body == "" {
		return kanbanError(EvidenceInvalidArgs, "body is required")
	}
	author := strings.TrimSpace(stringValue(in["author"]))
	if author == "" {
		author = t.cfg.Profile
	}
	if author == "" {
		author = "worker"
	}
	store, done, err := t.openStore(ctx)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	defer done()
	commentID, err := store.AddComment(ctx, taskID, author, body)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	return kanbanOK(map[string]any{"task_id": taskID, "comment_id": commentID})
}

func (t *kanbanTool) create(ctx context.Context, in map[string]any) json.RawMessage {
	title := strings.TrimSpace(stringValue(in["title"]))
	if title == "" {
		return kanbanError(EvidenceInvalidArgs, "title is required")
	}
	assignee := strings.TrimSpace(stringValue(in["assignee"]))
	if assignee == "" {
		return kanbanError(EvidenceInvalidArgs, "assignee is required")
	}
	parentIDs, ok := stringListValue(firstPresent(in, "parents", "parent_ids"))
	if !ok {
		return kanbanError(EvidenceInvalidArgs, "parents must be a list of task ids")
	}
	priority, ok := intValue(in["priority"])
	if !ok {
		return kanbanError(EvidenceInvalidArgs, "priority must be an integer")
	}
	workspaceKind := kanban.WorkspaceKind(strings.TrimSpace(stringValue(in["workspace_kind"])))
	if workspaceKind == "" {
		workspaceKind = kanban.WorkspaceScratch
	}
	store, done, err := t.openStore(ctx)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	defer done()
	task, err := store.CreateTask(ctx, kanban.CreateTaskInput{
		Title:         title,
		Body:          stringValue(in["body"]),
		Assignee:      assignee,
		ParentIDs:     parentIDs,
		Priority:      priority,
		WorkspaceKind: workspaceKind,
		WorkspacePath: stringValue(in["workspace_path"]),
		CreatedBy:     emptyDefault(t.cfg.Profile, "worker"),
	})
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	return kanbanOK(map[string]any{"task_id": task.ID, "status": string(task.Status)})
}

func (t *kanbanTool) link(ctx context.Context, in map[string]any) json.RawMessage {
	parentID := strings.TrimSpace(stringValue(in["parent_id"]))
	childID := strings.TrimSpace(stringValue(in["child_id"]))
	if parentID == "" || childID == "" {
		return kanbanError(EvidenceInvalidArgs, "both parent_id and child_id are required")
	}
	store, done, err := t.openStore(ctx)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	defer done()
	if err := store.LinkTasks(ctx, parentID, childID); err != nil {
		return kanbanError(EvidenceStoreUnavailable, err.Error())
	}
	return kanbanOK(map[string]any{"parent_id": parentID, "child_id": childID})
}

func (t *kanbanTool) defaultTaskID(taskID string) string {
	if strings.TrimSpace(taskID) != "" {
		return strings.TrimSpace(taskID)
	}
	return t.cfg.TaskID
}

func (t *kanbanTool) enforceWorkerTaskOwnership(taskID string) error {
	if t.cfg.TaskID == "" {
		return nil
	}
	if taskID != t.cfg.TaskID {
		return fmt.Errorf("worker is scoped to task %s; refusing to mutate %s", t.cfg.TaskID, taskID)
	}
	return nil
}

func (t *kanbanTool) openStore(ctx context.Context) (*kanban.Store, func(), error) {
	dbPath := strings.TrimSpace(t.cfg.DBPath)
	if dbPath == "" {
		return nil, nil, fmt.Errorf("kanban db path is required")
	}
	store, err := kanban.Open(ctx, dbPath)
	if err != nil {
		return nil, nil, err
	}
	return store, func() { _ = store.Close() }, nil
}

func parseArgs(args json.RawMessage) (map[string]any, error) {
	if len(strings.TrimSpace(string(args))) == 0 {
		args = json.RawMessage(`{}`)
	}
	var out map[string]any
	if err := json.Unmarshal(args, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = make(map[string]any)
	}
	return out, nil
}

func kanbanOK(fields map[string]any) json.RawMessage {
	out := make(map[string]any, len(fields)+1)
	out["ok"] = true
	for k, v := range fields {
		out[k] = v
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return kanbanError(EvidenceStoreUnavailable, "marshal kanban result: "+err.Error())
	}
	return raw
}

func kanbanError(evidence, message string) json.RawMessage {
	raw, err := json.Marshal(map[string]any{
		"ok":       false,
		"evidence": evidence,
		"error":    message,
	})
	if err != nil {
		return json.RawMessage(`{"ok":false,"evidence":"kanban_store_unavailable","error":"marshal kanban error"}`)
	}
	return raw
}

func metadataValue(value any) (map[string]any, bool) {
	if value == nil {
		return nil, true
	}
	out, ok := value.(map[string]any)
	return out, ok
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func stringListValue(value any) ([]string, bool) {
	if value == nil {
		return nil, true
	}
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, true
		}
		return []string{strings.TrimSpace(v)}, true
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(stringValue(item))
			if s == "" {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(item)
			if s == "" {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}

func intValue(value any) (int, bool) {
	if value == nil {
		return 0, true
	}
	switch v := value.(type) {
	case float64:
		i := int(v)
		return i, float64(i) == v
	case int:
		return v, true
	case json.Number:
		i, err := v.Int64()
		return int(i), err == nil
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, true
		}
		i, err := strconv.Atoi(strings.TrimSpace(v))
		return i, err == nil
	default:
		return 0, false
	}
}

func firstPresent(in map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := in[key]; ok {
			return value
		}
	}
	return nil
}

func latestRunID(ctx context.Context, store *kanban.Store, taskID string) any {
	runs, err := store.ListRuns(ctx, taskID)
	if err != nil || len(runs) == 0 {
		return nil
	}
	return runs[len(runs)-1].ID
}

func lastEvents(events []kanban.Event, limit int) []kanban.Event {
	if limit <= 0 || len(events) <= limit {
		return events
	}
	return events[len(events)-limit:]
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func envHasKanbanToolset() bool {
	for _, key := range []string{"GORMES_TOOLSETS", "GORMES_RUNTIME_TOOLSETS", "HERMES_TOOLSETS"} {
		for _, part := range strings.FieldsFunc(os.Getenv(key), func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\n'
		}) {
			if strings.EqualFold(strings.TrimSpace(part), "kanban") {
				return true
			}
		}
	}
	return false
}

func emptyDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func kanbanDescription(name string) string {
	switch name {
	case "kanban_show":
		return "Read a Kanban task, its dependency links, comments, run history, recent events, and worker context."
	case "kanban_complete":
		return "Mark the scoped Kanban task done with a human summary and optional structured metadata handoff."
	case "kanban_block":
		return "Move the scoped Kanban task to blocked with a reason for a human or orchestrator."
	case "kanban_heartbeat":
		return "Record that the scoped Kanban worker is still alive during a long operation."
	case "kanban_comment":
		return "Append a durable comment to the scoped Kanban task thread."
	case "kanban_create":
		return "Create a new Kanban task, optionally linked to parent tasks for dependency-gated promotion."
	case "kanban_link":
		return "Add a parent to child Kanban dependency edge after both tasks already exist."
	default:
		return "Use the native Kanban board."
	}
}

func kanbanSchema(name string) json.RawMessage {
	switch name {
	case "kanban_show":
		return json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string","description":"Task id. Defaults to the worker task from the environment."}},"required":[]}`)
	case "kanban_complete":
		return json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string","description":"Task id. Defaults to the worker task from the environment."},"summary":{"type":"string","description":"Human-readable handoff summary."},"metadata":{"type":"object","description":"Structured handoff facts for downstream workers."},"result":{"type":"string","description":"Legacy result field."}},"required":[]}`)
	case "kanban_block":
		return json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string","description":"Task id. Defaults to the worker task from the environment."},"reason":{"type":"string","description":"Reason the task is blocked."}},"required":["reason"]}`)
	case "kanban_heartbeat":
		return json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string","description":"Task id. Defaults to the worker task from the environment."},"note":{"type":"string","description":"Optional short progress note."}},"required":[]}`)
	case "kanban_comment":
		return json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string","description":"Task id. Defaults to the worker task from the environment."},"body":{"type":"string","description":"Comment body."},"author":{"type":"string","description":"Optional author override."}},"required":["body"]}`)
	case "kanban_create":
		return json.RawMessage(`{"type":"object","properties":{"title":{"type":"string"},"assignee":{"type":"string"},"body":{"type":"string"},"parents":{"oneOf":[{"type":"array","items":{"type":"string"}},{"type":"string"}]},"parent_ids":{"type":"array","items":{"type":"string"}},"priority":{"type":"integer"},"workspace_kind":{"type":"string","enum":["scratch","dir","worktree"]},"workspace_path":{"type":"string"}},"required":["title","assignee"]}`)
	case "kanban_link":
		return json.RawMessage(`{"type":"object","properties":{"parent_id":{"type":"string"},"child_id":{"type":"string"}},"required":["parent_id","child_id"]}`)
	default:
		return json.RawMessage(`{"type":"object","properties":{},"required":[]}`)
	}
}
