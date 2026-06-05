package acp

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

func TestACPJSONRPCPromptCarriesSessionCWD(t *testing.T) {
	var got []string
	runtime := NewSessionRuntime(SessionRuntimeConfig{
		IDGenerator: func() string {
			return "cwd-session"
		},
		Runner: PromptRunnerFunc(func(ctx context.Context, req RuntimePromptRequest, emit func(PromptEvent)) (PromptResult, error) {
			got = append(got, req.CWD)
			return PromptResult{Final: "ok", StopReason: "end_turn"}, nil
		}),
	})
	sess, err := runtime.NewSession(context.Background(), "C:\\Workspace\\Paperclip")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	if _, err := runtime.Prompt(context.Background(), RuntimePromptRequest{
		SessionID: sess.ID,
		Text:      "pwd",
	}, func(PromptEvent) {}); err != nil {
		t.Fatalf("Prompt: %v", err)
	}

	if !reflect.DeepEqual(got, []string{"/mnt/c/Workspace/Paperclip"}) {
		t.Fatalf("runner CWDs = %v, want translated ACP cwd", got)
	}
}

func TestACPJSONRPCQueuedPromptCarriesSessionCWD(t *testing.T) {
	var got []string
	runtime := NewSessionRuntime(SessionRuntimeConfig{
		IDGenerator: func() string {
			return "queue-cwd-session"
		},
		Runner: PromptRunnerFunc(func(ctx context.Context, req RuntimePromptRequest, emit func(PromptEvent)) (PromptResult, error) {
			got = append(got, req.Text+"@"+req.CWD)
			return PromptResult{Final: "ran:" + req.Text, StopReason: "end_turn"}, nil
		}),
	})
	sess, err := runtime.NewSession(context.Background(), "/repo/acp")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	if _, err := runtime.Prompt(context.Background(), RuntimePromptRequest{
		SessionID: sess.ID,
		Text:      "/queue second",
	}, func(PromptEvent) {}); err != nil {
		t.Fatalf("queue Prompt: %v", err)
	}
	if _, err := runtime.Prompt(context.Background(), RuntimePromptRequest{
		SessionID: sess.ID,
		Text:      "first",
	}, func(PromptEvent) {}); err != nil {
		t.Fatalf("first Prompt: %v", err)
	}

	want := []string{"first@/repo/acp", "second@/repo/acp"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runner prompts/CWDs = %v, want %v", got, want)
	}
}

func TestACPLoadResumeUpdatesPromptCWD(t *testing.T) {
	var got []string
	sessions := session.NewMemMap()
	runtime := NewSessionRuntime(SessionRuntimeConfig{
		SessionMap: sessions,
		IDGenerator: func() string {
			return "load-cwd-session"
		},
		Runner: PromptRunnerFunc(func(ctx context.Context, req RuntimePromptRequest, emit func(PromptEvent)) (PromptResult, error) {
			got = append(got, req.Text+"@"+req.CWD)
			return PromptResult{Final: "ok", StopReason: "end_turn"}, nil
		}),
	})
	sess, err := runtime.NewSession(context.Background(), "/initial")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if _, err := runtime.LoadSession(context.Background(), sess.ID, "/loaded"); err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if _, err := runtime.Prompt(context.Background(), RuntimePromptRequest{
		SessionID: sess.ID,
		Text:      "after load",
	}, func(PromptEvent) {}); err != nil {
		t.Fatalf("load Prompt: %v", err)
	}
	if _, err := runtime.ResumeSession(context.Background(), sess.ID, "D:\\Resume\\Repo"); err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}
	if _, err := runtime.Prompt(context.Background(), RuntimePromptRequest{
		SessionID: sess.ID,
		Text:      "after resume",
	}, func(PromptEvent) {}); err != nil {
		t.Fatalf("resume Prompt: %v", err)
	}

	want := []string{"after load@/loaded", "after resume@/mnt/d/Resume/Repo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runner prompts/CWDs = %v, want %v", got, want)
	}
}

func TestACPJSONRPCSlashSteerAndQueue(t *testing.T) {
	var prompts []string
	runtime := NewSessionRuntime(SessionRuntimeConfig{
		IDGenerator: func() string {
			return "queue-session"
		},
		Runner: PromptRunnerFunc(func(ctx context.Context, req RuntimePromptRequest, emit func(PromptEvent)) (PromptResult, error) {
			prompts = append(prompts, req.Text)
			emit(PromptEvent{Kind: PromptEventAgentMessageChunk, Text: "ran:" + req.Text})
			return PromptResult{Final: "ran:" + req.Text, StopReason: "end_turn"}, nil
		}),
	})
	sess, err := runtime.NewSession(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	var events []PromptEvent
	if _, err := runtime.Prompt(context.Background(), RuntimePromptRequest{
		SessionID: sess.ID,
		Text:      "/queue second turn",
	}, func(event PromptEvent) { events = append(events, event) }); err != nil {
		t.Fatalf("queue Prompt: %v", err)
	}
	if len(prompts) != 0 {
		t.Fatalf("queue command ran prompt immediately: %v", prompts)
	}
	if got := eventTexts(events); !reflect.DeepEqual(got, []string{"Queued for the next turn. (1 queued)"}) {
		t.Fatalf("queue events = %v", got)
	}

	events = nil
	if _, err := runtime.Prompt(context.Background(), RuntimePromptRequest{
		SessionID: sess.ID,
		Text:      "first turn",
	}, func(event PromptEvent) { events = append(events, event) }); err != nil {
		t.Fatalf("first Prompt: %v", err)
	}
	if !reflect.DeepEqual(prompts, []string{"first turn", "second turn"}) {
		t.Fatalf("runner prompts = %v, want first then queued second", prompts)
	}
	if got := eventTexts(events); !reflect.DeepEqual(got, []string{"ran:first turn", "second turn", "ran:second turn"}) {
		t.Fatalf("drain events = %v", got)
	}

	prompts = nil
	if _, err := runtime.Prompt(context.Background(), RuntimePromptRequest{
		SessionID: sess.ID,
		Text:      "/steer adjust course",
	}, func(PromptEvent) {}); err != nil {
		t.Fatalf("steer Prompt: %v", err)
	}
	if !reflect.DeepEqual(prompts, []string{"adjust course"}) {
		t.Fatalf("idle steer prompts = %v, want adjusted prompt", prompts)
	}
}

func TestACPJSONRPCPermissionsIsolation(t *testing.T) {
	broker := NewPermissionBroker()
	broker.SetRequester("session-a", PermissionRequesterFunc(func(ctx context.Context, req PermissionRequest) (PermissionDecision, error) {
		return PermissionDecision{Outcome: PermissionAllowAlways, Reason: "trusted"}, nil
	}))
	broker.SetRequester("session-b", PermissionRequesterFunc(func(ctx context.Context, req PermissionRequest) (PermissionDecision, error) {
		return PermissionDecision{Outcome: PermissionRejectOnce, Reason: "blocked"}, nil
	}))

	first, err := broker.Request(context.Background(), "session-a", PermissionRequest{Key: "sudo", Command: "touch a"})
	if err != nil {
		t.Fatalf("session-a request: %v", err)
	}
	if first.Result != "always" || first.Reason != "trusted" {
		t.Fatalf("session-a first decision = %+v", first)
	}
	broker.SetRequester("session-a", PermissionRequesterFunc(func(ctx context.Context, req PermissionRequest) (PermissionDecision, error) {
		t.Fatal("cached allow_always should not call requester again")
		return PermissionDecision{}, nil
	}))
	cached, err := broker.Request(context.Background(), "session-a", PermissionRequest{Key: "sudo", Command: "touch a"})
	if err != nil {
		t.Fatalf("session-a cached request: %v", err)
	}
	if cached.Result != "always" || cached.Cached != true {
		t.Fatalf("session-a cached decision = %+v, want cached always", cached)
	}

	other, err := broker.Request(context.Background(), "session-b", PermissionRequest{Key: "sudo", Command: "touch b"})
	if err != nil {
		t.Fatalf("session-b request: %v", err)
	}
	if other.Result != "deny" || other.Cached {
		t.Fatalf("session-b decision = %+v, want uncached deny", other)
	}

	timeoutBroker := NewPermissionBroker()
	timeoutBroker.SetRequester("slow", PermissionRequesterFunc(func(ctx context.Context, req PermissionRequest) (PermissionDecision, error) {
		<-ctx.Done()
		return PermissionDecision{}, ctx.Err()
	}))
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	timedOut, err := timeoutBroker.Request(ctx, "slow", PermissionRequest{Key: "sudo"})
	if err != nil {
		t.Fatalf("timeout request should degrade, got error: %v", err)
	}
	if timedOut.Result != "deny" || timedOut.Reason != "permission_timeout" {
		t.Fatalf("timeout decision = %+v, want deny permission_timeout", timedOut)
	}
}

func eventTexts(events []PromptEvent) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		if event.Text != "" {
			out = append(out, event.Text)
		}
	}
	return out
}
