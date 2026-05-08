package matrix

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/threadtext"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// RoomKind captures the Matrix room participation policy for a given inbound
// event. It is a pure input to the seam — no env inspection, no network
// access.
type RoomKind string

const (
	RoomKindDM             RoomKind = "dm"
	RoomKindFree           RoomKind = "free"
	RoomKindRequireMention RoomKind = "require_mention"
)

// MentionAndFreeRoomInputs are the pure-input decisions the seam models for
// DM, free-room, and require-mention policies. The caller populates them
// before calling the seam; the seam does not inspect the environment.
type MentionAndFreeRoomInputs struct {
	Kind           RoomKind
	MentionedInMsg bool
}

// ProcessingHook is a callback for Matrix message processing lifecycle
// events. All hooks are optional and fakeable — a nil implementation
// is a valid no-op.
type ProcessingHook func(msg threadtext.InboundMessage)

// ProcessingHooks groups the four lifecycle callbacks: start, complete,
// failure, and cancel. A nil pointer for any individual hook means the
// seam skips that callback without error.
type ProcessingHooks struct {
	OnStart    ProcessingHook
	OnComplete ProcessingHook
	OnFailure  ProcessingHook
	OnCancel   ProcessingHook
}

// Seam is a Matrix-specific normalization layer over the shared
// threaded-text contract. It owns no SDK, no live client, and no
// network state — it is a pure translation boundary.
type Seam struct {
	replyMode    threadtext.ReplyMode
	roomInputs   MentionAndFreeRoomInputs
	hooks        ProcessingHooks
	cancelled    bool
}

// NewSeam returns a Matrix Seam with the configured reply mode, room
// inputs, and processing hooks. If hooks is nil, a zero-value
// ProcessingHooks is used (all callbacks are no-ops).
//
// Callers should wire replyMode from the channel auto-thread config
// (true → threadtext.ReplyModeThread, false → threadtext.ReplyModeFlat)
// and roomInputs from the Matrix room-kind policy decided at binding time.
func NewSeam(replyMode threadtext.ReplyMode, roomInputs MentionAndFreeRoomInputs, hooks *ProcessingHooks) *Seam {
	s := &Seam{
		replyMode:  replyMode,
		roomInputs: roomInputs,
	}
	if hooks != nil {
		s.hooks = *hooks
	}
	return s
}

// NormalizeInbound threads a Matrix transport event into the shared gateway
// contract via the threaded-text normalisation layer, preserving the
// canonical thread root and command parsing.
func (s *Seam) NormalizeInbound(msg threadtext.InboundMessage) (gateway.InboundEvent, bool) {
	if s.hooks.OnStart != nil {
		s.hooks.OnStart(msg)
	}
	ev, ok := threadtext.NormalizeInbound("matrix", msg)
	if !ok {
		if s.hooks.OnFailure != nil && !s.cancelled {
			s.hooks.OnFailure(msg)
		}
		return gateway.InboundEvent{}, false
	}
	if s.hooks.OnComplete != nil && !s.cancelled {
		s.hooks.OnComplete(msg)
	}
	return ev, true
}

// ResolveReplyTarget converts an inbound Matrix message into the
// threaded-text reply target, honouring the seam's reply mode.
func (s *Seam) ResolveReplyTarget(msg threadtext.InboundMessage) (threadtext.ReplyTarget, bool) {
	return threadtext.ResolveReplyTarget(msg, s.replyMode)
}

// RoomInputs returns the room-kind policy inputs set at creation time.
func (s *Seam) RoomInputs() MentionAndFreeRoomInputs {
	return s.roomInputs
}

// Cancel marks the seam as cancelled. After cancellation, terminal
// hooks (OnComplete, OnFailure) are suppressed, and Cancel invokes
// the OnCancel hook with the last successfully normalized message.
func (s *Seam) Cancel(lastMsg threadtext.InboundMessage) {
	s.cancelled = true
	if s.hooks.OnCancel != nil {
		s.hooks.OnCancel(lastMsg)
	}
}

// Cancelled reports whether Cancel has been called.
func (s *Seam) Cancelled() bool {
	return s.cancelled
}
