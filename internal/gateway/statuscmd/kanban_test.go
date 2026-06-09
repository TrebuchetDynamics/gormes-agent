package statuscmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestFormatKanbanDispatcherLines(t *testing.T) {
	got := FormatKanbanDispatcherLines(KanbanDispatcherStatus{
		State:       "degraded",
		LastTickAt:  "2026-05-07T13:14:15Z",
		LastError:   "worker_spawn_failed: missing profile",
		Spawned:     2,
		SpawnFailed: 1,
		AutoBlocked: 3,
	}, func(s string) string { return "ESC(" + s + ")" })
	want := []string{
		"**Kanban Dispatcher:** `degraded`",
		"**Kanban Last Tick:** `2026-05-07T13:14:15Z`",
		"**Kanban Spawned:** 2",
		"**Kanban Spawn Failed:** 1",
		"**Kanban Auto Blocked:** 3",
		"**Kanban Last Error:** ESC(worker_spawn_failed: missing profile)",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FormatKanbanDispatcherLines() = %#v, want %#v", got, want)
	}
}

func TestFormatKanbanDispatcherLinesSanitizesCodeFields(t *testing.T) {
	got := FormatKanbanDispatcherLines(KanbanDispatcherStatus{
		State:      "running`\n**Injected:** yes",
		LastTickAt: "2026-05-07`\n## takeover",
	}, nil)
	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{"**Injected:**", "## takeover", "running`\n", "2026-05-07`\n"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("FormatKanbanDispatcherLines leaked %q in:\n%s", forbidden, joined)
		}
	}
	for _, want := range []string{"**Kanban Dispatcher:** `running' ''Injected:'' yes`", "**Kanban Last Tick:** `2026-05-07' ＃＃ takeover`"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("FormatKanbanDispatcherLines missing sanitized field %q in:\n%s", want, joined)
		}
	}
}

func TestFormatKanbanDispatcherLinesSanitizesLastError(t *testing.T) {
	got := FormatKanbanDispatcherLines(KanbanDispatcherStatus{
		LastError: "spawn failed\n**Injected:** api key plain-secret",
	}, func(s string) string { return "ESC(" + s + ")" })
	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{"plain-secret", "**Injected:**", "spawn failed"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("FormatKanbanDispatcherLines leaked unsafe last error %q in:\n%s", forbidden, joined)
		}
	}
	if !strings.Contains(joined, "**Kanban Last Error:** ESC([redacted])") {
		t.Fatalf("FormatKanbanDispatcherLines missing redacted last error in:\n%s", joined)
	}
}

func TestFormatKanbanDispatcherLinesDefaultsBlankState(t *testing.T) {
	got := FormatKanbanDispatcherLines(KanbanDispatcherStatus{}, nil)
	want := []string{
		"**Kanban Dispatcher:** `unknown`",
		"**Kanban Spawned:** 0",
		"**Kanban Spawn Failed:** 0",
		"**Kanban Auto Blocked:** 0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FormatKanbanDispatcherLines() = %#v, want %#v", got, want)
	}
}
