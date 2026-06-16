package approval

import approvalqueue "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/approval/queue"

var (
	ErrGatewayApprovalEmptySession  = approvalqueue.ErrGatewayApprovalEmptySession
	ErrGatewayApprovalInvalidChoice = approvalqueue.ErrGatewayApprovalInvalidChoice
	ErrGatewayApprovalNotPending    = approvalqueue.ErrGatewayApprovalNotPending
)

// GatewayApprovalRequest is the dangerous operation metadata queued while a
// gateway user chooses once/session/always/deny.
type GatewayApprovalRequest = approvalqueue.GatewayApprovalRequest

// GatewayApprovalTicket identifies one queued approval request.
type GatewayApprovalTicket = approvalqueue.GatewayApprovalTicket

// GatewayApprovalOutcome records the decision applied to a queued request.
type GatewayApprovalOutcome = approvalqueue.GatewayApprovalOutcome

// GatewayApprovalQueue stores pending approval requests per gateway session.
// It mirrors Hermes' FIFO gateway approval queue without doing any channel IO.
type GatewayApprovalQueue = approvalqueue.GatewayApprovalQueue

func NewGatewayApprovalQueue() *GatewayApprovalQueue {
	return approvalqueue.NewGatewayApprovalQueue()
}
