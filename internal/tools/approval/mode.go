package approval

import (
	"context"

	approvalmode "github.com/TrebuchetDynamics/gormes-agent/internal/tools/approval/mode"
)

const (
	ApprovalModeManual = approvalmode.ApprovalModeManual
	ApprovalModeSmart  = approvalmode.ApprovalModeSmart
	ApprovalModeOff    = approvalmode.ApprovalModeOff

	CronApprovalModeDeny    = approvalmode.CronApprovalModeDeny
	CronApprovalModeApprove = approvalmode.CronApprovalModeApprove
)

type ApprovalMode = approvalmode.ApprovalMode

func WithCronApprovalMode(ctx context.Context, value any) context.Context {
	return approvalmode.WithCronApprovalMode(ctx, value)
}

func CronApprovalModeFromContext(ctx context.Context) (string, bool) {
	return approvalmode.CronApprovalModeFromContext(ctx)
}

func NormalizeApprovalMode(value any) ApprovalMode {
	return approvalmode.NormalizeApprovalMode(value)
}

func NormalizeCronApprovalMode(value any) ApprovalMode {
	return approvalmode.NormalizeCronApprovalMode(value)
}
