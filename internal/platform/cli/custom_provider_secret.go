package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/providerconfig"

// CustomProviderRef is the pure input model describing a custom provider's
// credential configuration as it appears on disk or in memory.
type CustomProviderRef = providerconfig.CustomProviderRef

// CustomProviderResolution carries the result of resolving a custom provider
// credential. EffectiveSecret is the cleartext used for outbound calls;
// PersistAsRef is the value that should be written back to config so that
// references (env templates) are never replaced with plaintext. Evidence
// labels how the resolution was reached so callers can branch without
// inspecting strings for "${" prefixes.
type CustomProviderResolution = providerconfig.CustomProviderResolution

// ErrCustomProviderEnvUnset signals that an env-template ${VAR} reference was
// supplied but the named variable is missing or empty in the environment map.
var ErrCustomProviderEnvUnset = providerconfig.ErrCustomProviderEnvUnset

// ErrCustomProviderCredentialMissing signals that neither APIKey nor KeyEnv
// was supplied, so no credential could be resolved.
var ErrCustomProviderCredentialMissing = providerconfig.ErrCustomProviderCredentialMissing

// ResolveCustomProviderSecret resolves a custom provider credential without
// touching the filesystem, network, or process environment. The function
// preserves env-template references (`${VAR}`) in PersistAsRef so callers can
// persist the reference back to config without leaking plaintext, while still
// returning the resolved EffectiveSecret for outbound calls.
func ResolveCustomProviderSecret(ref CustomProviderRef, env map[string]string) (CustomProviderResolution, error) {
	return providerconfig.ResolveCustomProviderSecret(ref, env)
}
