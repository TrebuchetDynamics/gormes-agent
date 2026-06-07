package credentials

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials/anthropicauth"

const (
	AnthropicProvider = anthropicauth.AnthropicProvider

	AnthropicAuthStatusAuthorized      = anthropicauth.AnthropicAuthStatusAuthorized
	AnthropicAuthStatusMissing         = anthropicauth.AnthropicAuthStatusMissing
	AnthropicAuthStatusReloginRequired = anthropicauth.AnthropicAuthStatusReloginRequired
	AnthropicAuthStatusCorrupt         = anthropicauth.AnthropicAuthStatusCorrupt

	AnthropicOAuthEvidenceKeychainSelected    = anthropicauth.AnthropicOAuthEvidenceKeychainSelected
	AnthropicOAuthEvidenceJSONSelected        = anthropicauth.AnthropicOAuthEvidenceJSONSelected
	AnthropicOAuthEvidenceMissing             = anthropicauth.AnthropicOAuthEvidenceMissing
	AnthropicOAuthEvidenceKeychainUnavailable = anthropicauth.AnthropicOAuthEvidenceKeychainUnavailable
	AnthropicOAuthEvidenceCorruptBackup       = anthropicauth.AnthropicOAuthEvidenceCorruptBackup
	AnthropicOAuthEvidenceStaleOAuth          = anthropicauth.AnthropicOAuthEvidenceStaleOAuth

	AnthropicOAuthSourceMacOSKeychain = anthropicauth.AnthropicOAuthSourceMacOSKeychain
	AnthropicOAuthSourceJSONFile      = anthropicauth.AnthropicOAuthSourceJSONFile
)

type AnthropicKeychainReader = anthropicauth.AnthropicKeychainReader
type AnthropicAuthStateStoreOptions = anthropicauth.AnthropicAuthStateStoreOptions
type AnthropicAuthStateStore = anthropicauth.AnthropicAuthStateStore
type AnthropicClaudeCredentials = anthropicauth.AnthropicClaudeCredentials
type AnthropicAuthStatus = anthropicauth.AnthropicAuthStatus

func NewAnthropicAuthStateStore(opts AnthropicAuthStateStoreOptions) *AnthropicAuthStateStore {
	return anthropicauth.NewAnthropicAuthStateStore(opts)
}
