package branch

import (
	"context"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestTitleFromInput(t *testing.T) {
	if got := TitleFromInput("/branch refactor path"); got != "refactor path" {
		t.Fatalf("TitleFromInput = %q", got)
	}
	if got := TitleFromInput("/branch"); got != "" {
		t.Fatalf("TitleFromInput no title = %q", got)
	}
}

func TestSuccessStatus(t *testing.T) {
	if got := SuccessStatus(Result{SessionID: "child", TranscriptCopied: 2}); got != "branch: switched to child (2 turns)" {
		t.Fatalf("SuccessStatus without title = %q", got)
	}
	if got := SuccessStatus(Result{SessionID: "child", Title: "demo", TranscriptCopied: 3}); got != `branch: switched to child ("demo", 3 turns)` {
		t.Fatalf("SuccessStatus with title = %q", got)
	}
}

func TestHandleSlash(t *testing.T) {
	if res := HandleSlash("/branch", false, "parent", 0, nil, nil); res.Switch || res.Status != "branch: no conversation" {
		t.Fatalf("HandleSlash(no conversation) = %+v", res)
	}
	if res := HandleSlash("/branch", true, "parent", 1, nil, nil); res.Switch || res.Status != "branch: store unavailable" {
		t.Fatalf("HandleSlash(no fork) = %+v", res)
	}
	if res := HandleSlash("/branch", true, " ", 1, nil, func(context.Context, Request) (Result, error) { return Result{}, nil }); res.Switch || res.Status != "branch: no active session" {
		t.Fatalf("HandleSlash(no session) = %+v", res)
	}

	history := []llm.Message{{Role: "user", Content: "hello"}}
	res := HandleSlash("/branch demo", true, " parent ", 1, history, func(ctx context.Context, req Request) (Result, error) {
		if ctx == nil {
			t.Fatal("fork context is nil")
		}
		if req.ParentSessionID != "parent" || req.Title != "demo" || req.HistoryCount != 1 || len(req.History) != 1 {
			t.Fatalf("fork request = %+v", req)
		}
		return Result{SessionID: "child", Title: req.Title, TranscriptCopied: 1}, nil
	})
	if !res.Switch || res.Branch.SessionID != "child" || res.Status != `branch: switched to child ("demo", 1 turns)` {
		t.Fatalf("HandleSlash(success) = %+v", res)
	}
	if res := HandleSlash("/branch", true, "parent", 1, history, func(context.Context, Request) (Result, error) { return Result{}, errors.New("locked") }); res.Switch || res.Status != "branch: fork failed: locked" {
		t.Fatalf("HandleSlash(error) = %+v", res)
	}
}
