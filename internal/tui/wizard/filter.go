package wizard

import "strings"

// FilterChoices returns the subset of choices whose ID or Label contains
// every token in the query (case-insensitive, whitespace-split). An empty
// query returns all choices. The returned indices map each filtered choice
// back to its position in the original choices slice so the caller can
// resolve the selection against the full list.
func FilterChoices(choices []Choice, query string) (filtered []Choice, indices []int) {
	tokens := filterTokens(query)
	if len(tokens) == 0 {
		indices = make([]int, len(choices))
		for i := range choices {
			indices[i] = i
		}
		return append([]Choice(nil), choices...), indices
	}
	for i, choice := range choices {
		haystack := strings.ToLower(choice.ID) + " " + strings.ToLower(choice.Label)
		if filterMatchesAll(haystack, tokens) {
			filtered = append(filtered, choice)
			indices = append(indices, i)
		}
	}
	if filtered == nil {
		filtered = []Choice{}
	}
	if indices == nil {
		indices = []int{}
	}
	return filtered, indices
}

func filterTokens(query string) []string {
	raw := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	out := make([]string, 0, len(raw))
	for _, token := range raw {
		if token != "" {
			out = append(out, token)
		}
	}
	return out
}

func filterMatchesAll(haystack string, tokens []string) bool {
	for _, token := range tokens {
		if !strings.Contains(haystack, token) {
			return false
		}
	}
	return true
}
