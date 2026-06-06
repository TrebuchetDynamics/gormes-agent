package slashcompletion

import "sort"

func sortedCompletionKeys[T any](m map[string]T) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func flattenCompletionGroups(groups [][]Completion) []Completion {
	count := 0
	for _, group := range groups {
		count += len(group)
	}
	if count == 0 {
		return nil
	}
	merged := make([]Completion, 0, count)
	for _, group := range groups {
		merged = append(merged, group...)
	}
	return merged
}

func uniqueSortedCompletions(candidates []Completion) []Completion {
	if len(candidates) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(candidates))
	out := make([]Completion, 0, len(candidates))
	for _, c := range candidates {
		key := completionKey(c.Name)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool { return completionKey(out[i].Name) < completionKey(out[j].Name) })
	return out
}
