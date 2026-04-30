package tools

import (
	"context"
	"errors"
	"testing"
)

func TestApprovalCallbackPropagation_ConcurrentWorkerUsesInjectedCallback(t *testing.T) {
	called := 0
	ctx := WithApprovalCallback(context.Background(), func(ctx context.Context, req ApprovalRequest) (ApprovalDecision, error) {
		called++
		if req.Command != "git reset --hard" {
			t.Fatalf("approval command = %q, want git reset --hard", req.Command)
		}
		if req.ToolName != "terminal" {
			t.Fatalf("approval tool = %q, want terminal", req.ToolName)
		}
		return ApprovalDecision{Approved: true, Evidence: map[string]string{"approval_source": "fixture"}}, nil
	})

	result := GuardCommandWithApproval(ctx, "terminal", "git reset --hard", "manual")
	if called != 1 {
		t.Fatalf("approval callback calls = %d, want 1", called)
	}
	if !result.Approved || result.Description == "" {
		t.Fatalf("GuardCommandWithApproval result = %#v, want approved dangerous command evidence", result)
	}
	if got := result.Evidence["approval_source"]; got != "fixture" {
		t.Fatalf("approval_source evidence = %q, want fixture", got)
	}
}

func TestApprovalCallbackPropagation_MissingCallbackFailsClosed(t *testing.T) {
	result := GuardCommandWithApproval(context.Background(), "terminal", "git reset --hard", "manual")
	if result.Approved {
		t.Fatalf("approved = true, want fail-closed denial: %#v", result)
	}
	if !result.ApprovalRequired {
		t.Fatalf("approval required = false, want true: %#v", result)
	}
	if got := result.Evidence["reason"]; got != "approval_callback_missing" {
		t.Fatalf("reason evidence = %q, want approval_callback_missing", got)
	}
}

func TestApprovalCallbackPropagation_CallbackClearedAfterWorker(t *testing.T) {
	parent := context.Background()
	worker := WithApprovalCallback(parent, func(context.Context, ApprovalRequest) (ApprovalDecision, error) {
		return ApprovalDecision{Approved: true}, nil
	})
	if _, ok := ApprovalCallbackFromContext(worker); !ok {
		t.Fatal("worker context missing approval callback")
	}
	if _, ok := ApprovalCallbackFromContext(parent); ok {
		t.Fatal("parent context unexpectedly retained worker approval callback")
	}
}

func TestApprovalCallbackPropagation_CallbackErrorDenied(t *testing.T) {
	ctx := WithApprovalCallback(context.Background(), func(context.Context, ApprovalRequest) (ApprovalDecision, error) {
		return ApprovalDecision{}, errors.New("fixture approval unavailable")
	})
	result := GuardCommandWithApproval(ctx, "terminal", "git reset --hard", "manual")
	if result.Approved {
		t.Fatalf("approved = true, want callback error denial: %#v", result)
	}
	if got := result.Evidence["reason"]; got != "approval_callback_error" {
		t.Fatalf("reason evidence = %q, want approval_callback_error", got)
	}
	if got := result.Evidence["error"]; got != "fixture approval unavailable" {
		t.Fatalf("error evidence = %q, want sanitized callback error", got)
	}
}
