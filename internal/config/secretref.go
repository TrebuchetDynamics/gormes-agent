package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"

type SecretRefSource = credentials.SecretRefSource

const (
	SecretRefSourceEnv  = credentials.SecretRefSourceEnv
	SecretRefSourceFile = credentials.SecretRefSourceFile
	SecretRefSourceExec = credentials.SecretRefSourceExec

	DefaultSecretProviderAlias = credentials.DefaultSecretProviderAlias
	SecretProviderModeJSON     = credentials.SecretProviderModeJSON
	SecretProviderModeSingle   = credentials.SecretProviderModeSingle

	SecretRefEvidenceResolved             = credentials.SecretRefEvidenceResolved
	SecretRefEvidenceMissing              = credentials.SecretRefEvidenceMissing
	SecretRefEvidenceInvalid              = credentials.SecretRefEvidenceInvalid
	SecretRefEvidenceProviderUnconfigured = credentials.SecretRefEvidenceProviderUnconfigured
	SecretRefEvidenceProviderMismatch     = credentials.SecretRefEvidenceProviderMismatch
	SecretRefEvidenceInsecurePath         = credentials.SecretRefEvidenceInsecurePath
	SecretRefEvidenceReadFailed           = credentials.SecretRefEvidenceReadFailed
	SecretRefEvidenceUnsupported          = credentials.SecretRefEvidenceUnsupported
)

type SecretRef = credentials.SecretRef
type SecretsCfg = credentials.SecretsCfg
type SecretProviderDefaults = credentials.SecretProviderDefaults
type SecretProviderCfg = credentials.SecretProviderCfg
type BitwardenSecretSourceCfg = credentials.BitwardenSecretSourceCfg
type SecretResolverConfig = credentials.SecretResolverConfig
type SecretRefEvidence = credentials.SecretRefEvidence
type SecretResolver = credentials.SecretResolver

func NewSecretResolver(cfg SecretResolverConfig) *SecretResolver {
	return credentials.NewSecretResolver(cfg)
}

func normalizeSecretRef(ref SecretRef) SecretRef {
	return credentials.NormalizeSecretRef(ref)
}

func normalizeSecretRefSource(source SecretRefSource) SecretRefSource {
	return credentials.NormalizeSecretRefSource(source)
}

func validateSecretRef(ref SecretRef) error {
	return credentials.ValidateSecretRef(ref)
}
