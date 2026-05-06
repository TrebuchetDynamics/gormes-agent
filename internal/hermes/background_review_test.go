package hermes

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestBackgroundReviewFork_InheritsActiveRuntime(t *testing.T) {
	credentialPool := &struct{ name string }{"pool"}
	memoryStore := &struct{ name string }{"memory"}
	parent := BackgroundReviewRuntime{
		Model:              "gpt-5.5",
		Provider:           "openai-codex",
		APIMode:            "responses",
		BaseURL:            "https://api.example.test/v1/responses",
		APIKey:             "secret-token",
		CredentialPool:     credentialPool,
		Platform:           "telegram",
		ParentSessionID:    "sess-parent",
		MemoryStore:        memoryStore,
		MemoryEnabled:      true,
		UserProfileEnabled: true,
	}

	var captured BackgroundReviewFork
	runner := BackgroundReviewRunnerFunc(func(_ context.Context, fork BackgroundReviewFork) ([]BackgroundReviewMessage, error) {
		captured = fork
		return nil, nil
	})

	_, err := RunBackgroundReview(context.Background(), BackgroundReviewRequest{
		Runtime:      parent,
		ReviewMemory: true,
		Runner:       runner,
	})
	if err != nil {
		t.Fatalf("RunBackgroundReview() error = %v", err)
	}

	if !reflect.DeepEqual(captured.Runtime, parent) {
		t.Fatalf("captured runtime = %#v, want inherited %#v", captured.Runtime, parent)
	}
	if captured.Runtime.CredentialPool != credentialPool {
		t.Fatal("credential pool was not inherited by identity")
	}
	if captured.Runtime.MemoryStore != memoryStore {
		t.Fatal("memory store was not inherited by identity")
	}
	if captured.Runtime.APIKey != "secret-token" {
		t.Fatal("api key was not inherited from the active runtime")
	}
}

func TestBackgroundReviewFork_RestrictsTools(t *testing.T) {
	var captured BackgroundReviewFork
	runner := BackgroundReviewRunnerFunc(func(_ context.Context, fork BackgroundReviewFork) ([]BackgroundReviewMessage, error) {
		captured = fork
		return nil, nil
	})

	_, err := RunBackgroundReview(context.Background(), BackgroundReviewRequest{
		Runtime:      BackgroundReviewRuntime{Model: "gpt-5.5"},
		ReviewSkills: true,
		Runner:       runner,
	})
	if err != nil {
		t.Fatalf("RunBackgroundReview() error = %v", err)
	}

	if got, want := captured.EnabledToolsets, []string{"memory", "skills"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("EnabledToolsets = %v, want %v", got, want)
	}
	for _, denied := range []string{"terminal", "browser", "execute_code", "network", "file_write", "provider_management"} {
		ev, ok := captured.ToolsetPolicy.CheckToolset(denied)
		if ok {
			t.Fatalf("toolset %q allowed; evidence=%+v", denied, ev)
		}
		if ev.Status == "" || ev.DeniedToolset != denied {
			t.Fatalf("toolset %q evidence = %+v, want denied evidence", denied, ev)
		}
	}
	if captured.SkillManagerWriteOrigin != "background_review" {
		t.Fatalf("SkillManagerWriteOrigin = %q, want background_review", captured.SkillManagerWriteOrigin)
	}
}

func TestBackgroundReviewFork_SkipsPriorToolMessages(t *testing.T) {
	priorPayload := map[string]any{"success": true, "message": "Cron job 'old' created."}
	newPayload := map[string]any{"success": true, "message": "Entry added", "target": "memory"}
	prior := BackgroundReviewMessage{Role: "tool", ToolCallID: "call-old", Content: mustJSON(t, priorPayload)}
	reviewMessages := []BackgroundReviewMessage{
		prior,
		{Role: "tool", ToolCallID: "call-new", Content: mustJSON(t, newPayload)},
	}

	actions := SummarizeBackgroundReviewActions(reviewMessages, []BackgroundReviewMessage{prior})

	if containsString(actions, "Cron job 'old' created.") {
		t.Fatalf("actions resurfaced stale prior tool result: %v", actions)
	}
	if !containsString(actions, "Memory updated") {
		t.Fatalf("actions = %v, want Memory updated", actions)
	}
}

func TestBackgroundReviewFork_EmitsAttributedSummary(t *testing.T) {
	var summaries []string
	runner := BackgroundReviewRunnerFunc(func(_ context.Context, _ BackgroundReviewFork) ([]BackgroundReviewMessage, error) {
		return []BackgroundReviewMessage{{
			Role:       "tool",
			ToolCallID: "call-bg",
			Content:    mustJSON(t, map[string]any{"success": true, "message": "Entry added", "target": "memory"}),
		}}, nil
	})

	result, err := RunBackgroundReview(context.Background(), BackgroundReviewRequest{
		Runtime:         BackgroundReviewRuntime{Model: "gpt-5.5"},
		Messages:        []BackgroundReviewMessage{{Role: "user", Content: "hello"}},
		ReviewMemory:    true,
		Runner:          runner,
		SummaryCallback: func(summary string) { summaries = append(summaries, summary) },
	})
	if err != nil {
		t.Fatalf("RunBackgroundReview() error = %v", err)
	}

	if got, want := len(summaries), 1; got != want {
		t.Fatalf("summary callbacks = %d, want %d (%v)", got, want, summaries)
	}
	if !strings.Contains(summaries[0], "Self-improvement review: Memory updated") {
		t.Fatalf("summary = %q, want attributed self-improvement summary", summaries[0])
	}
	if result.Summary != summaries[0] || !reflect.DeepEqual(result.Actions, []string{"Memory updated"}) {
		t.Fatalf("result = %+v, want matching summary/actions", result)
	}
}

func TestBackgroundReviewFork_Cleanup(t *testing.T) {
	var events []string
	var duringApproval BackgroundReviewApprovalCallback
	approvalSlot := BackgroundReviewApprovalSlot{
		Set: func(cb BackgroundReviewApprovalCallback) {
			events = append(events, "approval_set")
			duringApproval = cb
		},
		Clear: func() {
			events = append(events, "approval_clear")
			duringApproval = nil
		},
	}
	runner := BackgroundReviewRunnerFunc(func(_ context.Context, _ BackgroundReviewFork) ([]BackgroundReviewMessage, error) {
		events = append(events, "run")
		if duringApproval == nil {
			t.Fatal("approval callback was not installed during background review")
		}
		if got := duringApproval("rm -rf /", "dangerous"); got != "deny" {
			t.Fatalf("approval callback returned %q, want deny", got)
		}
		return nil, nil
	})

	_, err := RunBackgroundReview(context.Background(), BackgroundReviewRequest{
		Runtime:      BackgroundReviewRuntime{Model: "gpt-5.5"},
		ReviewMemory: true,
		Runner:       runner,
		ApprovalSlot: approvalSlot,
		ShutdownMemoryProvider: func() {
			events = append(events, "shutdown_memory_provider")
		},
		Close: func() {
			events = append(events, "close")
		},
	})
	if err != nil {
		t.Fatalf("RunBackgroundReview() error = %v", err)
	}

	want := []string{"approval_set", "run", "shutdown_memory_provider", "close", "approval_clear"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	if duringApproval != nil {
		t.Fatal("approval callback leaked after background review cleanup")
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(raw)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
