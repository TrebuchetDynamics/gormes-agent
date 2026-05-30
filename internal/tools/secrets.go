package tools

import (
	"context"

	secretstools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/secrets"
)

const (
	SecretRefSourceEnv  = secretstools.SecretRefSourceEnv
	SecretRefSourceFile = secretstools.SecretRefSourceFile
	SecretRefSourceExec = secretstools.SecretRefSourceExec

	DefaultSecretProviderAlias = secretstools.DefaultSecretProviderAlias

	SecretsEvidenceApplied      = secretstools.SecretsEvidenceApplied
	SecretsEvidenceConfigured   = secretstools.SecretsEvidenceConfigured
	SecretsEvidenceReloaded     = secretstools.SecretsEvidenceReloaded
	SecretsEvidenceUnavailable  = secretstools.SecretsEvidenceUnavailable
	SecretsEvidenceAuditPassed  = secretstools.SecretsEvidenceAuditPassed
	SecretsEvidenceAuditFinding = secretstools.SecretsEvidenceAuditFinding

	SecretsFindingPlaintext       = secretstools.SecretsFindingPlaintext
	SecretsFindingUnresolvedRef   = secretstools.SecretsFindingUnresolvedRef
	SecretsFindingPrecedenceDrift = secretstools.SecretsFindingPrecedenceDrift
	SecretsFindingInvalidTarget   = secretstools.SecretsFindingInvalidTarget
)

type SecretRef = secretstools.SecretRef
type SecretRefEvidence = secretstools.SecretRefEvidence
type SecretResolver = secretstools.SecretResolver
type SecretResolverFunc = secretstools.SecretResolverFunc
type SecretsPlan = secretstools.SecretsPlan
type SecretTarget = secretstools.SecretTarget
type SecretsRuntimeEntry = secretstools.SecretsRuntimeEntry
type SecretsRuntimeSnapshot = secretstools.SecretsRuntimeSnapshot
type SecretsRuntimeControllerConfig = secretstools.SecretsRuntimeControllerConfig
type SecretsRuntimeController = secretstools.SecretsRuntimeController
type SecretsApplyResult = secretstools.SecretsApplyResult
type SecretsAuditRequest = secretstools.SecretsAuditRequest
type SecretsAuditResult = secretstools.SecretsAuditResult
type SecretsAuditFinding = secretstools.SecretsAuditFinding
type SecretsConfigureRequest = secretstools.SecretsConfigureRequest
type SecretsConfigureResult = secretstools.SecretsConfigureResult

func NewSecretsRuntimeController(cfg SecretsRuntimeControllerConfig) *SecretsRuntimeController {
	return secretstools.NewSecretsRuntimeController(cfg)
}

func AuditSecrets(ctx context.Context, req SecretsAuditRequest) SecretsAuditResult {
	return secretstools.AuditSecrets(ctx, req)
}

func ConfigureSecretRef(ctx context.Context, req SecretsConfigureRequest) (SecretsConfigureResult, error) {
	return secretstools.ConfigureSecretRef(ctx, req)
}
