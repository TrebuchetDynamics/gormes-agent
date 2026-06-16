package approval

import (
	approvalux "github.com/TrebuchetDynamics/gormes-agent/internal/tools/approval/ux"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety"
)

type UXDecision = approvalux.UXDecision

const (
	UXDecisionYes     = approvalux.UXDecisionYes
	UXDecisionNo      = approvalux.UXDecisionNo
	UXDecisionAlways  = approvalux.UXDecisionAlways
	UXDecisionUnknown = approvalux.UXDecisionUnknown
)

type ApprovalPrompt = approvalux.ApprovalPrompt
type ApprovalUXResult = approvalux.ApprovalUXResult
type ApprovalAuditRecord = approvalux.ApprovalAuditRecord
type ApprovalSession = approvalux.ApprovalSession

func NewApprovalSession(interactive bool) *ApprovalSession {
	return approvalux.NewApprovalSession(interactive)
}

func BuildApprovalPrompt(command string, category safety.BlocklistCategory) ApprovalPrompt {
	return approvalux.BuildApprovalPrompt(command, category)
}
