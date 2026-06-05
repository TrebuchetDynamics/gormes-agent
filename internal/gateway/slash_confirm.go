package gateway

import gatewayslashconfirm "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/slashconfirm"

var (
	ErrSlashConfirmationEmptySession  = gatewayslashconfirm.ErrEmptySession
	ErrSlashConfirmationInvalidChoice = gatewayslashconfirm.ErrInvalidChoice
	ErrSlashConfirmationNotPending    = gatewayslashconfirm.ErrNotPending
	ErrSlashConfirmationIDMismatch    = gatewayslashconfirm.ErrIDMismatch
)

// SlashConfirmationChoice is the bounded decision for a confirmable slash
// action such as Hermes' /reload-mcp prompt.
type SlashConfirmationChoice = gatewayslashconfirm.Choice

const (
	SlashConfirmationChoiceOnce   SlashConfirmationChoice = gatewayslashconfirm.ChoiceOnce
	SlashConfirmationChoiceAlways SlashConfirmationChoice = gatewayslashconfirm.ChoiceAlways
	SlashConfirmationChoiceCancel SlashConfirmationChoice = gatewayslashconfirm.ChoiceCancel
)

// SlashConfirmationRequest is the redacted metadata stored while a gateway
// user decides whether a confirmable slash action should run.
type SlashConfirmationRequest = gatewayslashconfirm.Request

// SlashConfirmationTicket identifies one pending slash confirmation.
type SlashConfirmationTicket = gatewayslashconfirm.Ticket

// SlashConfirmationPending is the read model for a session's current prompt.
type SlashConfirmationPending = gatewayslashconfirm.Pending

// SlashConfirmationResolution is the channel-neutral callback payload.
type SlashConfirmationResolution = gatewayslashconfirm.Resolution

// SlashConfirmationOutcome records a resolved confirmation. Scoped clears do
// not create outcomes because no slash action was answered or run.
type SlashConfirmationOutcome = gatewayslashconfirm.Outcome

// SlashConfirmationQueue stores at most one pending confirmable slash action
// per gateway session. Registering a new confirmation supersedes the previous
// one for that session, matching Hermes' slash_confirm module.
type SlashConfirmationQueue = gatewayslashconfirm.Queue

func NewSlashConfirmationQueue() *SlashConfirmationQueue {
	return gatewayslashconfirm.NewQueue()
}
