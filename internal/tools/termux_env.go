package tools

import "strings"

func isTermuxToolEnv(env func(string) string) bool {
	if strings.TrimSpace(env("TERMUX_VERSION")) != "" {
		return true
	}
	for _, key := range []string{"PREFIX", "HOME"} {
		if strings.Contains(env(key), "com.termux") {
			return true
		}
	}
	return false
}
