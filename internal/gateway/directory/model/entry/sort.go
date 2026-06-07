package entry

import entrycontract "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/entry/contract"

// SortEntriesByNameID orders entries by the stable display identity used by
// refresh persistence and grouped display rendering.
func SortEntriesByNameID(entries []Entry) {
	entrycontract.SortEntriesByNameID(entries)
}
