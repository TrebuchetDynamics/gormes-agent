package kanban

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpecifyTriageTaskJSONResponsePromotesAndAudits(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:  "rough idea",
		Body:   "needs shape",
		Triage: true,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if task.Status != StatusTriage {
		t.Fatalf("triage task status = %q, want %q", task.Status, StatusTriage)
	}

	specifier := &recordingTriageSpecifier{
		response: `{"title":"Write focused spec","body":"**Goal** - ship it\n**Acceptance criteria** - tests pass"}`,
	}
	outcome, err := SpecifyTriageTask(ctx, store, task.ID, specifier, SpecifyOptions{Author: "specifier"})
	if err != nil {
		t.Fatalf("SpecifyTriageTask() error = %v", err)
	}
	if !outcome.OK || outcome.Reason != "specified" || outcome.NewTitle != "Write focused spec" {
		t.Fatalf("outcome = %+v, want successful retitle", outcome)
	}
	if specifier.request.TaskID != task.ID || specifier.request.Title != "rough idea" || specifier.request.Body != "needs shape" {
		t.Fatalf("specifier request = %+v, want original task title/body", specifier.request)
	}

	got, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.Status != StatusReady {
		t.Fatalf("specified task status = %q, want ready after parent-free promotion", got.Status)
	}
	if got.Title != "Write focused spec" || !strings.Contains(got.Body, "**Goal**") {
		t.Fatalf("specified task = %+v, want title/body from LLM response", got)
	}

	comments, err := store.ListComments(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListComments() error = %v", err)
	}
	if len(comments) != 1 || comments[0].Author != "specifier" || !strings.Contains(comments[0].Body, "updated title, body") {
		t.Fatalf("comments = %+v, want audit comment", comments)
	}
	events, err := store.ListEvents(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if !hasEventKind(events, "specified") {
		t.Fatalf("events = %+v, want specified event", events)
	}
}

func TestSpecifyTriageTaskPlainTextFallbackMatchesHermes(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	task, err := store.CreateTask(ctx, CreateTaskInput{Title: "one liner", Triage: true})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	specifier := &recordingTriageSpecifier{response: "A concrete body from a non-JSON model response."}

	outcome, err := SpecifyTriageTask(ctx, store, task.ID, specifier, SpecifyOptions{Author: "specifier"})
	if err != nil {
		t.Fatalf("SpecifyTriageTask() error = %v", err)
	}
	if !outcome.OK || outcome.NewTitle != "" {
		t.Fatalf("outcome = %+v, want success without retitle", outcome)
	}
	got, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.Title != "one liner" || got.Body != "A concrete body from a non-JSON model response." {
		t.Fatalf("task = %+v, want unchanged title and raw body fallback", got)
	}
}

func TestSpecifyTriageTaskDegradedDoesNotMutate(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	ready, err := store.CreateTask(ctx, CreateTaskInput{Title: "already ready"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	specifier := &recordingTriageSpecifier{response: `{"title":"mutated","body":"mutated"}`}
	outcome, err := SpecifyTriageTask(ctx, store, ready.ID, specifier, SpecifyOptions{Author: "specifier"})
	if err != nil {
		t.Fatalf("SpecifyTriageTask(non-triage) error = %v", err)
	}
	if outcome.OK || !strings.Contains(outcome.Reason, "not in triage") {
		t.Fatalf("outcome = %+v, want non-triage degraded evidence", outcome)
	}
	if specifier.called {
		t.Fatal("specifier was called for a non-triage task")
	}

	got, err := store.GetTask(ctx, ready.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.Title != "already ready" || got.Status != StatusReady {
		t.Fatalf("non-triage task mutated: %+v", got)
	}

	triage, err := store.CreateTask(ctx, CreateTaskInput{Title: "empty response", Triage: true})
	if err != nil {
		t.Fatalf("CreateTask(triage) error = %v", err)
	}
	empty := &recordingTriageSpecifier{response: "   "}
	outcome, err = SpecifyTriageTask(ctx, store, triage.ID, empty, SpecifyOptions{Author: "specifier"})
	if err != nil {
		t.Fatalf("SpecifyTriageTask(empty) error = %v", err)
	}
	if outcome.OK || outcome.Reason != "LLM returned an empty response" {
		t.Fatalf("empty outcome = %+v, want empty-response degraded evidence", outcome)
	}
	stillTriage, err := store.GetTask(ctx, triage.ID)
	if err != nil {
		t.Fatalf("GetTask(triage) error = %v", err)
	}
	if stillTriage.Status != StatusTriage || stillTriage.Body != "" {
		t.Fatalf("empty-response task mutated: %+v", stillTriage)
	}

	outcome, err = SpecifyTriageTask(ctx, store, triage.ID, erroringTriageSpecifier{}, SpecifyOptions{Author: "specifier"})
	if err != nil {
		t.Fatalf("SpecifyTriageTask(erroring) error = %v", err)
	}
	if outcome.OK || outcome.Reason != "LLM error: triage specifier failed" {
		t.Fatalf("error outcome = %+v, want redacted LLM failure", outcome)
	}
}

type recordingTriageSpecifier struct {
	response string
	request  TriageSpecRequest
	called   bool
}

func (s *recordingTriageSpecifier) CompleteTriageSpec(_ context.Context, req TriageSpecRequest) (string, error) {
	s.called = true
	s.request = req
	return s.response, nil
}

type erroringTriageSpecifier struct{}

func (erroringTriageSpecifier) CompleteTriageSpec(context.Context, TriageSpecRequest) (string, error) {
	return "", errors.New("provider returned secret body")
}

func hasEventKind(events []Event, kind string) bool {
	for _, event := range events {
		if event.Kind == kind {
			return true
		}
	}
	return false
}
