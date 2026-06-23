package typo

// TypoSuggestion is the pre-Cobra extension point for deterministic,
// secret-safe guidance on removed command spellings.
func TypoSuggestion(args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	switch args[0] {
	case "login":
		return "The 'gormes login' command has been removed.\nUse 'gormes auth' to manage credentials,\n'gormes model' to select a provider, or 'gormes setup' for full setup.", true
	case "onboard":
		return "Use `gormes setup` for first-run setup, or `gormes doctor --offline --target terminal --json` for machine-readable readiness.", true
	default:
		return "", false
	}
}
