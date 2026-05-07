package tools

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	defaultBrowserInactivityTimeout = 300 * time.Second
	defaultBrowserReapInterval      = 60 * time.Second
	envBrowserInactivityTimeout     = "GORMES_BROWSER_INACTIVITY_TIMEOUT"
)

// BrowserSessionBackend closes an active browser session. Implementations
// include cloud provider bridges and the local harness backend.
type BrowserSessionBackend interface {
	Close(ctx context.Context, sessionID string) error
}

type browserSessionEntry struct {
	sessionID    string
	backend      BrowserSessionBackend
	lastActivity time.Time
}

// ReapEntry records the outcome of reaping one browser session.
type ReapEntry struct {
	SessionID string
	Err       error
}

// BrowserSessionTracker tracks active browser sessions and supports
// inactivity-based reaping. Safe for concurrent use.
type BrowserSessionTracker struct {
	mu       sync.Mutex
	sessions map[string]*browserSessionEntry
	nowFunc  func() time.Time
	log      *slog.Logger
}

// NewBrowserSessionTracker creates a tracker that uses nowFunc to
// resolve the current time (injected for testability).
func NewBrowserSessionTracker(nowFunc func() time.Time) *BrowserSessionTracker {
	return &BrowserSessionTracker{
		sessions: make(map[string]*browserSessionEntry),
		nowFunc:  nowFunc,
		log:      slog.Default(),
	}
}

// Register adds a session to the tracker. If a session with the same ID
// already exists, its lastActivity is refreshed.
func (t *BrowserSessionTracker) Register(sessionID string, backend BrowserSessionBackend, lastActivity time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sessions[sessionID] = &browserSessionEntry{
		sessionID:    sessionID,
		backend:      backend,
		lastActivity: lastActivity,
	}
}

// Touch refreshes the last-activity timestamp for sessionID.
func (t *BrowserSessionTracker) Touch(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if e, ok := t.sessions[sessionID]; ok {
		e.lastActivity = t.nowFunc()
	}
}

// Unregister removes a session from the tracker without calling Close.
func (t *BrowserSessionTracker) Unregister(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.sessions, sessionID)
}

// Len returns the number of tracked sessions.
func (t *BrowserSessionTracker) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.sessions)
}

// reapInactive closes and removes sessions whose lastActivity is older
// than now - inactivityTimeout. Returns one ReapEntry per session that
// was eligible for reaping, regardless of close success.
func (t *BrowserSessionTracker) reapInactive(now time.Time, inactivityTimeout time.Duration) []ReapEntry {
	t.mu.Lock()
	defer t.mu.Unlock()

	var reaped []ReapEntry
	for id, entry := range t.sessions {
		if now.Sub(entry.lastActivity) < inactivityTimeout {
			continue
		}
		var closeErr error
		if entry.backend != nil {
			closeErr = entry.backend.Close(context.Background(), id)
			if closeErr != nil {
				t.log.Warn("browser_inactivity_cleanup_failed",
					"session_id", id,
					"error", closeErr,
				)
			}
		}
		reaped = append(reaped, ReapEntry{SessionID: id, Err: closeErr})
		delete(t.sessions, id)
	}
	return reaped
}

// inactivityTimeout reads GORMES_BROWSER_INACTIVITY_TIMEOUT from the
// environment or returns the default 300s.
func (t *BrowserSessionTracker) inactivityTimeout() time.Duration {
	if v := os.Getenv(envBrowserInactivityTimeout); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultBrowserInactivityTimeout
}

// RunCleanup starts a blocking loop that periodically reaps inactive
// sessions. Callers should invoke this in a goroutine and cancel ctx to stop.
func (t *BrowserSessionTracker) RunCleanup(ctx context.Context, nowFunc func() time.Time, reapInterval, inactivityTimeout time.Duration) {
	if reapInterval <= 0 {
		reapInterval = defaultBrowserReapInterval
	}
	if inactivityTimeout <= 0 {
		inactivityTimeout = t.inactivityTimeout()
	}

	ticker := time.NewTicker(reapInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := nowFunc()
			reaped := t.reapInactive(now, inactivityTimeout)
			if len(reaped) > 0 {
				t.log.Info("browser_inactivity_cleanup_reaped",
					"count", len(reaped),
					"timeout_seconds", inactivityTimeout.Seconds(),
				)
			}
		}
	}
}


