package progress

import (
	"cmp"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// sortedMapKeys returns the keys of m in roadmap order so phases and
// subphases like 2.B.10 sort after 2.B.9 rather than before 2.B.2.
func sortedMapKeys[V any](m map[string]V) []string {
	return slices.SortedFunc(maps.Keys(m), compareRoadmapKeys)
}

func compareRoadmapKeys(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if diff := compareRoadmapPart(aParts[i], bParts[i]); diff != 0 {
			return diff
		}
	}

	return cmp.Compare(len(aParts), len(bParts))
}

func compareRoadmapPart(a, b string) int {
	if aNum, err := strconv.Atoi(a); err == nil {
		if bNum, err := strconv.Atoi(b); err == nil {
			return cmp.Compare(aNum, bNum)
		}
		return -1
	}
	if _, err := strconv.Atoi(b); err == nil {
		return 1
	}

	return cmp.Compare(a, b)
}
