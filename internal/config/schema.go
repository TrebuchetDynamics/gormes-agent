package config

import schemaconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/schema"

// CurrentConfigVersion is the schema version this binary writes + accepts.
// When a breaking change to the TOML schema lands, bump this constant and
// add a migration in runMigrations() so older files stay readable.
const CurrentConfigVersion = schemaconfig.CurrentConfigVersion

func configSchemaAllowsSection(section string) bool {
	return schemaconfig.AllowsSection(section)
}

func allowedSectionsList() string {
	return schemaconfig.AllowedSectionsList()
}

func DefaultConfigDocumentV2() map[string]any {
	return schemaconfig.DefaultDocumentV2()
}

func readConfigVersion(raw map[string]any) int {
	return schemaconfig.ReadVersion(raw)
}

func hasMainProfile(raw map[string]any) bool {
	return schemaconfig.HasMainProfile(raw)
}

func ensureMainProfile(raw map[string]any) {
	schemaconfig.EnsureMainProfile(raw)
}
