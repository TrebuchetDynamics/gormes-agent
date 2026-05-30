package reasoning

import (
	"errors"
	"sync"
)

// PersistGlobal stores a globally scoped reasoning effort.
type PersistGlobal func(ReasoningEffort) error

// Dispatcher owns per-session /reasoning command state. It keeps state
// mutation and persistence fallback policy out of the gateway Manager while
// preserving the manager's session-keyed behavior.
type Dispatcher struct {
	mu            sync.Mutex
	state         map[string]SessionReasoningState
	persistGlobal PersistGlobal
}

// NewDispatcher returns a session-keyed /reasoning dispatcher. When
// persistGlobal is nil, global sets fall back to session scope with
// PersistFailed set, matching the gateway's previous behavior.
func NewDispatcher(persistGlobal PersistGlobal) *Dispatcher {
	return &Dispatcher{
		state:         map[string]SessionReasoningState{},
		persistGlobal: persistGlobal,
	}
}

// Dispatch parses and applies a /reasoning command for sessionKey.
func (d *Dispatcher) Dispatch(sessionKey string, args []string) (ReasoningReply, error) {
	cmd, err := ParseReasoningCommand(args)
	if err != nil {
		return ReasoningReply{}, err
	}

	persist := PersistGlobal(nil)
	if d != nil {
		persist = d.persistGlobal
	}
	if persist == nil {
		persist = func(ReasoningEffort) error {
			return errors.New("gateway: PersistReasoningGlobal not configured")
		}
	}

	if d == nil {
		_, reply := ApplyReasoningCommand(SessionReasoningState{Source: ReasoningSourceUnset}, cmd, persist)
		return reply, nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state == nil {
		d.state = map[string]SessionReasoningState{}
	}
	state, ok := d.state[sessionKey]
	if !ok {
		state = SessionReasoningState{Source: ReasoningSourceUnset}
	}
	newState, reply := ApplyReasoningCommand(state, cmd, persist)
	d.state[sessionKey] = newState
	return reply, nil
}
