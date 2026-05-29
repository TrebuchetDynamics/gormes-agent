package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"

const (
	CodexOAuthProvider = credentials.CodexOAuthProvider

	CodexOAuthStatusAuthorized         = credentials.CodexOAuthStatusAuthorized
	CodexOAuthStatusMissing            = credentials.CodexOAuthStatusMissing
	CodexOAuthStatusReloginRequired    = credentials.CodexOAuthStatusReloginRequired
	CodexOAuthStatusImportNotRequested = credentials.CodexOAuthStatusImportNotRequested
	CodexOAuthStatusImportRejected     = credentials.CodexOAuthStatusImportRejected
	CodexOAuthStatusCorrupt            = credentials.CodexOAuthStatusCorrupt

	CodexOAuthEvidenceSaved         = credentials.CodexOAuthEvidenceSaved
	CodexOAuthEvidenceMissing       = credentials.CodexOAuthEvidenceMissing
	CodexOAuthEvidenceImportExpired = credentials.CodexOAuthEvidenceImportExpired
	CodexOAuthEvidenceImportSkipped = credentials.CodexOAuthEvidenceImportSkipped
	CodexOAuthEvidenceImportMissing = credentials.CodexOAuthEvidenceImportMissing
	CodexOAuthEvidenceCorrupt       = credentials.CodexOAuthEvidenceCorrupt

	CodexOAuthSourceDeviceCode     = credentials.CodexOAuthSourceDeviceCode
	CodexOAuthSourceCodexCLIImport = credentials.CodexOAuthSourceCodexCLIImport
)

type CodexOAuthStateStoreOptions = credentials.CodexOAuthStateStoreOptions
type CodexOAuthStateStore = credentials.CodexOAuthStateStore
type CodexOAuthTokens = credentials.CodexOAuthTokens
type CodexOAuthAuthStatus = credentials.CodexOAuthAuthStatus
type CodexCLIImportRequest = credentials.CodexCLIImportRequest

func NewCodexOAuthStateStore(opts CodexOAuthStateStoreOptions) *CodexOAuthStateStore {
	return credentials.NewCodexOAuthStateStore(opts)
}
