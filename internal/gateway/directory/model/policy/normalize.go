package policy

import "strings"

// TrimText is the shared text-normalization primitive for persisted directory
// value contracts. Keeping field trimming in one place prevents entry and
// remembered-source normalization from drifting.
func TrimText(value string) string {
	return strings.TrimSpace(value)
}
