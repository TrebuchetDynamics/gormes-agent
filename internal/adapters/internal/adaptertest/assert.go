// Package adaptertest provides reusable test helpers for adapter packages.
package adaptertest

// ContainsString reports whether values contains want exactly.
func ContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
