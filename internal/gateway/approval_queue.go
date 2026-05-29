package gateway

import gatewayapproval "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/approval"

var (
	ErrGatewayApprovalEmptySession  = gatewayapproval.ErrGatewayApprovalEmptySession
	ErrGatewayApprovalInvalidChoice = gatewayapproval.ErrGatewayApprovalInvalidChoice
	ErrGatewayApprovalNotPending    = gatewayapproval.ErrGatewayApprovalNotPending
)

// GatewayApprovalRequest is the dangerous operation metadata queued while a
// gateway user chooses once/session/always/deny.
type GatewayApprovalRequest = gatewayapproval.GatewayApprovalRequest

// GatewayApprovalTicket identifies one queued approval request.
type GatewayApprovalTicket = gatewayapproval.GatewayApprovalTicket

// GatewayApprovalOutcome records the decision applied to a queued request.
type GatewayApprovalOutcome = gatewayapproval.GatewayApprovalOutcome

// GatewayApprovalQueue stores pending approval requests per gateway session.
type GatewayApprovalQueue = gatewayapproval.GatewayApprovalQueue

func NewGatewayApprovalQueue() *GatewayApprovalQueue {
	return gatewayapproval.NewGatewayApprovalQueue()
}
