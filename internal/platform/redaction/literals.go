package redaction

import (
	"sort"
	"strings"
)

// RedactLiterals replaces literal secret values with marker. Values are
// processed longest-first (then lexicographic) so overlapping values cannot
// leave a partial match. Empty values are skipped because replacing the empty
// string would inject the marker between every byte of the output.
func RedactLiterals(text string, values []string, marker string) string {
	if len(values) == 0 {
		return text
	}
	if marker == "" {
		marker = defaultSecretMarker
	}
	ordered := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			ordered = append(ordered, value)
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if len(ordered[i]) != len(ordered[j]) {
			return len(ordered[i]) > len(ordered[j])
		}
		return ordered[i] < ordered[j]
	})
	for _, value := range ordered {
		text = strings.ReplaceAll(text, value, marker)
	}
	return text
}
