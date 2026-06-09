package statuscmd

import (
	"fmt"
	"strings"
)

// KanbanDispatcherStatus is the minimal read model needed to render the
// /status Kanban dispatcher section.
type KanbanDispatcherStatus struct {
	State       string
	LastTickAt  string
	LastError   string
	Spawned     int
	SpawnFailed int
	AutoBlocked int
}

// FormatKanbanDispatcherLines renders the optional Kanban dispatcher section
// for the gateway /status response. esc must apply the channel's text escaping
// policy to free-form error strings.
func renderCodeField(value string) string {
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
		"#", "＃",
	)
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func renderFreeText(value string) string {
	msg := strings.TrimSpace(value)
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	compact := compactSecretSeparators(lower)
	for _, marker := range []string{"token", "api_key", "apikey", "authorization", "bearer", "secret", "password"} {
		if strings.Contains(lower, marker) || strings.Contains(compact, marker) {
			return "[redacted]"
		}
	}
	return renderCodeField(msg)
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

func FormatKanbanDispatcherLines(status KanbanDispatcherStatus, esc func(string) string) []string {
	state := renderCodeField(status.State)
	if state == "" {
		state = "unknown"
	}
	lines := []string{
		"**Kanban Dispatcher:** `" + state + "`",
	}
	if lastTickAt := renderCodeField(status.LastTickAt); lastTickAt != "" {
		lines = append(lines, "**Kanban Last Tick:** `"+lastTickAt+"`")
	}
	lines = append(lines,
		fmt.Sprintf("**Kanban Spawned:** %d", status.Spawned),
		fmt.Sprintf("**Kanban Spawn Failed:** %d", status.SpawnFailed),
		fmt.Sprintf("**Kanban Auto Blocked:** %d", status.AutoBlocked),
	)
	if lastError := renderFreeText(status.LastError); lastError != "" {
		if esc == nil {
			esc = func(s string) string { return s }
		}
		lines = append(lines, "**Kanban Last Error:** "+esc(lastError))
	}
	return lines
}
