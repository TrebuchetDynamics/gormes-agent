package titlecmd

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/commandline"
)

const MaxSessionTitleRunes = 100

func ParseArg(text string) (string, bool) {
	body := strings.TrimSpace(text)
	if body == "" {
		return "", false
	}
	fields := strings.Fields(body)
	if len(fields) == 0 || commandline.Name(fields[0]) != "title" {
		return body, true
	}
	idx := strings.Index(body, fields[0])
	if idx < 0 {
		return "", false
	}
	arg := strings.TrimSpace(body[idx+len(fields[0]):])
	return arg, arg != ""
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
	cleaned := strings.Join(strings.Fields(b.String()), " ")
	if cleaned == "" {
		return "", nil
	}
	if utf8.RuneCountInString(cleaned) > MaxSessionTitleRunes {
		return "", fmt.Errorf("Title too long (%d chars, max %d)", utf8.RuneCountInString(cleaned), MaxSessionTitleRunes)
	}
	return cleaned, nil
}

func skipRune(r rune) bool {
	if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
		return true
	}
	if r == 0x7f {
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
