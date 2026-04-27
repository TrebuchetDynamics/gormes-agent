package gateway

import "strings"

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
