// Package compaction provides deterministic note compaction primitives for
// progress rows. It intentionally knows nothing about the progress schema; the
// root progress package owns row traversal and compatibility exports.
package compaction

import (
	"regexp"
	"strings"
)

// Prefix marks a note that has already been reduced to the one-line
// shipped-evidence pointer form. Git history holds the full prose.
const Prefix = "SHIPPED "

// Marker is the fixed infix every compacted note carries. It is what makes
// compaction recognisable and therefore idempotent.
const Marker = "see git log — "

// MaxConciseNoteLen is the byte budget under which a single-line note is
// already considered concise. A compacted pointer always fits inside it, so a
// second compaction pass is a no-op.
const MaxConciseNoteLen = 200

// MaxBehaviorSummary bounds the human-readable behavior tail of a compacted
// note so the whole pointer stays well under MaxConciseNoteLen.
const MaxBehaviorSummary = 140

var (
	wsRun    = regexp.MustCompile(`\s+`)
	isoDate  = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	leadBoil = regexp.MustCompile(`^SHIPPED\s+(\d{4}-\d{2}-\d{2}\s*)?(\([^)]*\)\s*)?[.\-:–—]*\s*`)
)

// Needs reports whether a completed-row note is verbose enough to be worth
// replacing with a pointer. Empty notes and notes that are already a single
// short line (including previously compacted pointers) are skipped.
func Needs(note string) bool {
	if strings.TrimSpace(note) == "" {
		return false
	}
	if strings.ContainsRune(note, '\n') {
		return true
	}
	return len(note) > MaxConciseNoteLen
}

// Compact builds the one-line pointer deterministically from the existing
// note: it preserves any ISO date already present and uses the leading prose
// (boilerplate stripped) as a bounded behavior summary. It never invents a
// commit SHA or any fact not already in the note.
func Compact(note string) string {
	collapsed := strings.TrimSpace(wsRun.ReplaceAllString(note, " "))
	date := isoDate.FindString(collapsed)

	summary := leadBoil.ReplaceAllString(collapsed, "")
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = collapsed
	}
	summary = TruncateSummary(summary, MaxBehaviorSummary)

	var b strings.Builder
	b.WriteString(Prefix)
	if date != "" {
		b.WriteString(date)
		b.WriteByte(' ')
	}
	b.WriteString(Marker)
	b.WriteString(summary)
	return b.String()
}

// TruncateSummary caps a single-line summary at max bytes, breaking on a word
// boundary where possible and signalling the cut with an ellipsis so the
// truncation is never silently lossy.
func TruncateSummary(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	if sp := strings.LastIndexByte(cut, ' '); sp > max/2 {
		cut = cut[:sp]
	}
	return strings.TrimRight(cut, " ") + "…"
}
