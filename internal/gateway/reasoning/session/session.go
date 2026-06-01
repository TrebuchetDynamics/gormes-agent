// Package session applies and dispatches gateway /reasoning commands for chat sessions.
package session

import (
	"errors"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/reasoning/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/reasoning/parser"
)

// PersistGlobal stores a globally scoped reasoning effort.
type PersistGlobal func(model.ReasoningEffort) error

// Apply mutates the supplied SessionReasoningState according to a parsed
// ReasoningCommand. persistGlobal is invoked only for Set actions with
// Global=true; on failure the slice falls back to a session-only override and
// surfaces PersistFailed=true so the caller can warn the user.
func Apply(
	state model.SessionReasoningState,
	cmd model.ReasoningCommand,
	persistGlobal func(model.ReasoningEffort) error,
) (model.SessionReasoningState, model.ReasoningReply) {
	switch cmd.Action {
	case model.ReasoningActionShow:
		return state, model.ReasoningReply{Effort: state.Effort, Scope: state.Source}
	case model.ReasoningActionReset:
		next := model.SessionReasoningState{Source: model.ReasoningSourceUnset}
		return next, model.ReasoningReply{Scope: model.ReasoningSourceUnset}
	case model.ReasoningActionSet:
		if cmd.Global {
			if err := persistGlobal(cmd.Effort); err != nil {
				next := model.SessionReasoningState{Effort: cmd.Effort, Source: model.ReasoningSourceSession}
				return next, model.ReasoningReply{
					Effort:        cmd.Effort,
					Scope:         model.ReasoningSourceSession,
					PersistFailed: true,
				}
			}
			next := model.SessionReasoningState{Effort: cmd.Effort, Source: model.ReasoningSourceGlobal}
			return next, model.ReasoningReply{Effort: cmd.Effort, Scope: model.ReasoningSourceGlobal}
		}
		next := model.SessionReasoningState{Effort: cmd.Effort, Source: model.ReasoningSourceSession}
		return next, model.ReasoningReply{Effort: cmd.Effort, Scope: model.ReasoningSourceSession}
	}
	return state, model.ReasoningReply{Effort: state.Effort, Scope: state.Source}
}

// Dispatcher owns per-session /reasoning command state. It keeps state
// mutation and persistence fallback policy out of the gateway Manager while
// preserving the manager's session-keyed behavior.
type Dispatcher struct {
	mu            sync.Mutex
	state         map[string]model.SessionReasoningState
	persistGlobal PersistGlobal
}

// NewDispatcher returns a session-keyed /reasoning dispatcher. When
// persistGlobal is nil, global sets fall back to session scope with
// PersistFailed set, matching the gateway's previous behavior.
func NewDispatcher(persistGlobal PersistGlobal) *Dispatcher {
	return &Dispatcher{
		state:         map[string]model.SessionReasoningState{},
		persistGlobal: persistGlobal,
	}
}

// Dispatch parses and applies a /reasoning command for sessionKey.
func (d *Dispatcher) Dispatch(sessionKey string, args []string) (model.ReasoningReply, error) {
	cmd, err := parser.Parse(args)
	if err != nil {
		return model.ReasoningReply{}, err
	}

	persist := PersistGlobal(nil)
	if d != nil {
		persist = d.persistGlobal
	}
	if persist == nil {
		persist = func(model.ReasoningEffort) error {
			return errors.New("gateway: PersistReasoningGlobal not configured")
		}
	}

	if d == nil {
		_, reply := Apply(model.SessionReasoningState{Source: model.ReasoningSourceUnset}, cmd, persist)
		return reply, nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state == nil {
		d.state = map[string]model.SessionReasoningState{}
	}
	state, ok := d.state[sessionKey]
	if !ok {
		state = model.SessionReasoningState{Source: model.ReasoningSourceUnset}
	}
	newState, reply := Apply(state, cmd, persist)
	d.state[sessionKey] = newState
	return reply, nil
}
