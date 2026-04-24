package tuigateway

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/kernel"
)

// RemoteClient composes DialSSE (downstream frames) and PostPlatformEvent
// (upstream events) into the minimal surface a remote Bubble Tea runner
// needs. It is the client-side seat for `cmd/gormes --remote <url>`: once
// wired, the TUI reads from Frames(ctx) and calls Submit/Cancel/Reset in
// place of the local kernel's Submit.
//
// FramesURL is the GET endpoint that serves `event: frame` SSE output
// (typically a NewSSEHandler server). EventsURL is the POST endpoint for
// upstream wire events (typically a NewEventHandler server). They are
// held separately so a production gateway can mount them under different
// paths (e.g. /frames, /events) or behind different origins if a CDN
// proxy fans out only the long-lived SSE leg.
//
// The zero value is not usable: set both URLs before calling any method.
type RemoteClient struct {
	FramesURL string
	EventsURL string
}

// Frames opens a long-lived SSE stream against FramesURL and returns a
// channel that yields decoded kernel.RenderFrames. The channel closes when
// the server emits `event: end`, when the underlying connection EOFs, or
// when ctx is cancelled — whichever fires first. Non-200 handshake
// responses (and dial errors) are returned synchronously so the --remote
// startup path can bail loudly instead of hanging on an empty channel.
func (c *RemoteClient) Frames(ctx context.Context) (<-chan kernel.RenderFrame, error) {
	return DialSSE(ctx, c.FramesURL)
}

// Submit POSTs a PlatformEventSubmit carrying text to EventsURL.
func (c *RemoteClient) Submit(ctx context.Context, text string) error {
	return PostPlatformEvent(ctx, c.EventsURL, kernel.PlatformEvent{
		Kind: kernel.PlatformEventSubmit,
		Text: text,
	})
}

// Cancel POSTs a PlatformEventCancel. It carries no payload; the kernel's
// in-flight turn (if any) is aborted when the event is observed.
func (c *RemoteClient) Cancel(ctx context.Context) error {
	return PostPlatformEvent(ctx, c.EventsURL, kernel.PlatformEvent{
		Kind: kernel.PlatformEventCancel,
	})
}

// Reset POSTs a PlatformEventResetSession, telling the remote kernel to
// begin a fresh session. Mirrors the local TUI's "new chat" affordance.
func (c *RemoteClient) Reset(ctx context.Context) error {
	return PostPlatformEvent(ctx, c.EventsURL, kernel.PlatformEvent{
		Kind: kernel.PlatformEventResetSession,
	})
}
