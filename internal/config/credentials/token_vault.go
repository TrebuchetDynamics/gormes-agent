package credentials

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials/tokenvault"

const (
	TokenVaultSourceRegistered = tokenvault.TokenVaultSourceRegistered
	TokenVaultSourceConfig     = tokenvault.TokenVaultSourceConfig
)

type TokenVaultReason = tokenvault.TokenVaultReason

const (
	TokenVaultReasonEmptyPath     = tokenvault.TokenVaultReasonEmptyPath
	TokenVaultReasonAbsolutePath  = tokenvault.TokenVaultReasonAbsolutePath
	TokenVaultReasonTraversal     = tokenvault.TokenVaultReasonTraversal
	TokenVaultReasonMissing       = tokenvault.TokenVaultReasonMissing
	TokenVaultReasonUnreadable    = tokenvault.TokenVaultReasonUnreadable
	TokenVaultReasonSymlinkEscape = tokenvault.TokenVaultReasonSymlinkEscape
)

type TokenVaultOptions = tokenvault.TokenVaultOptions
type CredentialFileMount = tokenvault.CredentialFileMount
type TokenVaultEvidence = tokenvault.TokenVaultEvidence
type TokenVaultError = tokenvault.TokenVaultError
type TokenVault = tokenvault.TokenVault
type TokenVaultConfigResult = tokenvault.TokenVaultConfigResult

func AsTokenVaultError(err error, target **TokenVaultError) bool {
	return tokenvault.AsTokenVaultError(err, target)
}

func NewTokenVault(opts TokenVaultOptions) (*TokenVault, error) {
	return tokenvault.NewTokenVault(opts)
}
