package skin

import "strings"

func SlashName(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	_, rest, ok := strings.Cut(trimmed, " ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(rest)
}

func DisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	return name
}
