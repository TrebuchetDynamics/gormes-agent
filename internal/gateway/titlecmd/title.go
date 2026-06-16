package titlecmd

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/commandline"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

const MaxSessionTitleRunes = 100

func ParseArg(text string) (string, bool) {
	body := strings.TrimSpace(text)
	if body == "" {
		return "", false
	}
	if arg, ok := commandline.PayloadIfCommand(body, "title"); ok {
		return arg, arg != ""
	}
	if isSlashCommandLike(body) {
		return "", false
	}
	return body, true
}

func isSlashCommandLike(body string) bool {
	return strings.HasPrefix(body, "/") || strings.HasPrefix(body, "／")
}

func Sanitize(title string) (string, error) {
	if title == "" {
		return "", nil
	}
	var b strings.Builder
	for _, r := range title {
		if skipRune(r) {
			continue
		}
		b.WriteRune(r)
	}
	cleaned := sanitizeTitleSecrets(b.String())
	if cleaned == "" {
		return "", nil
	}
	if utf8.RuneCountInString(cleaned) > MaxSessionTitleRunes {
		return "", fmt.Errorf("Title too long (%d chars, max %d)", utf8.RuneCountInString(cleaned), MaxSessionTitleRunes)
	}
	return cleaned, nil
}

func sanitizeTitleSecrets(value string) string {
	value = redaction.RedactSecrets(value)
	fields := strings.Fields(value)
	out := make([]string, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		lower := strings.ToLower(field)
		nextRedacted := i+1 < len(fields) && strings.Contains(strings.ToLower(fields[i+1]), "[redacted]")
		if secretLikeTitleField(lower) && (strings.Contains(lower, "[redacted]") || nextRedacted) {
			out = append(out, "[redacted]")
			if nextRedacted {
				i++
			}
			continue
		}
		out = append(out, field)
	}
	return strings.Join(out, " ")
}

func secretLikeTitleField(value string) bool {
	return strings.Contains(value, "api_key") || strings.Contains(value, "api-key") || strings.Contains(value, "apikey") || strings.Contains(value, "authorization") || strings.Contains(value, "bearer") || strings.Contains(value, "token") || strings.Contains(value, "secret") || strings.Contains(value, "password")
}

func skipRune(r rune) bool {
	if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
		return true
	}
	if r == 0x7f || (r >= 0x80 && r <= 0x9f) {
		return true
	}
	switch {
	case r >= 0x200b && r <= 0x200f:
		return true
	case r >= 0x2028 && r <= 0x202e:
		return true
	case r >= 0x2060 && r <= 0x2069:
		return true
	case r == 0xfeff || r == 0xfffc:
		return true
	case r >= 0xfff9 && r <= 0xfffb:
		return true
	default:
		return false
	}
}
