package gateway

import (
	"context"
	"strings"
)

type hookAutoAcceptEvidence string

const (
	hookAutoAcceptAcceptedByCLI    hookAutoAcceptEvidence = "accepted_by_cli"
	hookAutoAcceptAcceptedByEnv    hookAutoAcceptEvidence = "accepted_by_env"
	hookAutoAcceptAcceptedByConfig hookAutoAcceptEvidence = "accepted_by_config"
	hookAutoAcceptRejectedDefault  hookAutoAcceptEvidence = "rejected_default"
	hookAutoAcceptInvalid          hookAutoAcceptEvidence = "hook_auto_accept_invalid"
)

type hookAutoAcceptInputs struct {
	CLIAccept   bool
	EnvValue    *string
	ConfigValue any
}

type hookAutoAcceptDecision struct {
	Accept   bool
	Evidence hookAutoAcceptEvidence
}

// ApprovalChoice is the bounded decision a messaging-platform approval button
// may resolve for a pending gateway approval request.
type ApprovalChoice string

const (
	ApprovalChoiceOnce    ApprovalChoice = "once"
	ApprovalChoiceSession ApprovalChoice = "session"
	ApprovalChoiceAlways  ApprovalChoice = "always"
	ApprovalChoiceDeny    ApprovalChoice = "deny"
)

// ApprovalResolution is the redacted evidence passed from channel callbacks
// into the gateway approval store/resolver.
type ApprovalResolution struct {
	SessionKey string
	Choice     ApprovalChoice
	Platform   string
	ChatID     string
	MessageID  string
	ActorID    string
	Evidence   map[string]string
}

// ApprovalResolver owns the gateway-side approval state for pending dangerous
// operations. Channel implementations call it after a user chooses a bounded
// approval action.
type ApprovalResolver interface {
	ResolveGatewayApproval(context.Context, ApprovalResolution) error
}

// ApprovalResolverFunc adapts a function to ApprovalResolver.
type ApprovalResolverFunc func(context.Context, ApprovalResolution) error

func (f ApprovalResolverFunc) ResolveGatewayApproval(ctx context.Context, res ApprovalResolution) error {
	if f == nil {
		return nil
	}
	return f(ctx, res)
}

// ParseApprovalChoice normalizes a gateway approval decision label.
func ParseApprovalChoice(value string) (ApprovalChoice, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(ApprovalChoiceOnce):
		return ApprovalChoiceOnce, true
	case string(ApprovalChoiceSession):
		return ApprovalChoiceSession, true
	case string(ApprovalChoiceAlways):
		return ApprovalChoiceAlways, true
	case string(ApprovalChoiceDeny):
		return ApprovalChoiceDeny, true
	default:
		return "", false
	}
}

func resolveHookAutoAccept(inputs hookAutoAcceptInputs) hookAutoAcceptDecision {
	if inputs.CLIAccept {
		return hookAutoAcceptDecision{
			Accept:   true,
			Evidence: hookAutoAcceptAcceptedByCLI,
		}
	}
	if inputs.EnvValue != nil {
		return hookAutoAcceptDecisionForValue(*inputs.EnvValue, hookAutoAcceptAcceptedByEnv)
	}
	return hookAutoAcceptDecisionForValue(inputs.ConfigValue, hookAutoAcceptAcceptedByConfig)
}

func hookAutoAcceptDecisionForValue(value any, acceptedEvidence hookAutoAcceptEvidence) hookAutoAcceptDecision {
	accepted, valid := parseHookAutoAcceptValue(value)
	if accepted {
		return hookAutoAcceptDecision{
			Accept:   true,
			Evidence: acceptedEvidence,
		}
	}
	if !valid {
		return hookAutoAcceptDecision{
			Accept:   false,
			Evidence: hookAutoAcceptInvalid,
		}
	}
	return hookAutoAcceptDecision{
		Accept:   false,
		Evidence: hookAutoAcceptRejectedDefault,
	}
}

func parseHookAutoAcceptValue(value any) (accepted bool, valid bool) {
	switch v := value.(type) {
	case nil:
		return false, true
	case bool:
		return v, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}
