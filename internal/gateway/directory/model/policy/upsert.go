package policy

// UpsertByNormalizedID is the shared replacement policy for directory entries
// and remembered source entries. Callers keep their own normalization and
// minimum-validity contracts while reusing the identical ID replacement rule.
func UpsertByNormalizedID[T any](entries []T, entry T, normalize func(T) T, id func(T) string, valid func(T) bool) ([]T, bool) {
	entry = normalize(entry)
	if !valid(entry) {
		return entries, false
	}
	entryID := id(entry)
	for i, existing := range entries {
		if TrimText(id(existing)) == entryID {
			entries[i] = entry
			return entries, true
		}
	}
	return append(entries, entry), true
}
