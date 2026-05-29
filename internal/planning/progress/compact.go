package progress

import (
	"regexp"
	"strings"
)

// compactedNotePrefix marks a note that has already been reduced to the
// one-line shipped-evidence pointer form. Git history holds the full prose.
const compactedNotePrefix = "SHIPPED "

// compactedNoteMarker is the fixed infix every compacted note carries. It is
// what makes compaction recognisable and therefore idempotent.
const compactedNoteMarker = "see git log — "

// maxConciseNoteLen is the byte budget under which a single-line note is
// already considered concise. A compacted pointer always fits inside it, so a
// second compaction pass is a no-op.
const maxConciseNoteLen = 200

// maxBehaviorSummary bounds the human-readable behavior tail of a compacted
// note so the whole pointer stays well under maxConciseNoteLen.
const maxBehaviorSummary = 140

var (
	wsRun    = regexp.MustCompile(`\s+`)
	isoDate  = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	leadBoil = regexp.MustCompile(`^SHIPPED\s+(\d{4}-\d{2}-\d{2}\s*)?(\([^)]*\)\s*)?[.\-:–—]*\s*`)
)

// CompactCompletedNotes rewrites the verbose shipped-evidence note of every
// status=="complete" row to a single-line pointer of the form
//
//	SHIPPED <YYYY-MM-DD> see git log — <deterministic behavior summary>
//
// (the date is omitted when the source note carries none). Every other field
// of every row is left byte-identical, and notes on non-complete rows are
// never touched. Notes that are already concise (single line, short) — which
// includes notes this function previously produced — are left unchanged, so
// the operation is idempotent. It returns the number of notes rewritten.
func CompactCompletedNotes(p *Progress) int {
	if p == nil {
		return 0
	}
	n := 0
	for pk := range p.Phases {
		ph := p.Phases[pk]
		for sk := range ph.Subphases {
			sp := ph.Subphases[sk]
			for i := range sp.Items {
				if sp.Items[i].Status != StatusComplete {
					continue
				}
				note := sp.Items[i].Note
				if !needsCompaction(note) {
					continue
				}
				sp.Items[i].Note = compactNote(note)
				n++
			}
		}
	}
	return n
}

// needsCompaction reports whether a completed-row note is verbose enough to be
// worth replacing with a pointer. Empty notes and notes that are already a
// single short line (including previously compacted pointers) are skipped.
func needsCompaction(note string) bool {
	if strings.TrimSpace(note) == "" {
		return false
	}
	if strings.ContainsRune(note, '\n') {
		return true
	}
	return len(note) > maxConciseNoteLen
}

// compactNote builds the one-line pointer deterministically from the existing
// note: it preserves any ISO date already present and uses the leading prose
// (boilerplate stripped) as a bounded behavior summary. It never invents a
// commit SHA or any fact not already in the note.
func compactNote(note string) string {
	collapsed := strings.TrimSpace(wsRun.ReplaceAllString(note, " "))
	date := isoDate.FindString(collapsed)

	summary := leadBoil.ReplaceAllString(collapsed, "")
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = collapsed
	}
	summary = truncateSummary(summary, maxBehaviorSummary)

	var b strings.Builder
	b.WriteString(compactedNotePrefix)
	if date != "" {
		b.WriteString(date)
		b.WriteByte(' ')
	}
	b.WriteString(compactedNoteMarker)
	b.WriteString(summary)
	return b.String()
}

// truncateSummary caps a single-line summary at max bytes, breaking on a word
// boundary where possible and signalling the cut with an ellipsis so the
// truncation is never silently lossy.
func truncateSummary(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	if sp := strings.LastIndexByte(cut, ' '); sp > max/2 {
		cut = cut[:sp]
	}
	return strings.TrimRight(cut, " ") + "…"
}
