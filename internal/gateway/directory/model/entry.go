package model

import entrymodel "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/entry"

// Entry describes one platform target that can be selected by exact ID, human
// display name, guild-qualified name, or type/display lookup.
type Entry = entrymodel.Entry

// Evidence carries user-safe degraded-mode evidence without leaking local state paths.
type Evidence = entrymodel.Evidence

// NormalizePlatform returns the canonical platform key used by directory stores,
// refresh merges, and channel lookups.
func NormalizePlatform(platform string) string {
	return entrymodel.NormalizePlatform(platform)
}

// NormalizeEntry trims persisted target fields before merge and lookup.
func NormalizeEntry(entry Entry) Entry {
	return entrymodel.NormalizeEntry(entry)
}

// NormalizeQuery returns the canonical form used for human channel lookups.
func NormalizeQuery(value string) string {
	return entrymodel.NormalizeQuery(value)
}

// UpsertEntry replaces an existing entry with the same normalized ID or appends
// it when no cached target exists yet.
func UpsertEntry(entries []Entry, entry Entry) []Entry {
	return entrymodel.UpsertEntry(entries, entry)
}

// UpsertValidEntry normalizes and upserts a complete directory entry. It returns
// false without changing entries when the entry lacks the minimum cached-target
// contract used by refresh merges.
func UpsertValidEntry(entries []Entry, entry Entry) ([]Entry, bool) {
	return entrymodel.UpsertValidEntry(entries, entry)
}
