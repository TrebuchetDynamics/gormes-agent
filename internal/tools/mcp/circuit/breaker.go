package circuit

import (
	"errors"
	"sync"
	"time"
)

var ErrBreakerOpen = errors.New("mcp circuit breaker: open")

type Evidence string

const (
	EvidenceOK                Evidence = "mcp_ok"
	EvidenceServerUnreachable Evidence = "mcp_server_unreachable"
	EvidenceBreakerOpen       Evidence = "mcp_breaker_open"
	EvidenceHalfOpenFailed    Evidence = "mcp_half_open_failed"
	EvidenceReconnectRequired Evidence = "mcp_reconnect_required"
	EvidenceReconnectReset    Evidence = "mcp_reconnect_reset"
)

const (
	DefaultBreakerThreshold = 3
	DefaultBreakerCooldown  = 60 * time.Second
	DefaultServerName       = "default"
)

type BreakerOptions struct {
	Threshold int
	Cooldown  time.Duration
	Now       func() time.Time
}

type Breaker struct {
	mu          sync.Mutex
	threshold   int
	cooldown    time.Duration
	now         func() time.Time
	errorCounts map[string]int
	openedAt    map[string]time.Time
	halfOpen    map[string]bool
}

func NewBreaker(opts BreakerOptions) *Breaker {
	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = DefaultBreakerThreshold
	}
	cooldown := opts.Cooldown
	if cooldown <= 0 {
		cooldown = DefaultBreakerCooldown
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Breaker{
		threshold:   threshold,
		cooldown:    cooldown,
		now:         now,
		errorCounts: map[string]int{},
		openedAt:    map[string]time.Time{},
		halfOpen:    map[string]bool{},
	}
}

func (b *Breaker) ErrorCount(server string) int {
	if b == nil {
		return 0
	}
	server = NormalizeServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.errorCounts[server]
}

func (b *Breaker) ResetAfterReconnect(server string) Evidence {
	if b == nil {
		return EvidenceReconnectReset
	}
	server = NormalizeServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.errorCounts, server)
	delete(b.openedAt, server)
	delete(b.halfOpen, server)
	return EvidenceReconnectReset
}

func (b *Breaker) RecordSuccess(server string) Evidence {
	if b == nil {
		return EvidenceOK
	}
	server = NormalizeServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.errorCounts, server)
	delete(b.openedAt, server)
	delete(b.halfOpen, server)
	return EvidenceOK
}

func (b *Breaker) RecordFailure(server string, _ error) Evidence {
	if b == nil {
		return EvidenceServerUnreachable
	}
	server = NormalizeServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	if b.halfOpen[server] {
		b.errorCounts[server] = b.threshold
		b.openedAt[server] = now
		delete(b.halfOpen, server)
		return EvidenceHalfOpenFailed
	}
	b.errorCounts[server]++
	if b.errorCounts[server] >= b.threshold {
		b.openedAt[server] = now
	}
	return EvidenceServerUnreachable
}

func (b *Breaker) BeforeCall(server string) (bool, Evidence) {
	if b == nil {
		return true, EvidenceOK
	}
	server = NormalizeServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	count := b.errorCounts[server]
	if count < b.threshold {
		return true, EvidenceOK
	}
	now := b.now()
	opened := b.openedAt[server]
	if opened.IsZero() {
		b.openedAt[server] = now
		return false, EvidenceBreakerOpen
	}
	if now.Sub(opened) >= b.cooldown {
		b.halfOpen[server] = true
		return true, EvidenceOK
	}
	return false, EvidenceBreakerOpen
}

func NormalizeServer(server string) string {
	if server == "" {
		return DefaultServerName
	}
	return server
}
