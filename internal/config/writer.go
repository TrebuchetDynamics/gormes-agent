package config

import configwriter "github.com/TrebuchetDynamics/gormes-agent/internal/config/configwriter"

// EnvPath returns the Gormes-native dotenv file path under GormesHome.
func EnvPath() string {
	return configwriter.EnvPath()
}

// IsSecretKey reports whether the user-supplied dotted key targets the
// dotenv file rather than config.toml. Mirrors the Hermes set_config_value
// classifier: known aliases, *_API_KEY, and *_TOKEN suffixes are secrets.
// Section-prefixed keys ending in `_token` (e.g. telegram.bot_token) are
// also routed to .env so raw secrets never land in config.toml.
func IsSecretKey(key string) bool {
	return configwriter.IsSecretKey(key)
}

// SecretEnvName returns the canonical environment variable name a secret
// alias persists under. For non-alias secret keys the upper-cased key is
// returned unchanged.
func SecretEnvName(key string) string {
	return configwriter.SecretEnvName(key)
}

// WriteTOMLValue persists a single dotted key/value pair into the TOML file
// at path. The dotted key may be a top-level field (interpreted as
// llm.<name> for the small Hermes-aliased set) or a `<section>.<field>`
// pair. Unknown top-level sections are rejected before any write so a typo
// cannot create an unbounded namespace.
func WriteTOMLValue(path, key, value string) error {
	return configwriter.WriteTOMLValue(path, key, value)
}

// WriteEnvValue persists a KEY=VALUE pair into the dotenv file at path.
// Existing lines for the same key are collapsed to one replacement;
// otherwise the pair is appended. Parent directories are created with 0o700
// and the dotenv file is written 0o600 so secrets stay operator-readable only.
func WriteEnvValue(path, key, value string) error {
	return configwriter.WriteEnvValue(path, key, value)
}

// EnsureConfigFile creates a root v2 TOML file when it does not exist. It is
// safe to call repeatedly and does not overwrite existing files.
func EnsureConfigFile(path string) error {
	return configwriter.EnsureConfigFile(path)
}

func readTOMLDoc(path string) (map[string]any, error) {
	return configwriter.ReadTOMLDoc(path)
}

func writeTOMLDoc(path string, doc map[string]any) error {
	return configwriter.WriteTOMLDoc(path, doc)
}

func writeTOMLAtomic(path string, doc map[string]any) error {
	return configwriter.WriteTOMLAtomic(path, doc)
}
