package statuscmd

import (
	"reflect"
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
