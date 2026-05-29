package hookpolicy

import "strings"

// AutoAcceptEvidence identifies why hook auto-accept was allowed or rejected.
type AutoAcceptEvidence string

const (
	AutoAcceptAcceptedByCLI    AutoAcceptEvidence = "accepted_by_cli"
	AutoAcceptAcceptedByEnv    AutoAcceptEvidence = "accepted_by_env"
	AutoAcceptAcceptedByConfig AutoAcceptEvidence = "accepted_by_config"
	AutoAcceptRejectedDefault  AutoAcceptEvidence = "rejected_default"
	AutoAcceptInvalid          AutoAcceptEvidence = "hook_auto_accept_invalid"
)

// AutoAcceptInputs are the precedence-ordered inputs for hook auto-accept.
type AutoAcceptInputs struct {
	CLIAccept   bool
	EnvValue    *string
	ConfigValue any
}

// AutoAcceptDecision is the normalized hook auto-accept verdict.
type AutoAcceptDecision struct {
	Accept   bool
	Evidence AutoAcceptEvidence
}

// ResolveAutoAccept applies CLI > env > config precedence for hook auto-accept.
func ResolveAutoAccept(inputs AutoAcceptInputs) AutoAcceptDecision {
	if inputs.CLIAccept {
		return AutoAcceptDecision{Accept: true, Evidence: AutoAcceptAcceptedByCLI}
	}
	if inputs.EnvValue != nil {
		return autoAcceptDecisionForValue(*inputs.EnvValue, AutoAcceptAcceptedByEnv)
	}
	return autoAcceptDecisionForValue(inputs.ConfigValue, AutoAcceptAcceptedByConfig)
}

func autoAcceptDecisionForValue(value any, acceptedEvidence AutoAcceptEvidence) AutoAcceptDecision {
	accepted, valid := parseAutoAcceptValue(value)
	if accepted {
		return AutoAcceptDecision{Accept: true, Evidence: acceptedEvidence}
	}
	if !valid {
		return AutoAcceptDecision{Accept: false, Evidence: AutoAcceptInvalid}
	}
	return AutoAcceptDecision{Accept: false, Evidence: AutoAcceptRejectedDefault}
}

func parseAutoAcceptValue(value any) (accepted bool, valid bool) {
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
