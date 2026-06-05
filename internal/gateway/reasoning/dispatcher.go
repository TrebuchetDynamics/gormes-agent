package reasoning

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/reasoning/session"

// PersistGlobal stores a globally scoped reasoning effort.
type PersistGlobal = session.PersistGlobal

// Dispatcher owns per-session /reasoning command state. It keeps state
// mutation and persistence fallback policy out of the gateway Manager while
// preserving the manager's session-keyed behavior.
type Dispatcher = session.Dispatcher

// NewDispatcher returns a session-keyed /reasoning dispatcher. When
// persistGlobal is nil, global sets fall back to session scope with
// PersistFailed set, matching the gateway's previous behavior.
func NewDispatcher(persistGlobal PersistGlobal) *Dispatcher {
	return session.NewDispatcher(persistGlobal)
}
