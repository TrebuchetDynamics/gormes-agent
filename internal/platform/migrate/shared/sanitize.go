package shared

import "strings"

// SanitizeError strips known secret substrings from error messages before they
// can land in operator-facing output like WriteOutcome.Errors or ApplyOutcome.
// Manifest builders already redact secret values; this is a defense-in-depth
// check covering filesystem errors that might echo a path containing a
// secret-looking token.
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if idx := strings.Index(msg, "sk-"); idx >= 0 {
		end := idx + 3
		for end < len(msg) && (IsAlphanum(msg[end]) || msg[end] == '-' || msg[end] == '_') {
			end++
		}
		msg = msg[:idx] + "[REDACTED]" + msg[end:]
	}
	return msg
}

// IsAlphanum reports whether b is an ASCII letter or digit.
func IsAlphanum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
