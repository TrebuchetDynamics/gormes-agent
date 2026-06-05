package gateway

import (
	"strings"

	gatewaypairing "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/pairing"
)

// PairingPlatformState is the operator-facing per-platform pairing state.
type PairingPlatformState = gatewaypairing.PairingPlatformState

const (
	PairingPlatformStatePaired   PairingPlatformState = gatewaypairing.PairingPlatformStatePaired
	PairingPlatformStateUnpaired PairingPlatformState = gatewaypairing.PairingPlatformStateUnpaired
)

// PairingDegradedReason classifies read-only pairing-state degradation.
type PairingDegradedReason = gatewaypairing.PairingDegradedReason

const (
	PairingDegradedMissing          PairingDegradedReason = gatewaypairing.PairingDegradedMissing
	PairingDegradedCorrupt          PairingDegradedReason = gatewaypairing.PairingDegradedCorrupt
	PairingDegradedPermissionDenied PairingDegradedReason = gatewaypairing.PairingDegradedPermissionDenied
	PairingDegradedReadFailed       PairingDegradedReason = gatewaypairing.PairingDegradedReadFailed
	PairingDegradedRateLimited      PairingDegradedReason = gatewaypairing.PairingDegradedRateLimited
	PairingDegradedMaxPending       PairingDegradedReason = gatewaypairing.PairingDegradedMaxPending
	PairingDegradedExpired          PairingDegradedReason = gatewaypairing.PairingDegradedExpired
	PairingDegradedLockedOut        PairingDegradedReason = gatewaypairing.PairingDegradedLockedOut
	PairingDegradedAllowlistDenied  PairingDegradedReason = gatewaypairing.PairingDegradedAllowlistDenied
	PairingDegradedUnresolvedUser   PairingDegradedReason = gatewaypairing.PairingDegradedUnresolvedUser
)

// PairingCodeStatus is the state transition produced by a pairing-code request.
type PairingCodeStatus = gatewaypairing.PairingCodeStatus

const (
	PairingCodeIssued          PairingCodeStatus = gatewaypairing.PairingCodeIssued
	PairingCodeRateLimited     PairingCodeStatus = gatewaypairing.PairingCodeRateLimited
	PairingCodeMaxPending      PairingCodeStatus = gatewaypairing.PairingCodeMaxPending
	PairingCodeLockedOut       PairingCodeStatus = gatewaypairing.PairingCodeLockedOut
	PairingCodeAllowlistDenied PairingCodeStatus = gatewaypairing.PairingCodeAllowlistDenied
	PairingCodeUnresolvedUser  PairingCodeStatus = gatewaypairing.PairingCodeUnresolvedUser
)

// PairingApprovalStatus is the state transition produced by an approval
// attempt against a code.
type PairingApprovalStatus = gatewaypairing.PairingApprovalStatus

const (
	PairingApprovalApproved  PairingApprovalStatus = gatewaypairing.PairingApprovalApproved
	PairingApprovalInvalid   PairingApprovalStatus = gatewaypairing.PairingApprovalInvalid
	PairingApprovalExpired   PairingApprovalStatus = gatewaypairing.PairingApprovalExpired
	PairingApprovalLockedOut PairingApprovalStatus = gatewaypairing.PairingApprovalLockedOut
)

// PairingCodeRequest carries the platform-neutral state needed to request a
// pairing code. It intentionally excludes response-copy and adapter behavior.
type PairingCodeRequest = gatewaypairing.PairingCodeRequest

// PairingCodeResult reports whether a code was issued or why policy blocked it.
type PairingCodeResult = gatewaypairing.PairingCodeResult

// PairingApprovalResult reports whether approval succeeded or why it failed.
type PairingApprovalResult = gatewaypairing.PairingApprovalResult

// PairingPendingRecord is one pending pairing request in the read model.
type PairingPendingRecord = gatewaypairing.PairingPendingRecord

// PairingApprovedRecord is one approved user in the read model.
type PairingApprovedRecord = gatewaypairing.PairingApprovedRecord

// PairingPlatformStatus summarizes whether each platform has approved users.
type PairingPlatformStatus = gatewaypairing.PairingPlatformStatus

// PairingDegradedEvidence records read-model degradation and pairing-policy
// attempts operators need to see in status output.
type PairingDegradedEvidence = gatewaypairing.PairingDegradedEvidence

// PairingStatus is the deterministic, operator-facing pairing readout.
type PairingStatus = gatewaypairing.PairingStatus

// PairingStore persists gateway pairing state as one atomic JSON read model.
type PairingStore = gatewaypairing.PairingStore

// DefaultPairingStorePath returns the Gormes runtime-home path for the pairing
// read model. GORMES_HOME wins so gateway status uses the same isolated home as
// config, sessions, memory, and logs.
func DefaultPairingStorePath() string { return gatewaypairing.DefaultPairingStorePath() }

// NewXDGPairingStore returns a pairing store under the XDG data root.
func NewXDGPairingStore() *PairingStore { return gatewaypairing.NewXDGPairingStore() }

// NewPairingStore returns a JSON-backed pairing store at path.
func NewPairingStore(path string) *PairingStore { return gatewaypairing.NewPairingStore(path) }

// PairingCodeRequestFromInbound extracts the identity used by pairing policy
// from a gateway event. Telegram private-chat events may fall back to chat.id
// when from_user is unavailable; group/channel events do not.
func PairingCodeRequestFromInbound(ev InboundEvent, allowlistDenied bool) PairingCodeRequest {
	userID := ev.PairingUserID()
	userName := strings.TrimSpace(ev.UserName)
	if userName == "" && userID != "" && userID == strings.TrimSpace(ev.ChatID) && ev.IsDirectMessage() {
		userName = strings.TrimSpace(ev.ChatName)
	}
	return PairingCodeRequest{
		Platform:        ev.Platform,
		UserID:          userID,
		UserName:        userName,
		AllowlistDenied: allowlistDenied,
	}
}
