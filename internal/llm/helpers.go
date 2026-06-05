package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/envhelpers"

// IsTruthyValue coerces a bool-ish value using the shared truthy string set.
// Mirrors hermes-agent/utils.py is_truthy_value.
func IsTruthyValue(value interface{}, defaultValue bool) bool {
	return envhelpers.IsTruthyValue(value, defaultValue)
}

// EnvVarEnabled returns true when an environment variable is set to a truthy
// value. Mirrors hermes-agent/utils.py env_var_enabled.
func EnvVarEnabled(name, defaultVal string) bool {
	return envhelpers.EnvVarEnabled(name, defaultVal)
}
