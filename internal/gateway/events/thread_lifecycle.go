package events

import eventinbound "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events/inbound"

// ThreadLifecycleState is the platform-neutral lifecycle state for a threaded
// conversation surface such as a Discord thread or forum post.
type ThreadLifecycleState = eventinbound.ThreadLifecycleState

const (
	ThreadLifecycleOpen     = eventinbound.ThreadLifecycleOpen
	ThreadLifecycleClosed   = eventinbound.ThreadLifecycleClosed
	ThreadLifecycleArchived = eventinbound.ThreadLifecycleArchived
)

// ThreadLifecycleEvent carries normalized thread metadata alongside the
// channel-neutral inbound event envelope.
type ThreadLifecycleEvent = eventinbound.ThreadLifecycleEvent
