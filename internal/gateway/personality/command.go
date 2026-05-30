package personality

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/commandline"
)

// ParseArg returns the argument payload for a /personality invocation. If text
// is not a /personality command line, the trimmed text is returned unchanged so
// callers can reuse the parser for already-split payloads.
func ParseArg(text string) string {
	body := strings.TrimSpace(text)
	if body == "" {
		return ""
	}
	fields := strings.Fields(body)
	if len(fields) == 0 || commandline.Name(fields[0]) != "personality" {
		return body
	}
	idx := strings.Index(body, fields[0])
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(body[idx+len(fields[0]):])
}

// TruncateDesc shortens a personality prompt description by rune count.
func TruncateDesc(s string, max int) string {
	if len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
}

// RenderList formats the /personality list response.
func RenderList(active string, personalities map[string]string, descMax int) string {
	lines := []string{"**Personalities:**"}
	if strings.TrimSpace(active) != "" {
		lines = append(lines, fmt.Sprintf("Active: **%s**", active))
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
			desc := TruncateDesc(personalities[name], descMax)
			lines = append(lines, fmt.Sprintf("  • `/personality %s` — %s", name, desc))
		}
	} else {
		lines = append(lines, "(no personalities configured)")
	}
	lines = append(lines, "", "Usage: `/personality <name>` or `/personality none` to clear")
	return strings.Join(lines, "\n")
}

// RenderUnknown formats the unknown-personality guidance response.
func RenderUnknown(name string, personalities map[string]string) string {
	known := make([]string, 0, len(personalities))
	for n := range personalities {
		known = append(known, n)
	}
	sort.Strings(known)
	var hint string
	if len(known) > 0 {
		hint = " Available: " + strings.Join(known, ", ")
	}
	return fmt.Sprintf("Unknown personality %q.%s", name, hint)
}
