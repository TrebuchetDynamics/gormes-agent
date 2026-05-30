package gateway

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/clarifyresume"
)

// ClarifyResumeRoute identifies the channel/session awaiting the next user
// reply for a clarify tool call. The broker keeps this channel-neutral so TUI,
// Telegram, and future adapters can share the same one-shot semantics.
type ClarifyResumeRoute = clarifyresume.ClarifyResumeRoute

// PendingClarifyRoute is redacted diagnostic state for a pending clarify route.
type PendingClarifyRoute = clarifyresume.PendingClarifyRoute

// ClarifyResumeBroker owns one-shot clarify routes. Await registers a pending
// route and blocks until Resume supplies the next user reply or the context is
// cancelled. Either path clears the route exactly once.
type ClarifyResumeBroker = clarifyresume.ClarifyResumeBroker

func NewClarifyResumeBroker(now func() time.Time) *ClarifyResumeBroker {
	return clarifyresume.NewClarifyResumeBroker(now)
}
