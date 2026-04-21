package progress

import (
	"maps"
	"slices"
)

// sortedMapKeys returns the keys of m in lexicographic order.
// Single generic helper used wherever deterministic map iteration is
// needed (rendering, validation). Phase and subphase keys are expected
// to be single-digit indices; a key ≥ "10" would sort before "2"
// under lex ordering. Revisit if the roadmap ever grows past 9 phases
// or 9 sub-indices per phase.
func sortedMapKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
