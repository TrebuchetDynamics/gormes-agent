package queue

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/approval/choice"
)

var (
	ErrGatewayApprovalEmptySession  = errors.New("gateway approval session key is empty")
	ErrGatewayApprovalInvalidChoice = errors.New("gateway approval choice is invalid")
	ErrGatewayApprovalNotPending    = errors.New("gateway approval is not pending")
)

// GatewayApprovalRequest is the dangerous operation metadata queued while a
// gateway user chooses once/session/always/deny.
type GatewayApprovalRequest struct {
	Command     string
	Description string
	PatternKey  string
	PatternKeys []string
	Evidence    map[string]string
}

// GatewayApprovalTicket identifies one queued approval request.
type GatewayApprovalTicket struct {
	SessionKey string
	ID         uint64
}

// GatewayApprovalOutcome records the decision applied to a queued request.
type GatewayApprovalOutcome struct {
	Ticket     GatewayApprovalTicket
	Request    GatewayApprovalRequest
	Resolution choice.Resolution
	Choice     choice.Choice
	Canceled   bool
}

// GatewayApprovalQueue stores pending approval requests per gateway session.
// It mirrors Hermes' FIFO gateway approval queue without doing any channel IO.
type GatewayApprovalQueue struct {
	mu       sync.Mutex
	nextID   uint64
	queues   map[string][]*gatewayApprovalEntry
	outcomes map[uint64]GatewayApprovalOutcome
}

type gatewayApprovalEntry struct {
	ticket  GatewayApprovalTicket
	request GatewayApprovalRequest
}

func NewGatewayApprovalQueue() *GatewayApprovalQueue {
	return &GatewayApprovalQueue{
		queues:   map[string][]*gatewayApprovalEntry{},
		outcomes: map[uint64]GatewayApprovalOutcome{},
	}
}

func (q *GatewayApprovalQueue) SubmitGatewayApproval(sessionKey string, req GatewayApprovalRequest) (GatewayApprovalTicket, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return GatewayApprovalTicket{}, ErrGatewayApprovalEmptySession
	}
	if q == nil {
		return GatewayApprovalTicket{}, ErrGatewayApprovalNotPending
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.ensureLocked()
	q.nextID++
	ticket := GatewayApprovalTicket{SessionKey: sessionKey, ID: q.nextID}
	q.queues[sessionKey] = append(q.queues[sessionKey], &gatewayApprovalEntry{
		ticket:  ticket,
		request: cloneGatewayApprovalRequest(req),
	})
	return ticket, nil
}

func (q *GatewayApprovalQueue) HasBlockingApproval(sessionKey string) bool {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" || q == nil {
		return false
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queues[sessionKey]) > 0
}

func (q *GatewayApprovalQueue) ResolveGatewayApproval(_ context.Context, res choice.Resolution) error {
	sessionKey := strings.TrimSpace(res.SessionKey)
	if sessionKey == "" {
		return ErrGatewayApprovalEmptySession
	}
	if !choice.Valid(res.Choice) {
		return ErrGatewayApprovalInvalidChoice
	}
	if q == nil {
		return ErrGatewayApprovalNotPending
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.ensureLocked()
	queue := q.queues[sessionKey]
	if len(queue) == 0 {
		return ErrGatewayApprovalNotPending
	}
	res.SessionKey = sessionKey
	entry := queue[0]
	queue = queue[1:]
	if len(queue) == 0 {
		delete(q.queues, sessionKey)
	} else {
		q.queues[sessionKey] = queue
	}
	q.recordOutcomeLocked(entry, res, false)
	return nil
}

func (q *GatewayApprovalQueue) ResolveAllGatewayApprovals(sessionKey string, decision choice.Choice) (int, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return 0, ErrGatewayApprovalEmptySession
	}
	if !choice.Valid(decision) {
		return 0, ErrGatewayApprovalInvalidChoice
	}
	if q == nil {
		return 0, nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.ensureLocked()
	entries := q.queues[sessionKey]
	if len(entries) == 0 {
		return 0, nil
	}
	delete(q.queues, sessionKey)
	for _, entry := range entries {
		q.recordOutcomeLocked(entry, choice.Resolution{SessionKey: sessionKey, Choice: decision}, false)
	}
	return len(entries), nil
}

func (q *GatewayApprovalQueue) ClearGatewayApprovalSession(sessionKey string) int {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" || q == nil {
		return 0
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.ensureLocked()
	entries := q.queues[sessionKey]
	if len(entries) == 0 {
		return 0
	}
	delete(q.queues, sessionKey)
	for _, entry := range entries {
		q.recordOutcomeLocked(entry, choice.Resolution{SessionKey: sessionKey, Choice: choice.ChoiceDeny}, true)
	}
	return len(entries)
}

func (q *GatewayApprovalQueue) GatewayApprovalOutcome(ticket GatewayApprovalTicket) (GatewayApprovalOutcome, bool) {
	sessionKey := strings.TrimSpace(ticket.SessionKey)
	if q == nil || ticket.ID == 0 || sessionKey == "" {
		return GatewayApprovalOutcome{}, false
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	outcome, ok := q.outcomes[ticket.ID]
	if !ok || outcome.Ticket.SessionKey != sessionKey {
		return GatewayApprovalOutcome{}, false
	}
	return cloneGatewayApprovalOutcome(outcome), true
}

func (q *GatewayApprovalQueue) ensureLocked() {
	if q.queues == nil {
		q.queues = map[string][]*gatewayApprovalEntry{}
	}
	if q.outcomes == nil {
		q.outcomes = map[uint64]GatewayApprovalOutcome{}
	}
}

func (q *GatewayApprovalQueue) recordOutcomeLocked(entry *gatewayApprovalEntry, res choice.Resolution, canceled bool) {
	decision := res.Choice
	if decision == "" {
		decision = choice.ChoiceDeny
	}
	q.outcomes[entry.ticket.ID] = GatewayApprovalOutcome{
		Ticket:     entry.ticket,
		Request:    cloneGatewayApprovalRequest(entry.request),
		Resolution: cloneResolution(res),
		Choice:     decision,
		Canceled:   canceled,
	}
}

func cloneGatewayApprovalRequest(req GatewayApprovalRequest) GatewayApprovalRequest {
	req.Command = strings.TrimSpace(req.Command)
	req.Description = strings.TrimSpace(req.Description)
	req.PatternKey = strings.TrimSpace(req.PatternKey)
	patterns := make([]string, 0, len(req.PatternKeys))
	for _, pattern := range req.PatternKeys {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			patterns = append(patterns, pattern)
		}
	}
	req.PatternKeys = patterns
	req.Evidence = cloneEvidence(req.Evidence)
	return req
}

func cloneResolution(res choice.Resolution) choice.Resolution {
	res.SessionKey = strings.TrimSpace(res.SessionKey)
	res.Platform = strings.TrimSpace(res.Platform)
	res.ChatID = strings.TrimSpace(res.ChatID)
	res.MessageID = strings.TrimSpace(res.MessageID)
	res.ActorID = strings.TrimSpace(res.ActorID)
	res.Evidence = cloneEvidence(res.Evidence)
	return res
}

func cloneEvidence(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(value)
	}
	return out
}

func cloneGatewayApprovalOutcome(outcome GatewayApprovalOutcome) GatewayApprovalOutcome {
	outcome.Request = cloneGatewayApprovalRequest(outcome.Request)
	outcome.Resolution = cloneResolution(outcome.Resolution)
	return outcome
}
