package kernel

import (
	"errors"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// ErrResetDuringTurn is returned by Kernel.ResetSession when the kernel is
// not in a resettable phase (PhaseIdle or PhaseFailed). Preserves the
// Zero-Leak Invariant: in-flight turns are never truncated by reset.
var ErrResetDuringTurn = errors.New("kernel: cannot reset session during active turn")

// Phase is the kernel state-machine phase. Transitions happen only on the
// Run goroutine, serialised by the select loop.
type Phase int

const (
	PhaseIdle Phase = iota
	PhaseConnecting
	PhaseStreaming
	PhaseFinalizing
	PhaseCancelling
	PhaseFailed
	// PhaseReconnecting is the TDD seed for Phase-1.5 Route-B resilience
	// (spec §9.2 of 2026-04-18-gormes-frontend-adapter-design.md). No
	// transitions to this state exist yet — the future reconnect plan
	// flips reconnect_test.go from Skip to real pass by wiring this up.
	PhaseReconnecting
)

func (p Phase) String() string {
	return [...]string{"Idle", "Connecting", "Streaming", "Finalizing", "Cancelling", "Failed", "Reconnecting"}[p]
}

// RenderFrame is the only TUI input. The TUI never assembles assistant text
// from raw provider events; it renders this frame, full stop.
type RenderFrame struct {
	Seq             uint64
	Phase           Phase
	DraftText       string
	History         []hermes.Message
	Telemetry       telemetry.Snapshot
	StatusText      string
	SessionID       string
	Model           string
	ReasoningEffort hermes.ReasoningEffortEvidence
	ProviderStatus  hermes.ProviderStatus
	RetryStatus     RetryStatus
	LastError       string
	SoulEvents      []SoulEntry
	// ContextStatus snapshots the active ContextEngine status, when one is
	// configured. Nil means no context engine has been wired for this kernel.
	ContextStatus *hermes.ContextStatus

	// PanelState carries active modal panel data from the kernel to the TUI.
	// Nil/nil fields mean no panel is active. Only one panel may be active
	// at a time; when multiple are set the TUI follows the priority order:
	// Approval > Clarify > Secret.
	ApprovalState *KernelApprovalState
	ClarifyState  *KernelClarifyState
	SecretState   *KernelSecretState
}

type SoulEntry struct {
	At   time.Time
	Text string
}

// Mailbox capacities and timings. See spec §7.8 for the authoritative table.
const (
	RenderMailboxCap        = 1
	PlatformEventMailboxCap = 16

	FlushInterval    = 16 * time.Millisecond
	StoreAckDeadline = 250 * time.Millisecond
	ShutdownBudget   = 2 * time.Second
	SoulBufferSize   = 10
)

type PlatformEventKind int

const (
	PlatformEventSubmit PlatformEventKind = iota
	PlatformEventCancel
	PlatformEventQuit
	// PlatformEventResetSession clears k.history, k.sessionID, and
	// k.lastError. Valid from PhaseIdle and PhaseFailed; rejected with
	// ErrResetDuringTurn via the event's ack channel otherwise.
	PlatformEventResetSession
	// PlatformEventSteer carries operator guidance for an in-flight turn.
	// The kernel stores it only until the next completed tool-result batch,
	// then appends it to the final tool result before the next provider call.
	PlatformEventSteer
)

type PlatformEvent struct {
	Kind PlatformEventKind
	Text string
	// ContentParts carries multimodal user input alongside Text. When present,
	// providers that support native image content receive these parts; Text
	// remains the plain text projection for memory, recall, and legacy UI.
	ContentParts []hermes.MessageContentPart
	// Tools, when non-nil, overrides Config.Tools for this submit event only.
	// Gateway multi-agent routing uses this to expose the routed agent's
	// policy-filtered tool registry without mutating the resident kernel.
	Tools *tools.Registry
	// Skills, when non-nil, overrides Config.Skills for this submit event only.
	// Gateway multi-agent routing uses this to inject the routed agent's
	// skill allowlist while preserving the global runtime default.
	Skills SkillProvider
	// ToolSafety, when non-nil, is composed ahead of Config.ToolSafety for this
	// submit event only. A denial by either policy blocks execution.
	ToolSafety ToolSafetyPolicy
	// Model, when non-empty after trimming whitespace, overrides Config.Model
	// for this submit event only. The resident kernel configuration is not
	// mutated, so following turns fall back to Config.Model unless they carry
	// their own override.
	Model string
	// ReasoningEffort, when non-empty after trimming whitespace, overrides
	// Config.ReasoningEffort for this submit event only. The resident kernel
	// configuration is not mutated, so following turns fall back to the
	// configured/provider default unless they carry their own override.
	ReasoningEffort string
	// SessionID, when non-empty, overrides k.sessionID for this turn
	// only. Used by the Phase 2.D cron executor so each cron fire has
	// an isolated "cron:<job_id>:<unix_ts>" session. A non-cron event
	// leaves this empty and inherits k.sessionID as before. The
	// override is per-event — the kernel's resident sessionID is NOT
	// mutated; after the turn completes, the next non-cron event uses
	// whatever k.sessionID was before.
	SessionID string
	// SessionContext, when non-empty, is injected as the first system
	// message for this turn. Gateway frontends use it to describe the
	// current source chat and delivery options without mutating the
	// kernel's long-lived config.
	SessionContext string
	// CronJobID, when non-empty, causes the kernel to set cron=1 in the
	// AppendUserTurn payload, marking the persisted turn row as a cron
	// turn. The extractor (T3) uses this to skip cron turns during
	// entity extraction so agent-generated cron outputs don't corrupt
	// user representations. Opaque to the kernel — just passed through
	// to the store.Command payload.
	CronJobID string
	// ack is an unexported synchronous result channel used by
	// ResetSession. External callers constructing PlatformEvents for
	// Submit() cannot set this field, which is the desired API — the
	// synchronous ResetSession path is the only one that needs it.
	ack chan error
}

// =====================================================================
// Approval panel state — kernel → TUI contract
// =====================================================================

// ApprovalChoice enumerates the dangerous-command approval choices Hermes
// exposes. These values are the kernel-side representation; the TUI maps
// them to hermes_panels.go labels when rendering.
type ApprovalChoice int

const (
	ApprovalOnce ApprovalChoice = iota
	ApprovalSession
	ApprovalAlways
	ApprovalDeny
	ApprovalView
)

// KernelApprovalState is the approval panel data the kernel sends to the TUI.
// The TUI's hermes_panels.go converts this to ApprovalPanelState for rendering.
type KernelApprovalState struct {
	Description  string
	Command      string
	Choices      []ApprovalChoice
	Selected     ApprovalChoice
	ViewExpanded bool
	Width        int
	Height       int
}

// =====================================================================
// Clarify panel state — kernel → TUI contract
// =====================================================================

// KernelClarifyState is the clarify panel data the kernel sends to the TUI.
// The TUI's hermes_panels.go converts this to ClarifyPanelState for rendering.
type KernelClarifyState struct {
	Question    string
	Choices     []string
	Selected    int
	TimeoutHint string
	Width       int
	Height      int
}

// =====================================================================
// Secret/sudo panel state — kernel → TUI contract
// =====================================================================

// SecretPanelMode distinguishes sudo password from generic skill-secret capture.
type SecretPanelMode int

const (
	SecretPanelSudo SecretPanelMode = iota
	SecretPanelArbitrary
)

// KernelSecretState is the secret-panel data the kernel sends to the TUI.
// The TUI's hermes_panels.go converts this to SecretPanelState for rendering.
type KernelSecretState struct {
	PromptText string
	SecretLen  int
	Hint       string
	Countdown  time.Duration
	Mode       int
}
