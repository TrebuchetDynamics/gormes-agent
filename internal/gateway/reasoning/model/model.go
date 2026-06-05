// Package model defines the pure value types for the gateway /reasoning command.
package model

import "errors"

// ReasoningAction enumerates the parsed forms of the gateway /reasoning
// command. The parser is pure: state, persistence, and dispatch live in the
// follow-up apply/dispatch slice.
type ReasoningAction int

const (
	// ReasoningActionShow corresponds to /reasoning with no arguments.
	ReasoningActionShow ReasoningAction = iota
	// ReasoningActionSet corresponds to /reasoning <effort> [--global].
	ReasoningActionSet
	// ReasoningActionReset corresponds to /reasoning reset.
	ReasoningActionReset
)

// ReasoningEffort is the validated effort level recognized by the parser.
// The empty value represents "no effort selected" for non-Set actions.
type ReasoningEffort string

const (
	ReasoningEffortHigh   ReasoningEffort = "high"
	ReasoningEffortLow    ReasoningEffort = "low"
	ReasoningEffortMedium ReasoningEffort = "medium"
)

// ReasoningCommand is the parsed shape of a /reasoning invocation.
type ReasoningCommand struct {
	Action ReasoningAction
	Effort ReasoningEffort
	Global bool
}

// ErrInvalidEffort is returned when the user supplies an effort token outside
// the supported set (high|low|medium). The dispatcher renders this as the
// upstream "unknown argument" warning class.
var ErrInvalidEffort = errors.New("reasoning: invalid effort")

// ErrResetGlobalUnsupported is returned when "reset" is combined with
// "--global". The dispatcher surfaces this verbatim because the upstream
// gateway rejects this combination too.
var ErrResetGlobalUnsupported = errors.New("reasoning: reset --global unsupported")

// Reasoning scope tags surfaced in ReasoningReply.Scope and stored in
// SessionReasoningState.Source. They distinguish session overrides from the
// persisted global default and the default-unset state.
const (
	ReasoningSourceUnset   = "unset"
	ReasoningSourceSession = "session"
	ReasoningSourceGlobal  = "global"
)

// SessionReasoningState is the per-session reasoning effort the manager keeps
// alongside each chat. The empty value (Effort=="" and Source==unset) is the
// "no override yet" baseline used when a session has never run /reasoning.
type SessionReasoningState struct {
	Effort ReasoningEffort
	Source string
}

// ReasoningReply is the apply-step result the manager renders back to the
// caller. Scope mirrors the post-apply state's source so callers don't need to
// know about the session-vs-global distinction beyond the reply.
type ReasoningReply struct {
	Effort        ReasoningEffort
	Scope         string
	PersistFailed bool
}
