package configreload

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrUnavailable = errors.New("gateway config reload unavailable")

const sanitizedErrorMaxBytes = 240

// SanitizeError returns bounded, redacted operator-facing reload failure text.
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	for _, hint := range []string{"api_key", "token", "authorization", "bearer ", "secret", "password"} {
		if strings.Contains(lower, hint) {
			return "[redacted]"
		}
	}
	return truncateUTF8ByBytes(msg, sanitizedErrorMaxBytes)
}

func truncateUTF8ByBytes(msg string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(msg) <= maxBytes {
		return msg
	}
	for end := maxBytes; end > 0; end-- {
		if utf8.ValidString(msg[:end]) {
			return msg[:end]
		}
	}
	return ""
}

func CloneStringMap(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func CloneBoolMap(input map[string]bool) map[string]bool {
	out := make(map[string]bool, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func CloneNestedBoolMap(input map[string]map[string]bool) map[string]map[string]bool {
	out := make(map[string]map[string]bool, len(input))
	for platform, users := range input {
		out[platform] = CloneBoolMap(users)
	}
	return out
}
