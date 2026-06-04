package modelchoice

import "strings"

const UnlimitedSuggestions = -1

func SuggestionsForPrompt(suggestions []string, max int) []string {
	if max == UnlimitedSuggestions {
		max = len(suggestions)
	}
	return BoundedSuggestions(suggestions, max)
}

func BoundedSuggestions(suggestions []string, max int) []string {
	if max <= 0 {
		return nil
	}
	out := make([]string, 0, min(len(suggestions), max))
	seen := map[string]struct{}{}
	for _, suggestion := range suggestions {
		suggestion = strings.TrimSpace(suggestion)
		key := strings.ToLower(suggestion)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, suggestion)
		if len(out) == max {
			return out
		}
	}
	return out
}

func DefaultChoiceID(models []string, current string) string {
	current = strings.TrimSpace(current)
	if current == "" {
		return ""
	}
	for _, model := range models {
		if strings.EqualFold(strings.TrimSpace(model), current) {
			return strings.TrimSpace(model)
		}
	}
	return ""
}

func IndexChoice(models []string, current string) int {
	current = strings.TrimSpace(current)
	if current == "" {
		return -1
	}
	for i, model := range models {
		if strings.EqualFold(strings.TrimSpace(model), current) {
			return i
		}
	}
	return -1
}
