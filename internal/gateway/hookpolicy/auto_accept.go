package hookpolicy

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/hookpolicy/autoaccept"

// AutoAcceptEvidence identifies why hook auto-accept was allowed or rejected.
type AutoAcceptEvidence = autoaccept.AutoAcceptEvidence

const (
	AutoAcceptAcceptedByCLI    = autoaccept.AutoAcceptAcceptedByCLI
	AutoAcceptAcceptedByEnv    = autoaccept.AutoAcceptAcceptedByEnv
	AutoAcceptAcceptedByConfig = autoaccept.AutoAcceptAcceptedByConfig
	AutoAcceptRejectedDefault  = autoaccept.AutoAcceptRejectedDefault
	AutoAcceptInvalid          = autoaccept.AutoAcceptInvalid
)

// AutoAcceptInputs are the precedence-ordered inputs for hook auto-accept.
type AutoAcceptInputs = autoaccept.AutoAcceptInputs

// AutoAcceptDecision is the normalized hook auto-accept verdict.
type AutoAcceptDecision = autoaccept.AutoAcceptDecision

// ResolveAutoAccept applies CLI > env > config precedence for hook auto-accept.
func ResolveAutoAccept(inputs AutoAcceptInputs) AutoAcceptDecision {
	return autoaccept.ResolveAutoAccept(inputs)
}
