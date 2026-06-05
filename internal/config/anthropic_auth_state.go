package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"

const (
	AnthropicProvider = credentials.AnthropicProvider

	AnthropicAuthStatusAuthorized      = credentials.AnthropicAuthStatusAuthorized
	AnthropicAuthStatusMissing         = credentials.AnthropicAuthStatusMissing
	AnthropicAuthStatusReloginRequired = credentials.AnthropicAuthStatusReloginRequired
	AnthropicAuthStatusCorrupt         = credentials.AnthropicAuthStatusCorrupt

	AnthropicOAuthEvidenceKeychainSelected    = credentials.AnthropicOAuthEvidenceKeychainSelected
	AnthropicOAuthEvidenceJSONSelected        = credentials.AnthropicOAuthEvidenceJSONSelected
	AnthropicOAuthEvidenceMissing             = credentials.AnthropicOAuthEvidenceMissing
	AnthropicOAuthEvidenceKeychainUnavailable = credentials.AnthropicOAuthEvidenceKeychainUnavailable
	AnthropicOAuthEvidenceCorruptBackup       = credentials.AnthropicOAuthEvidenceCorruptBackup
	AnthropicOAuthEvidenceStaleOAuth          = credentials.AnthropicOAuthEvidenceStaleOAuth

	AnthropicOAuthSourceMacOSKeychain = credentials.AnthropicOAuthSourceMacOSKeychain
	AnthropicOAuthSourceJSONFile      = credentials.AnthropicOAuthSourceJSONFile
)

type AnthropicKeychainReader = credentials.AnthropicKeychainReader
type AnthropicAuthStateStoreOptions = credentials.AnthropicAuthStateStoreOptions
type AnthropicAuthStateStore = credentials.AnthropicAuthStateStore
type AnthropicClaudeCredentials = credentials.AnthropicClaudeCredentials
type AnthropicAuthStatus = credentials.AnthropicAuthStatus

func NewAnthropicAuthStateStore(opts AnthropicAuthStateStoreOptions) *AnthropicAuthStateStore {
	return credentials.NewAnthropicAuthStateStore(opts)
}
