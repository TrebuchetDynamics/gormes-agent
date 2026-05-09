package gateway

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrSlashConfirmationEmptySession  = errors.New("gateway slash confirmation session key is empty")
	ErrSlashConfirmationInvalidChoice = errors.New("gateway slash confirmation choice is invalid")
	ErrSlashConfirmationNotPending    = errors.New("gateway slash confirmation is not pending")
	ErrSlashConfirmationIDMismatch    = errors.New("gateway slash confirmation id mismatch")
)

// SlashConfirmationChoice is the bounded decision for a confirmable slash
// action such as Hermes' /reload-mcp prompt.
type SlashConfirmationChoice string

const (
	SlashConfirmationChoiceOnce   SlashConfirmationChoice = "once"
	SlashConfirmationChoiceAlways SlashConfirmationChoice = "always"
	SlashConfirmationChoiceCancel SlashConfirmationChoice = "cancel"
)

// SlashConfirmationRequest is the redacted metadata stored while a gateway
// user decides whether a confirmable slash action should run.
type SlashConfirmationRequest struct {
	Command     string
	Description string
	Evidence    map[string]string
}

// SlashConfirmationTicket identifies one pending slash confirmation.
type SlashConfirmationTicket struct {
	SessionKey string
	ID         uint64
}

// SlashConfirmationPending is the read model for a session's current prompt.
type SlashConfirmationPending struct {
	Ticket    SlashConfirmationTicket
	Request   SlashConfirmationRequest
	CreatedAt time.Time
}

// SlashConfirmationResolution is the channel-neutral callback payload.
type SlashConfirmationResolution struct {
	SessionKey string
	ID         uint64
	Choice     SlashConfirmationChoice
}

// SlashConfirmationOutcome records a resolved confirmation. Scoped clears do
// not create outcomes because no slash action was answered or run.
type SlashConfirmationOutcome struct {
	Ticket     SlashConfirmationTicket
	Request    SlashConfirmationRequest
	Resolution SlashConfirmationResolution
	Choice     SlashConfirmationChoice
	Canceled   bool
}

// SlashConfirmationQueue stores at most one pending confirmable slash action
// per gateway session. Registering a new confirmation supersedes the previous
// one for that session, matching Hermes' slash_confirm module.
type SlashConfirmationQueue struct {
	mu       sync.Mutex
	nextID   uint64
	pending  map[string]*slashConfirmationEntry
	outcomes map[uint64]SlashConfirmationOutcome
	now      func() time.Time
}

type slashConfirmationEntry struct {
	pending SlashConfirmationPending
}

func NewSlashConfirmationQueue() *SlashConfirmationQueue {
	return &SlashConfirmationQueue{
		pending:  map[string]*slashConfirmationEntry{},
		outcomes: map[uint64]SlashConfirmationOutcome{},
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (q *SlashConfirmationQueue) RegisterSlashConfirmation(sessionKey string, req SlashConfirmationRequest) (SlashConfirmationTicket, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return SlashConfirmationTicket{}, ErrSlashConfirmationEmptySession
	}
	if q == nil {
		return SlashConfirmationTicket{}, ErrSlashConfirmationNotPending
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.ensureLocked()
	q.nextID++
	ticket := SlashConfirmationTicket{SessionKey: sessionKey, ID: q.nextID}
	q.pending[sessionKey] = &slashConfirmationEntry{
		pending: SlashConfirmationPending{
			Ticket:    ticket,
			Request:   cloneSlashConfirmationRequest(req),
			CreatedAt: q.now().UTC(),
		},
	}
	return ticket, nil
}

func (q *SlashConfirmationQueue) PendingSlashConfirmation(sessionKey string) (SlashConfirmationPending, bool) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" || q == nil {
		return SlashConfirmationPending{}, false
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	entry, ok := q.pending[sessionKey]
	if !ok || entry == nil {
		return SlashConfirmationPending{}, false
	}
	return cloneSlashConfirmationPending(entry.pending), true
}

func (q *SlashConfirmationQueue) ResolveSlashConfirmation(_ context.Context, res SlashConfirmationResolution) (SlashConfirmationOutcome, error) {
	sessionKey := strings.TrimSpace(res.SessionKey)
	if sessionKey == "" {
		return SlashConfirmationOutcome{}, ErrSlashConfirmationEmptySession
	}
	if !validSlashConfirmationChoice(res.Choice) {
		return SlashConfirmationOutcome{}, ErrSlashConfirmationInvalidChoice
	}
	if q == nil {
		return SlashConfirmationOutcome{}, ErrSlashConfirmationNotPending
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.ensureLocked()
	entry := q.pending[sessionKey]
	if entry == nil {
		return SlashConfirmationOutcome{}, ErrSlashConfirmationNotPending
	}
	if entry.pending.Ticket.ID != res.ID {
		return SlashConfirmationOutcome{}, ErrSlashConfirmationIDMismatch
	}
	delete(q.pending, sessionKey)
	outcome := SlashConfirmationOutcome{
		Ticket:     entry.pending.Ticket,
		Request:    cloneSlashConfirmationRequest(entry.pending.Request),
		Resolution: cloneSlashConfirmationResolution(res),
		Choice:     res.Choice,
		Canceled:   res.Choice == SlashConfirmationChoiceCancel,
	}
	q.outcomes[outcome.Ticket.ID] = cloneSlashConfirmationOutcome(outcome)
	return outcome, nil
}

func (q *SlashConfirmationQueue) ClearSlashConfirmationSession(sessionKey string) bool {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" || q == nil {
		return false
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.ensureLocked()
	if _, ok := q.pending[sessionKey]; !ok {
		return false
	}
	delete(q.pending, sessionKey)
	return true
}

func (q *SlashConfirmationQueue) SlashConfirmationOutcome(ticket SlashConfirmationTicket) (SlashConfirmationOutcome, bool) {
	if q == nil || ticket.ID == 0 || strings.TrimSpace(ticket.SessionKey) == "" {
		return SlashConfirmationOutcome{}, false
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	outcome, ok := q.outcomes[ticket.ID]
	if !ok || outcome.Ticket.SessionKey != ticket.SessionKey {
		return SlashConfirmationOutcome{}, false
	}
	return cloneSlashConfirmationOutcome(outcome), true
}

func (q *SlashConfirmationQueue) ensureLocked() {
	if q.pending == nil {
		q.pending = map[string]*slashConfirmationEntry{}
	}
	if q.outcomes == nil {
		q.outcomes = map[uint64]SlashConfirmationOutcome{}
	}
	if q.now == nil {
		q.now = func() time.Time { return time.Now().UTC() }
	}
}

func validSlashConfirmationChoice(choice SlashConfirmationChoice) bool {
	switch choice {
	case SlashConfirmationChoiceOnce, SlashConfirmationChoiceAlways, SlashConfirmationChoiceCancel:
		return true
	default:
		return false
	}
}

func cloneSlashConfirmationRequest(req SlashConfirmationRequest) SlashConfirmationRequest {
	req.Command = strings.TrimSpace(req.Command)
	req.Evidence = cloneStringMap(req.Evidence)
	return req
}

func cloneSlashConfirmationResolution(res SlashConfirmationResolution) SlashConfirmationResolution {
	res.SessionKey = strings.TrimSpace(res.SessionKey)
	return res
}

func cloneSlashConfirmationPending(pending SlashConfirmationPending) SlashConfirmationPending {
	pending.Request = cloneSlashConfirmationRequest(pending.Request)
	return pending
}

func cloneSlashConfirmationOutcome(outcome SlashConfirmationOutcome) SlashConfirmationOutcome {
	outcome.Request = cloneSlashConfirmationRequest(outcome.Request)
	outcome.Resolution = cloneSlashConfirmationResolution(outcome.Resolution)
	return outcome
}
