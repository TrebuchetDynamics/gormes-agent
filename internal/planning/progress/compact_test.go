package progress

import (
	"reflect"
	"strings"
	"testing"
)

func fatNote() string {
	return "SHIPPED 2026-05-12 (gormes-tdd-slice). " +
		strings.Repeat("Implemented the deep behavior with full TDD coverage. ", 80) +
		"\n\nFollow-up: none. tdd-slice does not commit."
}

// CompactCompletedNotes rewrites only completed-row notes to a one-line
// pointer, preserving every other field, and leaves non-complete rows and
// already-concise notes untouched.
func TestCompactCompletedNotes_OnlyCompletedVerboseNotes(t *testing.T) {
	p := &Progress{Phases: map[string]Phase{
		"8": {Name: "Phase 8", Deliverable: "d", Subphases: map[string]Subphase{
			"8.F": {Name: "Cost", Items: []Item{
				{Name: "done row", Status: StatusComplete, Priority: "P1",
					Contract: "keep me", Acceptance: []string{"a1"}, Note: fatNote()},
				{Name: "planned row", Status: StatusPlanned, Note: fatNote()},
				{Name: "short done", Status: StatusComplete, Note: "Complete: trivial."},
			}},
		}},
	}}
	before := deepCopyProgress(t, p)

	n := CompactCompletedNotes(p)
	if n != 1 {
		t.Fatalf("exactly the one verbose completed row should compact, got n=%d", n)
	}

	got := p.Phases["8"].Subphases["8.F"].Items
	doneNote := got[0].Note
	if strings.Contains(doneNote, "\n") {
		t.Fatalf("compacted note must be a single line: %q", doneNote)
	}
	if !strings.HasPrefix(doneNote, "SHIPPED ") || !strings.Contains(doneNote, "see git log — ") {
		t.Fatalf("compacted note must be the SHIPPED pointer form: %q", doneNote)
	}
	if len(doneNote) > 200 {
		t.Fatalf("compacted note must be short, got %d chars: %q", len(doneNote), doneNote)
	}
	if !strings.Contains(doneNote, "2026-05-12") {
		t.Fatalf("date present in source note should be preserved: %q", doneNote)
	}

	// Every other field of the compacted row is unchanged.
	wantRow0 := before.Phases["8"].Subphases["8.F"].Items[0]
	gotRow0 := got[0]
	gotRow0.Note = wantRow0.Note // ignore the (intentionally changed) note
	if !reflect.DeepEqual(gotRow0, wantRow0) {
		t.Fatalf("compaction must preserve all non-note fields:\n got=%+v\nwant=%+v", gotRow0, wantRow0)
	}
	// Non-complete row note untouched; already-concise completed note untouched.
	if got[1].Note != fatNote() {
		t.Fatalf("planned row note must be untouched")
	}
	if got[2].Note != "Complete: trivial." {
		t.Fatalf("already-concise completed note must be untouched, got %q", got[2].Note)
	}
}

// Idempotent: compacting an already-compacted backlog is a no-op.
func TestCompactCompletedNotes_Idempotent(t *testing.T) {
	p := &Progress{Phases: map[string]Phase{
		"8": {Name: "P", Deliverable: "d", Subphases: map[string]Subphase{
			"8.F": {Name: "C", Items: []Item{
				{Name: "r", Status: StatusComplete, Note: fatNote()},
			}},
		}},
	}}
	CompactCompletedNotes(p)
	first := p.Phases["8"].Subphases["8.F"].Items[0].Note

	n := CompactCompletedNotes(p)
	if n != 0 {
		t.Fatalf("second compaction must be a no-op, got n=%d", n)
	}
	if p.Phases["8"].Subphases["8.F"].Items[0].Note != first {
		t.Fatalf("idempotency violated: %q -> %q", first, p.Phases["8"].Subphases["8.F"].Items[0].Note)
	}
}

func deepCopyProgress(t *testing.T, p *Progress) *Progress {
	t.Helper()
	out := &Progress{Meta: p.Meta, Phases: make(map[string]Phase, len(p.Phases))}
	for pk, ph := range p.Phases {
		sps := make(map[string]Subphase, len(ph.Subphases))
		for sk, sp := range ph.Subphases {
			items := append([]Item(nil), sp.Items...)
			sp.Items = items
			sps[sk] = sp
		}
		ph.Subphases = sps
		out.Phases[pk] = ph
	}
	return out
}
