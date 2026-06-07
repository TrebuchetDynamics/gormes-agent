package clarifyresume

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// ClarifyResumeRoute identifies the channel/session awaiting the next user
// reply for a clarify tool call. The broker keeps this channel-neutral so TUI,
// Telegram, and future adapters can share the same one-shot semantics.
type ClarifyResumeRoute struct {
	SessionID string
	Platform  string
	ChatID    string
	MsgID     string
}

// PendingClarifyRoute is redacted diagnostic state for a pending clarify route.
type PendingClarifyRoute struct {
	SessionID string
	Platform  string
	ChatID    string
	MsgID     string
	Question  string
	Choices   []string
	CreatedAt time.Time
}

type clarifyPending struct {
	route  PendingClarifyRoute
	answer chan string
}

// ClarifyResumeBroker owns one-shot clarify routes. Await registers a pending
// route and blocks until Resume supplies the next user reply or the context is
// cancelled. Either path clears the route exactly once.
type ClarifyResumeBroker struct {
	mu      sync.Mutex
	now     func() time.Time
	pending map[string]*clarifyPending
}

func NewClarifyResumeBroker(now func() time.Time) *ClarifyResumeBroker {
	if now == nil {
		now = time.Now
	}
	return &ClarifyResumeBroker{now: now, pending: map[string]*clarifyPending{}}
}

func (b *ClarifyResumeBroker) Await(ctx context.Context, route ClarifyResumeRoute, req tools.ClarifyRequest) (tools.ClarifyResponse, error) {
	if b == nil || strings.TrimSpace(route.Platform) == "" || strings.TrimSpace(route.ChatID) == "" {
		return tools.ClarifyResponse{}, tools.ErrClarifyRouteMissing
	}
	key := clarifyResumeKey(route.Platform, route.ChatID)
	pending := &clarifyPending{
		route: PendingClarifyRoute{
			SessionID: strings.TrimSpace(route.SessionID),
			Platform:  strings.TrimSpace(route.Platform),
			ChatID:    strings.TrimSpace(route.ChatID),
			MsgID:     strings.TrimSpace(route.MsgID),
			Question:  strings.TrimSpace(req.Question),
			Choices:   slices.Clone(req.Choices),
			CreatedAt: b.now().UTC(),
		},
		answer: make(chan string, 1),
	}

	b.mu.Lock()
	b.pending[key] = pending
	b.mu.Unlock()

	select {
	case answer := <-pending.answer:
		return tools.ClarifyResponse{UserResponse: strings.TrimSpace(answer)}, nil
	case <-ctx.Done():
		b.clearIfCurrent(key, pending)
		return tools.ClarifyResponse{}, tools.ErrClarifyTimeout
	}
}

func (b *ClarifyResumeBroker) Resume(platform, chatID, userResponse string) bool {
	if b == nil {
		return false
	}
	key := clarifyResumeKey(platform, chatID)
	b.mu.Lock()
	pending, ok := b.pending[key]
	if ok {
		delete(b.pending, key)
	}
	b.mu.Unlock()
	if !ok {
		return false
	}
	pending.answer <- strings.TrimSpace(userResponse)
	return true
}

func (b *ClarifyResumeBroker) Pending(platform, chatID string) (PendingClarifyRoute, bool) {
	if b == nil {
		return PendingClarifyRoute{}, false
	}
	key := clarifyResumeKey(platform, chatID)
	b.mu.Lock()
	defer b.mu.Unlock()
	pending, ok := b.pending[key]
	if !ok {
		return PendingClarifyRoute{}, false
	}
	out := pending.route
	out.Choices = slices.Clone(pending.route.Choices)
	return out, true
}

func (b *ClarifyResumeBroker) clearIfCurrent(key string, pending *clarifyPending) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pending[key] == pending {
		delete(b.pending, key)
	}
}

func clarifyResumeKey(platform, chatID string) string {
	return strings.TrimSpace(platform) + ":" + strings.TrimSpace(chatID)
}
