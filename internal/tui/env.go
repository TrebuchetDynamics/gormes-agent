package tui

func envValue(env map[string]string, key string) string {
	if env == nil {
		return ""
	}
	return env[key]
}
