package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"

const (
	CredentialAuthAPIKey = credentials.CredentialAuthAPIKey
	CredentialAuthOAuth  = credentials.CredentialAuthOAuth

	CredentialStatusOK        = credentials.CredentialStatusOK
	CredentialStatusExhausted = credentials.CredentialStatusExhausted

	CredentialPoolEvidenceLoaded        = credentials.CredentialPoolEvidenceLoaded
	CredentialPoolEvidenceEmpty         = credentials.CredentialPoolEvidenceEmpty
	CredentialPoolEvidenceCorrupt       = credentials.CredentialPoolEvidenceCorrupt
	CredentialPoolEvidenceSelected      = credentials.CredentialPoolEvidenceSelected
	CredentialPoolEvidenceUnavailable   = credentials.CredentialPoolEvidenceUnavailable
	CredentialPoolEvidenceExhausted     = credentials.CredentialPoolEvidenceExhausted
	CredentialPoolEvidenceLeaseAcquired = credentials.CredentialPoolEvidenceLeaseAcquired
	CredentialPoolEvidenceLeaseReleased = credentials.CredentialPoolEvidenceLeaseReleased

	CredentialPoolStrategyFillFirst  = credentials.CredentialPoolStrategyFillFirst
	CredentialPoolStrategyRoundRobin = credentials.CredentialPoolStrategyRoundRobin
	CredentialPoolStrategyLeastUsed  = credentials.CredentialPoolStrategyLeastUsed
	CredentialPoolStrategyRandom     = credentials.CredentialPoolStrategyRandom

	NousOAuthProvider         = credentials.NousOAuthProvider
	NousOAuthDeviceCodeSource = credentials.NousOAuthDeviceCodeSource
)

type CredentialPoolStrategy = credentials.CredentialPoolStrategy
type CredentialPoolOptions = credentials.CredentialPoolOptions
type PooledCredential = credentials.PooledCredential
type RedactedCredentialStatus = credentials.RedactedCredentialStatus
type CredentialPoolStatus = credentials.CredentialPoolStatus
type CredentialPoolEvidence = credentials.CredentialPoolEvidence
type CredentialExhaustion = credentials.CredentialExhaustion
type CredentialPoolError = credentials.CredentialPoolError
type CredentialPool = credentials.CredentialPool
type NousOAuthCredentials = credentials.NousOAuthCredentials

func SaveCredentialPoolEntries(opts CredentialPoolOptions, entries []PooledCredential) error {
	return credentials.SaveCredentialPoolEntries(opts, entries)
}

func SaveNousOAuthCredentials(opts CredentialPoolOptions, creds NousOAuthCredentials) (PooledCredential, error) {
	return credentials.SaveNousOAuthCredentials(opts, creds)
}

func LoadCredentialPool(opts CredentialPoolOptions) (*CredentialPool, CredentialPoolEvidence, error) {
	return credentials.LoadCredentialPool(opts)
}

func ListCredentialPoolProviders(opts CredentialPoolOptions) ([]string, error) {
	return credentials.ListCredentialPoolProviders(opts)
}

func LoadNousOAuthCredentials(opts CredentialPoolOptions) (NousOAuthCredentials, error) {
	return credentials.LoadNousOAuthCredentials(opts)
}
