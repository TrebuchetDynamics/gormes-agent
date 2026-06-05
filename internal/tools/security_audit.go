package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety"

const (
	SecurityAuditCodeCompleted = safety.SecurityAuditCodeCompleted

	SecurityAuditStatusPass = safety.SecurityAuditStatusPass
	SecurityAuditStatusWarn = safety.SecurityAuditStatusWarn
	SecurityAuditStatusFail = safety.SecurityAuditStatusFail

	SecurityAuditCategoryGatewayAuth         = safety.SecurityAuditCategoryGatewayAuth
	SecurityAuditCategoryStateIntegrity      = safety.SecurityAuditCategoryStateIntegrity
	SecurityAuditCategoryChannelSecurity     = safety.SecurityAuditCategoryChannelSecurity
	SecurityAuditCategoryShellBlocklist      = safety.SecurityAuditCategoryShellBlocklist
	SecurityAuditCategoryFilesystemScoping   = safety.SecurityAuditCategoryFilesystemScoping
	SecurityAuditCategoryCredentialRedaction = safety.SecurityAuditCategoryCredentialRedaction
	SecurityAuditCategorySecretRefs          = safety.SecurityAuditCategorySecretRefs

	SecurityAuditFindingGatewayAuthMissing       = safety.SecurityAuditFindingGatewayAuthMissing
	SecurityAuditFindingGatewayProbeUnavailable  = safety.SecurityAuditFindingGatewayProbeUnavailable
	SecurityAuditFindingStateFileMissing         = safety.SecurityAuditFindingStateFileMissing
	SecurityAuditFindingStateFileInvalid         = safety.SecurityAuditFindingStateFileInvalid
	SecurityAuditFindingChannelCredentialMissing = safety.SecurityAuditFindingChannelCredentialMissing
	SecurityAuditFindingChannelScopeOpen         = safety.SecurityAuditFindingChannelScopeOpen
	SecurityAuditFindingShellBlocklistGap        = safety.SecurityAuditFindingShellBlocklistGap
	SecurityAuditFindingFilesystemScopeOpen      = safety.SecurityAuditFindingFilesystemScopeOpen
	SecurityAuditFindingCredentialLeak           = safety.SecurityAuditFindingCredentialLeak
	SecurityAuditFindingSecretRefUnavailable     = safety.SecurityAuditFindingSecretRefUnavailable
	SecurityAuditFindingSecretRefUnsupported     = safety.SecurityAuditFindingSecretRefUnsupported
	SecurityAuditFindingFixFailed                = safety.SecurityAuditFindingFixFailed
	SecurityAuditFindingUnsafeFixSkipped         = safety.SecurityAuditFindingUnsafeFixSkipped

	SecurityAuditFixFilePermissions            = safety.SecurityAuditFixFilePermissions
	SecurityAuditFixGatewayAuthTokenGenerated  = safety.SecurityAuditFixGatewayAuthTokenGenerated
	SecurityAuditFixGatewayAuthTokenGeneration = safety.SecurityAuditFixGatewayAuthTokenGeneration
)

type SecurityAuditRequest = safety.SecurityAuditRequest
type SecurityAuditGatewayAuth = safety.SecurityAuditGatewayAuth
type SecurityAuditProbe = safety.SecurityAuditProbe
type SecurityAuditState = safety.SecurityAuditState
type SecurityAuditStateFile = safety.SecurityAuditStateFile
type SecurityAuditChannel = safety.SecurityAuditChannel
type SecurityAuditFilesystem = safety.SecurityAuditFilesystem
type SecurityAuditCredentialRedaction = safety.SecurityAuditCredentialRedaction
type SecurityAuditSecretRef = safety.SecurityAuditSecretRef
type SecurityAuditFixCandidate = safety.SecurityAuditFixCandidate
type SecurityAuditFixApplier = safety.SecurityAuditFixApplier
type SecurityAuditTokenGenerator = safety.SecurityAuditTokenGenerator
type SecurityAuditResult = safety.SecurityAuditResult
type SecurityAuditSummary = safety.SecurityAuditSummary
type SecurityAuditCategoryResult = safety.SecurityAuditCategoryResult
type SecurityAuditFinding = safety.SecurityAuditFinding
type SecurityAuditFix = safety.SecurityAuditFix

func AuditSecurity(req SecurityAuditRequest) SecurityAuditResult {
	return safety.AuditSecurity(req)
}

func RedactSecurityAuditText(text string, secrets []string) string {
	return safety.RedactSecurityAuditText(text, secrets)
}
