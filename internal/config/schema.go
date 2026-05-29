package config

import (
	"sort"
	"strings"
)

// CurrentConfigVersion is the schema version this binary writes + accepts.
// When a breaking change to the TOML schema lands, bump this constant and
// add a migration in runMigrations() so older files stay readable.
const CurrentConfigVersion = 2

// configSchemaAllowedSections is the closed set of top-level TOML tables this
// binary writes. Hermes parity work that introduces a new section must add it
// here. Keep in sync with the Config struct in config.go.
var configSchemaAllowedSections = map[string]struct{}{
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

func configSchemaAllowsSection(section string) bool {
	_, ok := configSchemaAllowedSections[strings.TrimSpace(section)]
	return ok
}

func allowedSectionsList() string {
	names := make([]string, 0, len(configSchemaAllowedSections))
	for n := range configSchemaAllowedSections {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func DefaultConfigDocumentV2() map[string]any {
	return map[string]any{
		"config_version": int64(CurrentConfigVersion),
		"profiles": map[string]any{
			DefaultProfileID: map[string]any{
				"enabled": true,
				"name":    "",
			},
		},
	}
}

func readConfigVersion(raw map[string]any) int {
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

func hasMainProfile(raw map[string]any) bool {
	profiles, ok := raw["profiles"].(map[string]any)
	if !ok {
		return false
	}
	main, ok := profiles[DefaultProfileID].(map[string]any)
	if !ok {
		return false
	}
	_, hasEnabled := main["enabled"]
	_, hasName := main["name"]
	return hasEnabled && hasName
}

func ensureMainProfile(raw map[string]any) {
	profiles, ok := raw["profiles"].(map[string]any)
	if !ok {
		profiles = map[string]any{}
	}
	main, ok := profiles[DefaultProfileID].(map[string]any)
	if !ok {
		main = map[string]any{}
	}
	if _, ok := main["enabled"]; !ok {
		main["enabled"] = true
	}
	if _, ok := main["name"]; !ok {
		main["name"] = ""
	}
	profiles[DefaultProfileID] = main
	raw["profiles"] = profiles
}
