package tools

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/approval"
)

// ApprovalRequest describes a dangerous operation that needs an explicit
// operator approval decision before a worker may continue.
type ApprovalRequest = approval.ApprovalRequest

// ApprovalDecision is the callback's bounded decision. Approved=false denies.
type ApprovalDecision = approval.ApprovalDecision

// ApprovalCallback is injected by an interactive parent when a concurrent
// worker is allowed to ask the same explicit approval policy. Absence means
// noninteractive fail-closed; callers must not fall back to stdin.
type ApprovalCallback = approval.ApprovalCallback

// WithApprovalCallback returns a child context carrying cb. Passing nil leaves
// ctx unchanged so callback cleanup is just normal context scoping.
func WithApprovalCallback(ctx context.Context, cb ApprovalCallback) context.Context {
	return approval.WithApprovalCallback(ctx, cb)
}

// ApprovalCallbackFromContext returns the explicitly injected approval callback.
func ApprovalCallbackFromContext(ctx context.Context) (ApprovalCallback, bool) {
	return approval.ApprovalCallbackFromContext(ctx)
}

// GuardCommandWithApproval applies GuardCommand and, for recoverable dangerous
// commands only, consults an explicit context callback. Missing callbacks and
// callback failures deny without interactive stdin fallback.
func GuardCommandWithApproval(ctx context.Context, toolName, cmd, mode string) BlockedResult {
	return approval.GuardCommandWithApproval(ctx, toolName, cmd, mode)
}
