package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"

const (
	TokenVaultSourceRegistered = credentials.TokenVaultSourceRegistered
	TokenVaultSourceConfig     = credentials.TokenVaultSourceConfig
)

type TokenVaultReason = credentials.TokenVaultReason

const (
	TokenVaultReasonEmptyPath     = credentials.TokenVaultReasonEmptyPath
	TokenVaultReasonAbsolutePath  = credentials.TokenVaultReasonAbsolutePath
	TokenVaultReasonTraversal     = credentials.TokenVaultReasonTraversal
	TokenVaultReasonMissing       = credentials.TokenVaultReasonMissing
	TokenVaultReasonUnreadable    = credentials.TokenVaultReasonUnreadable
	TokenVaultReasonSymlinkEscape = credentials.TokenVaultReasonSymlinkEscape
)

type TokenVaultOptions = credentials.TokenVaultOptions
type CredentialFileMount = credentials.CredentialFileMount
type TokenVaultEvidence = credentials.TokenVaultEvidence
type TokenVaultError = credentials.TokenVaultError
type TokenVault = credentials.TokenVault
type TokenVaultConfigResult = credentials.TokenVaultConfigResult

func AsTokenVaultError(err error, target **TokenVaultError) bool {
	return credentials.AsTokenVaultError(err, target)
}

func NewTokenVault(opts TokenVaultOptions) (*TokenVault, error) {
	return credentials.NewTokenVault(opts)
}
