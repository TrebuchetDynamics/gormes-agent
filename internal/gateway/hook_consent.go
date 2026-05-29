package gateway

import (
	gatewayapproval "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/approval"
	gatewayhookpolicy "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/hookpolicy"
)

type hookAutoAcceptEvidence = gatewayhookpolicy.AutoAcceptEvidence

const (
	hookAutoAcceptAcceptedByCLI    hookAutoAcceptEvidence = gatewayhookpolicy.AutoAcceptAcceptedByCLI
	hookAutoAcceptAcceptedByEnv    hookAutoAcceptEvidence = gatewayhookpolicy.AutoAcceptAcceptedByEnv
	hookAutoAcceptAcceptedByConfig hookAutoAcceptEvidence = gatewayhookpolicy.AutoAcceptAcceptedByConfig
	hookAutoAcceptRejectedDefault  hookAutoAcceptEvidence = gatewayhookpolicy.AutoAcceptRejectedDefault
	hookAutoAcceptInvalid          hookAutoAcceptEvidence = gatewayhookpolicy.AutoAcceptInvalid
)

type hookAutoAcceptInputs = gatewayhookpolicy.AutoAcceptInputs

type hookAutoAcceptDecision = gatewayhookpolicy.AutoAcceptDecision

// ApprovalChoice is the bounded decision a messaging-platform approval button
// may resolve for a pending gateway approval request.
type ApprovalChoice = gatewayapproval.Choice

const (
	ApprovalChoiceOnce    ApprovalChoice = gatewayapproval.ChoiceOnce
	ApprovalChoiceSession ApprovalChoice = gatewayapproval.ChoiceSession
	ApprovalChoiceAlways  ApprovalChoice = gatewayapproval.ChoiceAlways
	ApprovalChoiceDeny    ApprovalChoice = gatewayapproval.ChoiceDeny
)

// ApprovalResolution is the redacted evidence passed from channel callbacks
// into the gateway approval store/resolver.
type ApprovalResolution = gatewayapproval.Resolution

// ApprovalResolver owns the gateway-side approval state for pending dangerous
// operations. Channel implementations call it after a user chooses a bounded
// approval action.
type ApprovalResolver = gatewayapproval.Resolver

// ApprovalResolverFunc adapts a function to ApprovalResolver.
type ApprovalResolverFunc = gatewayapproval.ResolverFunc

// ParseApprovalChoice normalizes a gateway approval decision label.
func ParseApprovalChoice(value string) (ApprovalChoice, bool) {
	return gatewayapproval.ParseChoice(value)
}

func resolveHookAutoAccept(inputs hookAutoAcceptInputs) hookAutoAcceptDecision {
	return gatewayhookpolicy.ResolveAutoAccept(inputs)
}
