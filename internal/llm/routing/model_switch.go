package routing

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing/modelswitch"

// ModelIdentity represents a vendor slug and family prefix used for catalog
// resolution. Mirrors hermes-agent/hermes_cli/model_switch.py ModelIdentity.
type ModelIdentity = modelswitch.ModelIdentity

// ModelAliases is the built-in map of short alias names to ModelIdentity,
// matching hermes-agent/hermes_cli/model_switch.py MODEL_ALIASES.
var ModelAliases = modelswitch.ModelAliases

// DirectAlias is an exact model mapping that bypasses catalog resolution.
// Mirrors hermes-agent/hermes_cli/model_switch.py DirectAlias.
type DirectAlias = modelswitch.DirectAlias

// ModelSwitchResult is the result of a model switch attempt.
// Mirrors hermes-agent/hermes_cli/model_switch.py ModelSwitchResult.
type ModelSwitchResult = modelswitch.ModelSwitchResult

// ParseModelFlags parses --provider and --global flags from model command args.
// Mirrors hermes-agent/hermes_cli/model_switch.py parse_model_flags.
func ParseModelFlags(rawArgs string) (modelInput string, explicitProvider string, isGlobal bool) {
	return modelswitch.ParseModelFlags(rawArgs)
}

// ModelSortKey produces a deterministic sort key for model IDs.
// Mirrors hermes-agent/hermes_cli/model_switch.py _model_sort_key.
func ModelSortKey(modelID, prefix string) (int, string, string) {
	return modelswitch.ModelSortKey(modelID, prefix)
}

// SortedModelAliases returns model alias keys sorted by ModelSortKey.
func SortedModelAliases() []string {
	return modelswitch.SortedModelAliases()
}
