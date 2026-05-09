package kanban

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

const triageSpecifierSystemPrompt = `You are the Kanban triage specifier for the Gormes board.
A user dropped a rough idea into the Triage column. Turn it into a concrete, actionable task spec that an autonomous worker can execute without more clarification.

Output a single JSON object with exactly two keys:
{"title":"<tightened task title, <= 80 chars, imperative voice>","body":"<multi-line spec with Goal, Approach, Acceptance criteria, and Out of scope when useful>"}

Keep the tightened title close to the original idea, preserve existing detail, never invent scope creep, and output only the JSON object.`

type TriageSpecRequest struct {
	TaskID string
	Title  string
	Body   string
}

type TriageSpecifier interface {
	CompleteTriageSpec(context.Context, TriageSpecRequest) (string, error)
}

type SpecifyOptions struct {
	Author string
}

type SpecifyOutcome struct {
	TaskID   string `json:"task_id"`
	OK       bool   `json:"ok"`
	Reason   string `json:"reason"`
	NewTitle string `json:"new_title,omitempty"`
	Status   Status `json:"status,omitempty"`
}

func SpecifyTriageTask(ctx context.Context, store *Store, taskID string, specifier TriageSpecifier, opts SpecifyOptions) (SpecifyOutcome, error) {
	outcome := SpecifyOutcome{TaskID: strings.TrimSpace(taskID)}
	if store == nil {
		return outcome, errors.New("kanban store is required")
	}
	if outcome.TaskID == "" {
		outcome.Reason = "task id is required"
		return outcome, nil
	}

	task, err := store.GetTask(ctx, outcome.TaskID)
	if err != nil {
		outcome.Reason = "unknown task id"
		return outcome, nil
	}
	if task.Status != StatusTriage {
		outcome.Reason = fmt.Sprintf("task is not in triage (status=%q)", task.Status)
		return outcome, nil
	}
	if specifier == nil {
		outcome.Reason = "auxiliary client unavailable"
		return outcome, nil
	}

	raw, err := specifier.CompleteTriageSpec(ctx, TriageSpecRequest{
		TaskID: task.ID,
		Title:  task.Title,
		Body:   task.Body,
	})
	if err != nil {
		outcome.Reason = "LLM error: triage specifier failed"
		return outcome, nil
	}
	title, body, ok, reason := parseTriageSpecResponse(raw)
	if !ok {
		outcome.Reason = reason
		return outcome, nil
	}

	updated, changed, err := store.specifyTask(ctx, task.ID, title, body, opts.Author)
	if err != nil {
		return outcome, err
	}
	if updated.ID == "" {
		outcome.Reason = "task moved out of triage before promotion"
		return outcome, nil
	}
	outcome.OK = true
	outcome.Reason = "specified"
	outcome.NewTitle = changed.newTitle
	outcome.Status = updated.Status
	return outcome, nil
}

type HermesTriageSpecifier struct {
	Client      hermes.Client
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}

func (s HermesTriageSpecifier) CompleteTriageSpec(ctx context.Context, req TriageSpecRequest) (string, error) {
	if s.Client == nil {
		return "", errors.New("auxiliary client unavailable")
	}
	model := strings.TrimSpace(s.Model)
	if model == "" {
		return "", errors.New("auxiliary model unavailable")
	}
	if s.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Timeout)
		defer cancel()
	}
	temp := s.Temperature
	if temp == 0 {
		temp = 0.3
	}
	maxTokens := s.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1500
	}
	stream, err := s.Client.OpenStream(ctx, hermes.ChatRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: &temp,
		Stream:      true,
		Messages: []hermes.Message{
			{Role: "system", Content: triageSpecifierSystemPrompt},
			{Role: "user", Content: formatTriageSpecifierUserMessage(req)},
		},
	})
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var b strings.Builder
	for {
		ev, err := stream.Recv(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if ev.Kind == hermes.EventToken {
			b.WriteString(ev.Token)
		}
	}
	return b.String(), nil
}

func formatTriageSpecifierUserMessage(req TriageSpecRequest) string {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		body = "(no body)"
	}
	return fmt.Sprintf("Task id: %s\nCurrent title: %s\nCurrent body:\n%s\n", req.TaskID, truncateTriageSpecText(req.Title, 400), truncateTriageSpecText(body, 4000))
}

func truncateTriageSpecText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	if limit <= 0 {
		return ""
	}
	return text[:limit-1] + "..."
}

type specifyChanged struct {
	newTitle string
}

func (s *Store) specifyTask(ctx context.Context, taskID string, title, body *string, author string) (Task, specifyChanged, error) {
	if title != nil && strings.TrimSpace(*title) == "" {
		return Task{}, specifyChanged{}, errors.New("title cannot be blank")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, specifyChanged{}, fmt.Errorf("begin kanban specify: %w", err)
	}
	defer tx.Rollback()

	var existingTitle, existingBody string
	err = tx.QueryRowContext(ctx, `SELECT title, body FROM tasks WHERE id = ? AND status = ?`, taskID, string(StatusTriage)).Scan(&existingTitle, &existingBody)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, specifyChanged{}, nil
	}
	if err != nil {
		return Task{}, specifyChanged{}, fmt.Errorf("read triage task %q: %w", taskID, err)
	}

	sets := []string{"status = ?"}
	args := []any{string(StatusTodo)}
	var changedFields []string
	changed := specifyChanged{}
	if title != nil {
		next := strings.TrimSpace(*title)
		if next != existingTitle {
			sets = append(sets, "title = ?")
			args = append(args, next)
			changedFields = append(changedFields, "title")
			changed.newTitle = next
		}
	}
	if body != nil {
		next := *body
		if next != existingBody {
			sets = append(sets, "body = ?")
			args = append(args, next)
			changedFields = append(changedFields, "body")
		}
	}
	args = append(args, taskID, string(StatusTriage))
	result, err := tx.ExecContext(ctx, `UPDATE tasks SET `+strings.Join(sets, ", ")+` WHERE id = ? AND status = ?`, args...)
	if err != nil {
		return Task{}, specifyChanged{}, fmt.Errorf("update triage task %q: %w", taskID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Task{}, specifyChanged{}, fmt.Errorf("update triage task rows: %w", err)
	}
	if rows != 1 {
		return Task{}, specifyChanged{}, nil
	}
	if len(changedFields) > 0 && strings.TrimSpace(author) != "" {
		comment := "Specified - updated " + strings.Join(changedFields, ", ") + " and promoted to todo."
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_comments(task_id, author, body, created_at) VALUES (?, ?, ?, ?)`, taskID, strings.TrimSpace(author), comment, s.now().UTC().UnixMilli()); err != nil {
			return Task{}, specifyChanged{}, fmt.Errorf("insert specify audit comment: %w", err)
		}
	}
	payload := ""
	if len(changedFields) > 0 {
		raw, _ := json.Marshal(map[string][]string{"changed_fields": changedFields})
		payload = string(raw)
	}
	if err := insertEvent(ctx, tx, taskID, "specified", payload); err != nil {
		return Task{}, specifyChanged{}, err
	}
	if err := recomputeTaskReadiness(ctx, tx, taskID); err != nil {
		return Task{}, specifyChanged{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, specifyChanged{}, fmt.Errorf("commit kanban specify: %w", err)
	}
	task, err := s.GetTask(ctx, taskID)
	return task, changed, err
}

func parseTriageSpecResponse(raw string) (title, body *string, ok bool, reason string) {
	stripped := strings.TrimSpace(raw)
	if stripped == "" {
		return nil, nil, false, "LLM returned an empty response"
	}
	if parsed, parsedOK := extractTriageSpecJSON(stripped); parsedOK {
		var gotTitle *string
		if text, ok := parsed["title"].(string); ok && strings.TrimSpace(text) != "" {
			next := strings.TrimSpace(text)
			gotTitle = &next
		}
		var gotBody *string
		if text, ok := parsed["body"].(string); ok && strings.TrimSpace(text) != "" {
			next := text
			gotBody = &next
		}
		if gotTitle == nil && gotBody == nil {
			return nil, nil, false, "LLM response missing title and body"
		}
		return gotTitle, gotBody, true, ""
	}
	return nil, &stripped, true, ""
}

func extractTriageSpecJSON(raw string) (map[string]any, bool) {
	stripped := stripTriageSpecFence(raw)
	first := strings.Index(stripped, "{")
	last := strings.LastIndex(stripped, "}")
	if first < 0 || last <= first {
		return nil, false
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(stripped[first:last+1]), &parsed); err != nil {
		return nil, false
	}
	return parsed, true
}

func stripTriageSpecFence(raw string) string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) >= 2 && strings.HasPrefix(strings.TrimSpace(lines[0]), "```") && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
		return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
	}
	return strings.TrimSpace(raw)
}
