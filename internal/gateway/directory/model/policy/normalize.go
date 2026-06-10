package policy

import (
	"strings"
	"unicode"
)

// TrimText is the shared text-normalization primitive for persisted directory
// value contracts. Keeping field trimming in one place prevents entry and
// remembered-source normalization from drifting.
func TrimText(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, value)
	return strings.TrimSpace(value)
}
