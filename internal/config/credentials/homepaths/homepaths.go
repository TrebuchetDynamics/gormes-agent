package homepaths

import (
	"os"
	"path/filepath"
	"strings"
)

// GormesHome resolves the local Gormes home from GORMES_HOME or ~/.gormes.
func GormesHome() string {
	if v := strings.TrimSpace(os.Getenv("GORMES_HOME")); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gormes")
}

// BaseHome returns the shared base home when the current home is scoped to a profile.
func BaseHome() string {
	return BaseHomeFor(GormesHome())
}

// BaseHomeFor returns the parent Gormes home for .../profiles/<id> homes.
func BaseHomeFor(current string) string {
	clean := filepath.Clean(strings.TrimSpace(current))
	if clean == "." || clean == string(filepath.Separator) {
		return current
	}
	if filepath.Base(filepath.Dir(clean)) == "profiles" {
		return filepath.Dir(filepath.Dir(clean))
	}
	return current
}
