package environment

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/environment/passthrough"

// ProviderCredentialEnvBlocklist is the mutable provider credential blocklist
// used when new session registries are constructed. Existing registries keep a
// construction-time snapshot so later mutations cannot cross session bounds.
var ProviderCredentialEnvBlocklist = passthrough.ProviderCredentialEnvBlocklist

// EnvPassthroughRegistry is a session-scoped allowlist for environment
// variables that may pass through to sandboxed tools. It mirrors Hermes'
// ContextVar-backed registry with an explicit Go object so callers can keep
// separate sessions isolated and tests can prove no cross-session bleed.
type EnvPassthroughRegistry = passthrough.Registry

// NewEnvPassthroughRegistry creates a registry with config-sourced allowlist
// entries. Provider credentials in the configured list are ignored, matching
// Hermes' safety rule that operator config also cannot override sandbox
// credential scrubbing.
func NewEnvPassthroughRegistry(configured []string) *EnvPassthroughRegistry {
	return passthrough.NewRegistryWithBlocklist(configured, ProviderCredentialEnvBlocklist)
}
