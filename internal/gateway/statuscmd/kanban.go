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
func FormatKanbanDispatcherLines(status KanbanDispatcherStatus, esc func(string) string) []string {
	state := strings.TrimSpace(status.State)
	if state == "" {
		state = "unknown"
	}
	lines := []string{
		"**Kanban Dispatcher:** `" + state + "`",
	}
	if strings.TrimSpace(status.LastTickAt) != "" {
		lines = append(lines, "**Kanban Last Tick:** `"+strings.TrimSpace(status.LastTickAt)+"`")
	}
	lines = append(lines,
		fmt.Sprintf("**Kanban Spawned:** %d", status.Spawned),
		fmt.Sprintf("**Kanban Spawn Failed:** %d", status.SpawnFailed),
		fmt.Sprintf("**Kanban Auto Blocked:** %d", status.AutoBlocked),
	)
	if strings.TrimSpace(status.LastError) != "" {
		if esc == nil {
			esc = func(s string) string { return s }
		}
		lines = append(lines, "**Kanban Last Error:** "+esc(status.LastError))
	}
	return lines
}
