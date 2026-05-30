package progress

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/roadmaporder"

// sortedMapKeys returns the keys of m in roadmap order so phases and
// subphases like 2.B.10 sort after 2.B.9 rather than before 2.B.2.
func sortedMapKeys[V any](m map[string]V) []string {
	return roadmaporder.SortedMapKeys(m)
}

func compareRoadmapKeys(a, b string) int {
	return roadmaporder.CompareKeys(a, b)
}
