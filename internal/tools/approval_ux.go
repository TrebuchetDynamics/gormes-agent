package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/approval"

type UXDecision = approval.UXDecision

const (
	UXDecisionYes     = approval.UXDecisionYes
	UXDecisionNo      = approval.UXDecisionNo
	UXDecisionAlways  = approval.UXDecisionAlways
	UXDecisionUnknown = approval.UXDecisionUnknown
)

type ApprovalPrompt = approval.ApprovalPrompt
type ApprovalUXResult = approval.ApprovalUXResult
type ApprovalAuditRecord = approval.ApprovalAuditRecord
type ApprovalSession = approval.ApprovalSession

func NewApprovalSession(interactive bool) *ApprovalSession {
	return approval.NewApprovalSession(interactive)
}

func BuildApprovalPrompt(command string, category BlocklistCategory) ApprovalPrompt {
	return approval.BuildApprovalPrompt(command, category)
}
