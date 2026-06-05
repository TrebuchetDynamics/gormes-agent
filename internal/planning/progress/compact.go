package progress

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/compaction"

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
				if !compaction.Needs(note) {
					continue
				}
				sp.Items[i].Note = compaction.Compact(note)
				n++
			}
		}
	}
	return n
}
