package cli

import "strings"

const loginAuthAddOAuthSuggestion = "did you mean \"gormes auth add <provider> --type oauth\"?"

// TypoSuggestion returns deterministic, secret-safe operator guidance for
// removed or compatibility command spellings that must not silently execute.
func TypoSuggestion(args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	if strings.TrimSpace(args[0]) != "login" {
		return "", false
	}
	return loginAuthAddOAuthSuggestion, true
}
