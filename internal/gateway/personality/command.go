package personality

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/commandline"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

// ParseArg returns the argument payload for a /personality invocation. If text
// is not a /personality command line, the trimmed text is returned unchanged so
// callers can reuse the parser for already-split payloads.
func ParseArg(text string) string {
	body := strings.TrimSpace(text)
	if body == "" {
		return ""
	}
	if payload, ok := commandline.PayloadIfCommand(body, "personality"); ok {
		return payload
	}
	if isSlashCommandLike(body) {
		return ""
	}
	return body
}

func isSlashCommandLike(body string) bool {
	return strings.HasPrefix(body, "/") || strings.HasPrefix(body, "／")
}

// TruncateDesc shortens a personality prompt description by rune count.
func TruncateDesc(s string, max int) string {
	if max < 0 {
		max = 0
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// RenderList formats the /personality list response.
func RenderList(active string, personalities map[string]string, descMax int) string {
	lines := []string{"**Personalities:**"}
	if strings.TrimSpace(active) != "" {
		lines = append(lines, fmt.Sprintf("Active: **%s**", renderValue(active)))
	} else {
		lines = append(lines, "Active: *(none)*")
	}
	if len(personalities) > 0 {
		names := make([]string, 0, len(personalities))
		for name := range personalities {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			desc := TruncateDesc(renderValue(personalities[name]), descMax)
			lines = append(lines, fmt.Sprintf("  • `/personality %s` — %s", renderValue(name), desc))
		}
	} else {
		lines = append(lines, "(no personalities configured)")
	}
	lines = append(lines, "", "Usage: `/personality <name>` or `/personality none` to clear")
	return strings.Join(lines, "\n")
}

// RenderSetConfirmation formats the confirmation after switching personalities.
func RenderSetConfirmation(name string) string {
	return fmt.Sprintf("Personality set to **%s**.", renderValue(name))
}

// RenderUnknown formats the unknown-personality guidance response.
func RenderUnknown(name string, personalities map[string]string) string {
	known := make([]string, 0, len(personalities))
	for n := range personalities {
		known = append(known, renderValue(n))
	}
	sort.Strings(known)
	var hint string
	if len(known) > 0 {
		hint = " Available: " + strings.Join(known, ", ")
	}
	return fmt.Sprintf("Unknown personality %q.%s", renderValue(name), hint)
}

func renderValue(value string) string {
	value = collapseRedactedPersonalityAssignments(redaction.RedactSecrets(value))
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
	)
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func collapseRedactedPersonalityAssignments(value string) string {
	replacer := strings.NewReplacer(
		"api_key=[redacted]", "[redacted]",
		"api-key=[redacted]", "[redacted]",
		"authorization=[redacted]", "[redacted]",
		"bearer=[redacted]", "[redacted]",
		"token=[redacted]", "[redacted]",
		"secret=[redacted]", "[redacted]",
		"password=[redacted]", "[redacted]",
	)
	fields := strings.Fields(replacer.Replace(value))
	out := make([]string, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		lower := strings.ToLower(field)
		nextRedacted := i+1 < len(fields) && strings.Contains(strings.ToLower(fields[i+1]), "[redacted]")
		if personalitySecretField(lower) && (strings.Contains(lower, "[redacted]") || nextRedacted) {
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

func personalitySecretField(value string) bool {
	return strings.Contains(value, "api_key") || strings.Contains(value, "api-key") || strings.Contains(value, "apikey") || strings.Contains(value, "authorization") || strings.Contains(value, "bearer") || strings.Contains(value, "token") || strings.Contains(value, "secret") || strings.Contains(value, "password")
}
