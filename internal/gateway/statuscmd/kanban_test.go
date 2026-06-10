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

func TestFormatKanbanDispatcherLinesRemovesHiddenFormatting(t *testing.T) {
	got := FormatKanbanDispatcherLines(KanbanDispatcherStatus{
		State:      "running\u202e",
		LastTickAt: "2026-05-07\u200dT13:14:15Z",
		LastError:  "worker failed\u2066 retry",
	}, nil)
	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{"\u202e", "\u200d", "\u2066"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("FormatKanbanDispatcherLines kept hidden formatting rune %q in:\n%s", forbidden, joined)
		}
	}
	for _, want := range []string{"running", "2026-05-07T13:14:15Z", "worker failed retry"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("FormatKanbanDispatcherLines missing sanitized fragment %q in:\n%s", want, joined)
		}
	}
}

func TestFormatKanbanDispatcherLinesRemovesControlCharacters(t *testing.T) {
	got := FormatKanbanDispatcherLines(KanbanDispatcherStatus{
		State:      "running\x1b[31m",
		LastTickAt: "2026-05-07\u009b13:14:15Z",
		LastError:  "worker failed\x7f retry",
	}, nil)
	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{"\x1b", "\u009b", "\x7f"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("FormatKanbanDispatcherLines kept control character %q in:\n%s", forbidden, joined)
		}
	}
	for _, want := range []string{"running [31m", "2026-05-07 13:14:15Z", "worker failed retry"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("FormatKanbanDispatcherLines missing sanitized fragment %q in:\n%s", want, joined)
		}
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

func TestFormatKanbanDispatcherLinesClampsNegativeCounters(t *testing.T) {
	got := FormatKanbanDispatcherLines(KanbanDispatcherStatus{
		Spawned:     -2,
		SpawnFailed: -3,
		AutoBlocked: -4,
	}, nil)
	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{"-2", "-3", "-4"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("FormatKanbanDispatcherLines leaked negative counter %q in:\n%s", forbidden, joined)
		}
	}
	for _, want := range []string{"**Kanban Spawned:** 0", "**Kanban Spawn Failed:** 0", "**Kanban Auto Blocked:** 0"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("FormatKanbanDispatcherLines missing clamped counter %q in:\n%s", want, joined)
		}
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
