package gateway

import gatewaytokenlock "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/tokenlock"

const tokenLockKind = gatewaytokenlock.TokenLockKind

// TokenLockStatus is operator-facing evidence for credential-scoped gateway
// lock decisions.
type TokenLockStatus = gatewaytokenlock.TokenLockStatus

const (
	TokenLockStatusAcquired               = gatewaytokenlock.TokenLockStatusAcquired
	TokenLockStatusHeld                   = gatewaytokenlock.TokenLockStatusHeld
	TokenLockStatusStaleCleared           = gatewaytokenlock.TokenLockStatusStaleCleared
	TokenLockStatusCredentialHashMismatch = gatewaytokenlock.TokenLockStatusCredentialHashMismatch
	TokenLockStatusReleased               = gatewaytokenlock.TokenLockStatusReleased
	TokenLockStatusReleaseFailed          = gatewaytokenlock.TokenLockStatusReleaseFailed
)

var (
	ErrTokenLockHeld                   = gatewaytokenlock.ErrTokenLockHeld
	ErrTokenLockCredentialHashMismatch = gatewaytokenlock.ErrTokenLockCredentialHashMismatch
	ErrTokenLockReleaseFailed          = gatewaytokenlock.ErrTokenLockReleaseFailed
)

// TokenLockRequest describes the external credential identity a gateway
// process wants to reserve.
type TokenLockRequest = gatewaytokenlock.TokenLockRequest

// TokenLockEvidence is safe to persist in runtime status JSON. It carries only
// platform names, paths, process identity, and non-reversible credential hashes.
type TokenLockEvidence = gatewaytokenlock.TokenLockEvidence

// TokenLockStore manages credential-scoped gateway lock records under one
// machine-local lock directory.
type TokenLockStore = gatewaytokenlock.TokenLockStore

// TokenScopedGatewayLock represents a lock record owned by the current
// process according to PID and process start-time evidence.
type TokenScopedGatewayLock = gatewaytokenlock.TokenScopedGatewayLock

// NewTokenLockStore returns a JSON-file-backed token lock store.
func NewTokenLockStore(dir string) *TokenLockStore {
	return gatewaytokenlock.NewTokenLockStore(dir)
}

// TokenCredentialHash returns the non-reversible credential scope hash used in
// lock filenames and status evidence.
func TokenCredentialHash(credential string) string {
	return gatewaytokenlock.TokenCredentialHash(credential)
}
