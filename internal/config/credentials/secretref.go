package credentials

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials/secretref"

type SecretRefSource = secretref.SecretRefSource

const (
	SecretRefSourceEnv  = secretref.SecretRefSourceEnv
	SecretRefSourceFile = secretref.SecretRefSourceFile
	SecretRefSourceExec = secretref.SecretRefSourceExec

	DefaultSecretProviderAlias = secretref.DefaultSecretProviderAlias
	SecretProviderModeJSON     = secretref.SecretProviderModeJSON
	SecretProviderModeSingle   = secretref.SecretProviderModeSingle

	SecretRefEvidenceResolved             = secretref.SecretRefEvidenceResolved
	SecretRefEvidenceMissing              = secretref.SecretRefEvidenceMissing
	SecretRefEvidenceInvalid              = secretref.SecretRefEvidenceInvalid
	SecretRefEvidenceProviderUnconfigured = secretref.SecretRefEvidenceProviderUnconfigured
	SecretRefEvidenceProviderMismatch     = secretref.SecretRefEvidenceProviderMismatch
	SecretRefEvidenceInsecurePath         = secretref.SecretRefEvidenceInsecurePath
	SecretRefEvidenceReadFailed           = secretref.SecretRefEvidenceReadFailed
	SecretRefEvidenceUnsupported          = secretref.SecretRefEvidenceUnsupported
)

type SecretRef = secretref.SecretRef
type SecretsCfg = secretref.SecretsCfg
type BitwardenSecretSourceCfg = secretref.BitwardenSecretSourceCfg
type SecretProviderDefaults = secretref.SecretProviderDefaults
type SecretProviderCfg = secretref.SecretProviderCfg
type SecretResolverConfig = secretref.SecretResolverConfig
type SecretRefEvidence = secretref.SecretRefEvidence
type SecretResolver = secretref.SecretResolver

func NewSecretResolver(cfg SecretResolverConfig) *SecretResolver {
	return secretref.NewSecretResolver(cfg)
}

// NormalizeSecretRef normalizes SecretRef metadata for compatibility shims and callers.
func NormalizeSecretRef(ref SecretRef) SecretRef {
	return secretref.NormalizeSecretRef(ref)
}

// NormalizeSecretRefSource normalizes a SecretRef source token.
func NormalizeSecretRefSource(source SecretRefSource) SecretRefSource {
	return secretref.NormalizeSecretRefSource(source)
}

// ValidateSecretRef validates a SecretRef without resolving secret material.
func ValidateSecretRef(ref SecretRef) error {
	return secretref.ValidateSecretRef(ref)
}
