//go:build !slim

package commandtemplate

import (
	"regexp"
	"runtime"
	"strings"
)

func Render(template string, placeholders map[string]string) string {
	if template == "" || len(placeholders) == 0 {
		return template
	}
	markerOpen := "\x00GORMES_TTS_OPEN\x00"
	markerClose := "\x00GORMES_TTS_CLOSE\x00"
	protected := strings.ReplaceAll(template, "{{", markerOpen)
	protected = strings.ReplaceAll(protected, "}}", markerClose)
	pattern := regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`)
	matches := pattern.FindAllStringSubmatchIndex(protected, -1)
	var rendered strings.Builder
	rendered.Grow(len(protected))
	last := 0
	for _, match := range matches {
		rendered.WriteString(protected[last:match[0]])
		token := protected[match[0]:match[1]]
		name := protected[match[2]:match[3]]
		value, ok := placeholders[name]
		if !ok {
			rendered.WriteString(token)
		} else {
			rendered.WriteString(quotePlaceholder(value, shellQuoteContext(protected, match[0])))
		}
		last = match[1]
	}
	rendered.WriteString(protected[last:])
	out := rendered.String()
	out = strings.ReplaceAll(out, markerOpen, "{")
	out = strings.ReplaceAll(out, markerClose, "}")
	return out
}

func shellQuoteContext(template string, position int) string {
	quote := byte(0)
	escaped := false
	for i := 0; i < position && i < len(template); i++ {
		ch := template[i]
		switch quote {
		case '\'':
			if ch == '\'' {
				quote = 0
			}
		case '"':
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				quote = 0
			}
		default:
			if ch == '\'' || ch == '"' {
				quote = ch
			} else if ch == '\\' {
				i++
			}
		}
	}
	if quote == 0 {
		return ""
	}
	return string(quote)
}

func quotePlaceholder(value, quoteContext string) string {
	switch quoteContext {
	case "'":
		return strings.ReplaceAll(value, "'", `'\''`)
	case `"`:
		replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`")
		return replacer.Replace(value)
	default:
		return shellQuote(value)
	}
}

var shellSafePlaceholder = regexp.MustCompile(`^[A-Za-z0-9_./:@%+=,-]+$`)

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(value, `"`, `\"`)
	}
	if shellSafePlaceholder.MatchString(value) {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
