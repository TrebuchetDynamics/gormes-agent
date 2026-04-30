package tools

import "context"

// ApprovalRequest describes a dangerous operation that needs an explicit
// operator approval decision before a worker may continue.
type ApprovalRequest struct {
	ToolName    string
	Command     string
	Mode        string
	Description string
	Evidence    map[string]string
}

// ApprovalDecision is the callback's bounded decision. Approved=false denies.
type ApprovalDecision struct {
	Approved bool
	Reason   string
	Evidence map[string]string
}

// ApprovalCallback is injected by an interactive parent when a concurrent
// worker is allowed to ask the same explicit approval policy. Absence means
// noninteractive fail-closed; callers must not fall back to stdin.
type ApprovalCallback func(context.Context, ApprovalRequest) (ApprovalDecision, error)

type approvalCallbackContextKey struct{}

// WithApprovalCallback returns a child context carrying cb. Passing nil leaves
// ctx unchanged so callback cleanup is just normal context scoping.
func WithApprovalCallback(ctx context.Context, cb ApprovalCallback) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if cb == nil {
		return ctx
	}
	return context.WithValue(ctx, approvalCallbackContextKey{}, cb)
}

// ApprovalCallbackFromContext returns the explicitly injected approval callback.
func ApprovalCallbackFromContext(ctx context.Context) (ApprovalCallback, bool) {
	if ctx == nil {
		return nil, false
	}
	cb, ok := ctx.Value(approvalCallbackContextKey{}).(ApprovalCallback)
	return cb, ok && cb != nil
}

// GuardCommandWithApproval applies GuardCommand and, for recoverable dangerous
// commands only, consults an explicit context callback. Missing callbacks and
// callback failures deny without interactive stdin fallback.
func GuardCommandWithApproval(ctx context.Context, toolName, cmd, mode string) BlockedResult {
	result := GuardCommand(cmd, mode)
	if result.Description == "" || result.Approved || result.Hardline || !result.ApprovalRequired {
		return result
	}
	cb, ok := ApprovalCallbackFromContext(ctx)
	if !ok {
		return approvalCallbackDenied(result, "approval_callback_missing", "")
	}
	decision, err := cb(ctx, ApprovalRequest{
		ToolName:    toolName,
		Command:     cmd,
		Mode:        result.Operator,
		Description: result.Description,
		Evidence:    cloneStringMap(result.Evidence),
	})
	if err != nil {
		return approvalCallbackDenied(result, "approval_callback_error", err.Error())
	}
	if !decision.Approved {
		reason := decision.Reason
		if reason == "" {
			reason = "approval_callback_denied"
		}
		denied := approvalCallbackDenied(result, reason, "")
		for k, v := range decision.Evidence {
			denied.Evidence[k] = v
		}
		return denied
	}
	result.Approved = true
	result.ApprovalRequired = false
	result.Evidence["approval_callback"] = "approved"
	for k, v := range decision.Evidence {
		result.Evidence[k] = v
	}
	return result
}

func approvalCallbackDenied(result BlockedResult, reason, errText string) BlockedResult {
	result.Approved = false
	result.ApprovalRequired = true
	if result.Evidence == nil {
		result.Evidence = map[string]string{}
	}
	result.Evidence["reason"] = reason
	result.Evidence["approval_callback"] = "denied"
	if errText != "" {
		result.Evidence["error"] = errText
	}
	return result
}
