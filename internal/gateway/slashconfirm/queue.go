package slashconfirm

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrEmptySession  = errors.New("gateway slash confirmation session key is empty")
	ErrInvalidChoice = errors.New("gateway slash confirmation choice is invalid")
	ErrNotPending    = errors.New("gateway slash confirmation is not pending")
	ErrIDMismatch    = errors.New("gateway slash confirmation id mismatch")
)

// Choice is the bounded decision for a confirmable slash action such as
// Hermes' /reload-mcp prompt.
type Choice string

const (
	ChoiceOnce   Choice = "once"
	ChoiceAlways Choice = "always"
	ChoiceCancel Choice = "cancel"
)

// Request is the redacted metadata stored while a gateway user decides whether
// a confirmable slash action should run.
type Request struct {
	Command     string
	Description string
	Evidence    map[string]string
}

// Ticket identifies one pending slash confirmation.
type Ticket struct {
	SessionKey string
	ID         uint64
}

// Pending is the read model for a session's current prompt.
type Pending struct {
	Ticket    Ticket
	Request   Request
	CreatedAt time.Time
}

// Resolution is the channel-neutral callback payload.
type Resolution struct {
	SessionKey string
	ID         uint64
	Choice     Choice
}

// Outcome records a resolved confirmation. Scoped clears do not create
// outcomes because no slash action was answered or run.
type Outcome struct {
	Ticket     Ticket
	Request    Request
	Resolution Resolution
	Choice     Choice
	Canceled   bool
}

// Queue stores at most one pending confirmable slash action per gateway
// session. Registering a new confirmation supersedes the previous one for that
// session, matching Hermes' slash_confirm module.
type Queue struct {
	mu       sync.Mutex
	nextID   uint64
	pending  map[string]*entry
	outcomes map[uint64]Outcome
	now      func() time.Time
}

type entry struct {
	pending Pending
}

func NewQueue() *Queue {
	return &Queue{
		pending:  map[string]*entry{},
		outcomes: map[uint64]Outcome{},
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (q *Queue) RegisterSlashConfirmation(sessionKey string, req Request) (Ticket, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return Ticket{}, ErrEmptySession
	}
	if q == nil {
		return Ticket{}, ErrNotPending
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.ensureLocked()
	q.nextID++
	ticket := Ticket{SessionKey: sessionKey, ID: q.nextID}
	q.pending[sessionKey] = &entry{
		pending: Pending{
			Ticket:    ticket,
			Request:   cloneRequest(req),
			CreatedAt: q.now().UTC(),
		},
	}
	return ticket, nil
}

func (q *Queue) PendingSlashConfirmation(sessionKey string) (Pending, bool) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" || q == nil {
		return Pending{}, false
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	entry, ok := q.pending[sessionKey]
	if !ok || entry == nil {
		return Pending{}, false
	}
	return clonePending(entry.pending), true
}

func (q *Queue) ResolveSlashConfirmation(_ context.Context, res Resolution) (Outcome, error) {
	sessionKey := strings.TrimSpace(res.SessionKey)
	if sessionKey == "" {
		return Outcome{}, ErrEmptySession
	}
	if !validChoice(res.Choice) {
		return Outcome{}, ErrInvalidChoice
	}
	if q == nil {
		return Outcome{}, ErrNotPending
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.ensureLocked()
	entry := q.pending[sessionKey]
	if entry == nil {
		return Outcome{}, ErrNotPending
	}
	if entry.pending.Ticket.ID != res.ID {
		return Outcome{}, ErrIDMismatch
	}
	delete(q.pending, sessionKey)
	outcome := Outcome{
		Ticket:     entry.pending.Ticket,
		Request:    cloneRequest(entry.pending.Request),
		Resolution: cloneResolution(res),
		Choice:     res.Choice,
		Canceled:   res.Choice == ChoiceCancel,
	}
	q.outcomes[outcome.Ticket.ID] = cloneOutcome(outcome)
	return outcome, nil
}

func (q *Queue) ClearSlashConfirmationSession(sessionKey string) bool {
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

func (q *Queue) SlashConfirmationOutcome(ticket Ticket) (Outcome, bool) {
	if q == nil || ticket.ID == 0 || strings.TrimSpace(ticket.SessionKey) == "" {
		return Outcome{}, false
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	outcome, ok := q.outcomes[ticket.ID]
	if !ok || outcome.Ticket.SessionKey != ticket.SessionKey {
		return Outcome{}, false
	}
	return cloneOutcome(outcome), true
}

func (q *Queue) ensureLocked() {
	if q.pending == nil {
		q.pending = map[string]*entry{}
	}
	if q.outcomes == nil {
		q.outcomes = map[uint64]Outcome{}
	}
	if q.now == nil {
		q.now = func() time.Time { return time.Now().UTC() }
	}
}

func validChoice(choice Choice) bool {
	switch choice {
	case ChoiceOnce, ChoiceAlways, ChoiceCancel:
		return true
	default:
		return false
	}
}

func cloneRequest(req Request) Request {
	req.Command = strings.TrimSpace(req.Command)
	req.Evidence = cloneStringMap(req.Evidence)
	return req
}

func cloneResolution(res Resolution) Resolution {
	res.SessionKey = strings.TrimSpace(res.SessionKey)
	return res
}

func clonePending(pending Pending) Pending {
	pending.Request = cloneRequest(pending.Request)
	return pending
}

func cloneOutcome(outcome Outcome) Outcome {
	outcome.Request = cloneRequest(outcome.Request)
	outcome.Resolution = cloneResolution(outcome.Resolution)
	return outcome
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
