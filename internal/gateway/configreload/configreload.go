package configreload

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/mapclone"
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
	compact := compactSecretSeparators(lower)
	for _, hint := range []string{"api_key", "api key", "api-key", "token", "authorization", "bearer ", "secret", "password"} {
		if strings.Contains(lower, hint) {
			return "[redacted]"
		}
	}
	for _, hint := range []string{"apikey", "authorization", "bearer", "secret", "password", "token"} {
		if strings.Contains(compact, hint) {
			return "[redacted]"
		}
	}
	return truncateUTF8ByBytes(renderOperatorErrorText(msg), sanitizedErrorMaxBytes)
}

func compactSecretSeparators(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func renderOperatorErrorText(msg string) string {
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
		"#", "＃",
	)
	return strings.Join(strings.Fields(replacer.Replace(msg)), " ")
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
	return mapclone.StringStringOrEmpty(input)
}

func CloneBoolMap(input map[string]bool) map[string]bool {
	return mapclone.StringBoolOrEmpty(input)
}

func CloneNestedBoolMap(input map[string]map[string]bool) map[string]map[string]bool {
	return mapclone.NestedStringBoolOrEmpty(input)
}
