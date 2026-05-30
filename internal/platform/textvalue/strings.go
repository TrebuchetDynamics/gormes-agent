package textvalue

import (
	"sort"
	"strings"
)

var keyTokenReplacer = strings.NewReplacer("_", "", "-", "", ".", "", " ", "")

// IsNonBlank reports whether value has any non-whitespace content.
func IsNonBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// LowerTrim returns value with surrounding whitespace removed and letters folded
// to lower case for command, provider, channel, and evidence keys.
func LowerTrim(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// FirstNonBlank returns the first value whose trimmed form is non-empty while
// preserving the original value. Use FirstNonEmptyTrimmed when callers need the
// normalized value instead of the source text.
func FirstNonBlank(values ...string) string {
	for _, value := range values {
		if IsNonBlank(value) {
			return value
		}
	}
	return ""
}

// FirstNonEmptyTrimmed returns the first value whose trimmed form is non-empty.
// The returned value is trimmed so status/evidence fields do not preserve
// accidental surrounding whitespace.
func FirstNonEmptyTrimmed(values ...string) string {
	return strings.TrimSpace(FirstNonBlank(values...))
}

// CompactWhitespace trims text and collapses all whitespace runs to one ASCII
// space for command paths, seeds, and other operator-facing identifiers.
func CompactWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

// TrimmedLines splits value on newlines and trims surrounding whitespace from
// each line while preserving line count for callers that care about interior
// blank lines.
func TrimmedLines(value string) []string {
	parts := strings.Split(value, "\n")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

// FirstNonBlankLine returns the first trimmed line with content.
func FirstNonBlankLine(value string) string {
	for _, line := range TrimmedLines(value) {
		if IsNonBlank(line) {
			return line
		}
	}
	return ""
}

// CompactKeyToken lowercases, trims, and removes the separators commonly used
// in JSON/log field names so policy checks can match api_key, api-key,
// api.key, and api key with one contract.
func CompactKeyToken(value string) string {
	return keyTokenReplacer.Replace(LowerTrim(value))
}

// SortedKeys returns a deterministic copy of the keys in a string-keyed map.
// It centralizes platform report/manifest ordering so callers do not each
// reimplement the same map iteration and sort policy.
func SortedKeys[V any](values map[string]V) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
