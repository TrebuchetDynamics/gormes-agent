package safety

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/securityaudit"

const (
	SecurityAuditCodeCompleted = securityaudit.SecurityAuditCodeCompleted

	SecurityAuditStatusPass = securityaudit.SecurityAuditStatusPass
	SecurityAuditStatusWarn = securityaudit.SecurityAuditStatusWarn
	SecurityAuditStatusFail = securityaudit.SecurityAuditStatusFail

	SecurityAuditCategoryGatewayAuth         = securityaudit.SecurityAuditCategoryGatewayAuth
	SecurityAuditCategoryStateIntegrity      = securityaudit.SecurityAuditCategoryStateIntegrity
	SecurityAuditCategoryChannelSecurity     = securityaudit.SecurityAuditCategoryChannelSecurity
	SecurityAuditCategoryShellBlocklist      = securityaudit.SecurityAuditCategoryShellBlocklist
	SecurityAuditCategoryFilesystemScoping   = securityaudit.SecurityAuditCategoryFilesystemScoping
	SecurityAuditCategoryCredentialRedaction = securityaudit.SecurityAuditCategoryCredentialRedaction
	SecurityAuditCategorySecretRefs          = securityaudit.SecurityAuditCategorySecretRefs

	SecurityAuditFindingGatewayAuthMissing       = securityaudit.SecurityAuditFindingGatewayAuthMissing
	SecurityAuditFindingGatewayProbeUnavailable  = securityaudit.SecurityAuditFindingGatewayProbeUnavailable
	SecurityAuditFindingStateFileMissing         = securityaudit.SecurityAuditFindingStateFileMissing
	SecurityAuditFindingStateFileInvalid         = securityaudit.SecurityAuditFindingStateFileInvalid
	SecurityAuditFindingChannelCredentialMissing = securityaudit.SecurityAuditFindingChannelCredentialMissing
	SecurityAuditFindingChannelScopeOpen         = securityaudit.SecurityAuditFindingChannelScopeOpen
	SecurityAuditFindingShellBlocklistGap        = securityaudit.SecurityAuditFindingShellBlocklistGap
	SecurityAuditFindingFilesystemScopeOpen      = securityaudit.SecurityAuditFindingFilesystemScopeOpen
	SecurityAuditFindingCredentialLeak           = securityaudit.SecurityAuditFindingCredentialLeak
	SecurityAuditFindingSecretRefUnavailable     = securityaudit.SecurityAuditFindingSecretRefUnavailable
	SecurityAuditFindingSecretRefUnsupported     = securityaudit.SecurityAuditFindingSecretRefUnsupported
	SecurityAuditFindingFixFailed                = securityaudit.SecurityAuditFindingFixFailed
	SecurityAuditFindingUnsafeFixSkipped         = securityaudit.SecurityAuditFindingUnsafeFixSkipped

	SecurityAuditFixFilePermissions            = securityaudit.SecurityAuditFixFilePermissions
	SecurityAuditFixGatewayAuthTokenGenerated  = securityaudit.SecurityAuditFixGatewayAuthTokenGenerated
	SecurityAuditFixGatewayAuthTokenGeneration = securityaudit.SecurityAuditFixGatewayAuthTokenGeneration
)

type SecurityAuditRequest = securityaudit.SecurityAuditRequest
type SecurityAuditGatewayAuth = securityaudit.SecurityAuditGatewayAuth
type SecurityAuditProbe = securityaudit.SecurityAuditProbe
type SecurityAuditState = securityaudit.SecurityAuditState
type SecurityAuditStateFile = securityaudit.SecurityAuditStateFile
type SecurityAuditChannel = securityaudit.SecurityAuditChannel
type SecurityAuditFilesystem = securityaudit.SecurityAuditFilesystem
type SecurityAuditCredentialRedaction = securityaudit.SecurityAuditCredentialRedaction
type SecurityAuditSecretRef = securityaudit.SecurityAuditSecretRef
type SecurityAuditFixCandidate = securityaudit.SecurityAuditFixCandidate
type SecurityAuditFixApplier = securityaudit.SecurityAuditFixApplier
type SecurityAuditTokenGenerator = securityaudit.SecurityAuditTokenGenerator
type SecurityAuditResult = securityaudit.SecurityAuditResult
type SecurityAuditSummary = securityaudit.SecurityAuditSummary
type SecurityAuditCategoryResult = securityaudit.SecurityAuditCategoryResult
type SecurityAuditFinding = securityaudit.SecurityAuditFinding
type SecurityAuditFix = securityaudit.SecurityAuditFix

func AuditSecurity(req SecurityAuditRequest) SecurityAuditResult {
	return securityaudit.AuditSecurity(req)
}

func RedactSecurityAuditText(text string, secrets []string) string {
	return securityaudit.RedactSecurityAuditText(text, secrets)
}
