package hermes

import (
	"os"
	"strings"
)

// truthyStrings is the shared set of string values considered "truthy",
// mirroring hermes-agent/utils.py TRUTHY_STRINGS.
var truthyStrings = map[string]struct{}{
	"1":    {},
	"true": {},
	"yes":  {},
	"on":   {},
}

// IsTruthyValue coerces a bool-ish value using the shared truthy string set.
// Mirrors hermes-agent/utils.py is_truthy_value.
func IsTruthyValue(value interface{}, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	if b, ok := value.(bool); ok {
		return b
	}
	if s, ok := value.(string); ok {
		_, ok := truthyStrings[strings.ToLower(strings.TrimSpace(s))]
		return ok
	}
	return false
}

// EnvVarEnabled returns true when an environment variable is set to a truthy
// value. Mirrors hermes-agent/utils.py env_var_enabled.
func EnvVarEnabled(name, defaultVal string) bool {
	return IsTruthyValue(os.Getenv(name), false)
}
