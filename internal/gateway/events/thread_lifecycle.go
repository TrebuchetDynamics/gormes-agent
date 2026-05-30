package events

// ThreadLifecycleState is the platform-neutral lifecycle state for a threaded
// conversation surface such as a Discord thread or forum post.
type ThreadLifecycleState string

const (
	ThreadLifecycleOpen     ThreadLifecycleState = "open"
	ThreadLifecycleClosed   ThreadLifecycleState = "closed"
	ThreadLifecycleArchived ThreadLifecycleState = "archived"
)

// ThreadLifecycleEvent carries normalized thread metadata alongside the
// channel-neutral inbound event envelope.
type ThreadLifecycleEvent struct {
	ID       string
	ParentID string
	Name     string
	State    ThreadLifecycleState
	Archived bool
	Locked   bool
}
