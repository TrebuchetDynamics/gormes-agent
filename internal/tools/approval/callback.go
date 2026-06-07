package approval

import (
	"context"

	approvalcallback "github.com/TrebuchetDynamics/gormes-agent/internal/tools/approval/callback"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety"
)

type ApprovalRequest = approvalcallback.ApprovalRequest
type ApprovalDecision = approvalcallback.ApprovalDecision
type ApprovalCallback = approvalcallback.ApprovalCallback

func WithApprovalCallback(ctx context.Context, cb ApprovalCallback) context.Context {
	return approvalcallback.WithApprovalCallback(ctx, cb)
}

func ApprovalCallbackFromContext(ctx context.Context) (ApprovalCallback, bool) {
	return approvalcallback.ApprovalCallbackFromContext(ctx)
}

func GuardCommandWithApproval(ctx context.Context, toolName, cmd, mode string) safety.BlockedResult {
	return approvalcallback.GuardCommandWithApproval(ctx, toolName, cmd, mode)
}
