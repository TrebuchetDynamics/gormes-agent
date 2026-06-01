package model

import entrymodel "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/entry"

// SortEntriesByNameID orders entries by the stable display identity used by
// refresh persistence and grouped display rendering.
func SortEntriesByNameID(entries []Entry) {
	entrymodel.SortEntriesByNameID(entries)
}
