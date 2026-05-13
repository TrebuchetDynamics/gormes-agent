package cli

// TypoSuggestion is the pre-Cobra extension point for deterministic,
// secret-safe guidance on removed command spellings. It currently has no
// entries because top-level login is a registered compatibility command.
func TypoSuggestion(args []string) (string, bool) {
	return "", false
}
