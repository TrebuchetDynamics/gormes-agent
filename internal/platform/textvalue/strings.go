package textvalue

import "strings"

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

// CompactKeyToken lowercases, trims, and removes the separators commonly used
// in JSON/log field names so policy checks can match api_key, api-key,
// api.key, and api key with one contract.
func CompactKeyToken(value string) string {
	return keyTokenReplacer.Replace(LowerTrim(value))
}
