package tools

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/approval"
)

const (
	ApprovalModeManual = approval.ApprovalModeManual
	ApprovalModeSmart  = approval.ApprovalModeSmart
	ApprovalModeOff    = approval.ApprovalModeOff

	CronApprovalModeDeny    = approval.CronApprovalModeDeny
	CronApprovalModeApprove = approval.CronApprovalModeApprove
)

// ApprovalMode describes a normalized dangerous-command approval mode plus
// redacted evidence about whether the value had to fall back to the safe
// manual default.
type ApprovalMode = approval.ApprovalMode

// WithCronApprovalMode marks a tool execution context as a noninteractive
// cron turn governed by approvals.cron_mode instead of the interactive
// terminal approval path.
func WithCronApprovalMode(ctx context.Context, value any) context.Context {
	return approval.WithCronApprovalMode(ctx, value)
}

// CronApprovalModeFromContext returns the normalized cron approval mode when
// the caller explicitly scoped this tool execution to a cron turn.
func CronApprovalModeFromContext(ctx context.Context) (string, bool) {
	return approval.CronApprovalModeFromContext(ctx)
}

// NormalizeApprovalMode ports Hermes' approval mode parsing semantics for
// config-loaded values.
func NormalizeApprovalMode(value any) ApprovalMode {
	return approval.NormalizeApprovalMode(value)
}

// NormalizeCronApprovalMode ports Hermes' approvals.cron_mode parser.
func NormalizeCronApprovalMode(value any) ApprovalMode {
	return approval.NormalizeCronApprovalMode(value)
}
