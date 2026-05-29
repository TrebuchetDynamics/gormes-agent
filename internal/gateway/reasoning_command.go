package gateway

import gatewayreasoning "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/reasoning"

// ReasoningAction enumerates the parsed forms of the gateway /reasoning
// command. The parser is pure: state, persistence, and dispatch live in the
// follow-up apply/dispatch slice.
type ReasoningAction = gatewayreasoning.ReasoningAction

const (
	// ReasoningActionShow corresponds to /reasoning with no arguments.
	ReasoningActionShow = gatewayreasoning.ReasoningActionShow
	// ReasoningActionSet corresponds to /reasoning <effort> [--global].
	ReasoningActionSet = gatewayreasoning.ReasoningActionSet
	// ReasoningActionReset corresponds to /reasoning reset.
	ReasoningActionReset = gatewayreasoning.ReasoningActionReset
)

// ReasoningEffort is the validated effort level recognized by the parser.
// The empty value represents "no effort selected" for non-Set actions.
type ReasoningEffort = gatewayreasoning.ReasoningEffort

const (
	ReasoningEffortHigh   = gatewayreasoning.ReasoningEffortHigh
	ReasoningEffortLow    = gatewayreasoning.ReasoningEffortLow
	ReasoningEffortMedium = gatewayreasoning.ReasoningEffortMedium
)

// ReasoningCommand is the parsed shape of a /reasoning invocation.
type ReasoningCommand = gatewayreasoning.ReasoningCommand

// ErrInvalidEffort is returned when the user supplies an effort token outside
// the supported set (high|low|medium). The dispatcher renders this as the
// upstream "unknown argument" warning class.
var ErrInvalidEffort = gatewayreasoning.ErrInvalidEffort

// ErrResetGlobalUnsupported is returned when "reset" is combined with
// "--global". The dispatcher surfaces this verbatim because the upstream
// gateway rejects this combination too.
var ErrResetGlobalUnsupported = gatewayreasoning.ErrResetGlobalUnsupported

// Reasoning scope tags surfaced in ReasoningReply.Scope and stored in
// SessionReasoningState.Source. They distinguish session overrides from the
// persisted global default and the default-unset state.
const (
	ReasoningSourceUnset   = gatewayreasoning.ReasoningSourceUnset
	ReasoningSourceSession = gatewayreasoning.ReasoningSourceSession
	ReasoningSourceGlobal  = gatewayreasoning.ReasoningSourceGlobal
)

// SessionReasoningState is the per-session reasoning effort the manager keeps
// alongside each chat. The empty value (Effort=="" and Source==unset) is the
// "no override yet" baseline used when a session has never run /reasoning.
type SessionReasoningState = gatewayreasoning.SessionReasoningState

// ReasoningReply is the apply-step result the manager renders back to the
// caller. Scope mirrors the post-apply state's source so callers don't need to
// know about the session-vs-global distinction beyond the reply.
type ReasoningReply = gatewayreasoning.ReasoningReply

// ApplyReasoningCommand mutates the supplied SessionReasoningState according
// to a parsed ReasoningCommand. persistGlobal is invoked only for Set actions
// with Global=true; on failure the slice falls back to a session-only override
// and surfaces PersistFailed=true so the caller can warn the user.
func ApplyReasoningCommand(
	state SessionReasoningState,
	cmd ReasoningCommand,
	persistGlobal func(ReasoningEffort) error,
) (SessionReasoningState, ReasoningReply) {
	return gatewayreasoning.ApplyReasoningCommand(state, cmd, persistGlobal)
}

// ParseReasoningCommand turns the raw split arguments of /reasoning into a
// typed ReasoningCommand. It is pure: no I/O, no clock, no state.
func ParseReasoningCommand(args []string) (ReasoningCommand, error) {
	return gatewayreasoning.ParseReasoningCommand(args)
}
