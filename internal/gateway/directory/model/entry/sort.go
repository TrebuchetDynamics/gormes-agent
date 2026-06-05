package entry

import "sort"

// SortEntriesByNameID orders entries by the stable display identity used by
// refresh persistence and grouped display rendering.
func SortEntriesByNameID(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].ID < entries[j].ID
	})
}
