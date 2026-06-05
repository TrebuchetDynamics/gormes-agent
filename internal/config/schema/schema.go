package schema

import (
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/profilestorage"
)

// CurrentConfigVersion is the schema version this binary writes + accepts.
// When a breaking change to the TOML schema lands, bump this constant and add
// a migration in runMigrations() so older files stay readable.
const CurrentConfigVersion = 2

// allowedSections is the closed set of top-level TOML tables this binary
// writes. Hermes parity work that introduces a new section must add it here.
// Keep in sync with the Config struct in config.go.
var allowedSections = map[string]struct{}{
	"hermes":      {},
	"router":      {},
	"profiles":    {},
	"credentials": {},
	"runtime":     {},
	"tts":         {},
	"image_gen":   {},
	"terminal":    {},
	"gateway":     {},
	"tui":         {},
	"input":       {},
	"voice":       {},
	"telegram":    {},
	"discord":     {},
	"slack":       {},
	"yuanbao":     {},
	"web":         {},
	"navivox":     {},
	"browser":     {},
	"security":    {},
	"secrets":     {},
	"agents":      {},
	"bindings":    {},
	"cron":        {},
	"skills":      {},
	"delegation":  {},
	"goncho":      {},
	"display":     {},
	"updates":     {},
}

func AllowsSection(section string) bool {
	_, ok := allowedSections[strings.TrimSpace(section)]
	return ok
}

func AllowedSectionsList() string {
	names := make([]string, 0, len(allowedSections))
	for n := range allowedSections {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func DefaultDocumentV2() map[string]any {
	return map[string]any{
		"config_version": int64(CurrentConfigVersion),
		"profiles": map[string]any{
			profilestorage.DefaultProfileID: map[string]any{
				"enabled": true,
				"name":    "",
			},
		},
	}
}

func ReadVersion(raw map[string]any) int {
	v, ok := raw["config_version"]
	if !ok {
		v, ok = raw["_config_version"]
	}
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return int(n)
	case int:
		return n
	case float64:
		return int(n)
	}
	return 0
}

func HasMainProfile(raw map[string]any) bool {
	profiles, ok := raw["profiles"].(map[string]any)
	if !ok {
		return false
	}
	main, ok := profiles[profilestorage.DefaultProfileID].(map[string]any)
	if !ok {
		return false
	}
	_, hasEnabled := main["enabled"]
	_, hasName := main["name"]
	return hasEnabled && hasName
}

func EnsureMainProfile(raw map[string]any) {
	profiles, ok := raw["profiles"].(map[string]any)
	if !ok {
		profiles = map[string]any{}
	}
	main, ok := profiles[profilestorage.DefaultProfileID].(map[string]any)
	if !ok {
		main = map[string]any{}
	}
	if _, ok := main["enabled"]; !ok {
		main["enabled"] = true
	}
	if _, ok := main["name"]; !ok {
		main["name"] = ""
	}
	profiles[profilestorage.DefaultProfileID] = main
	raw["profiles"] = profiles
}
