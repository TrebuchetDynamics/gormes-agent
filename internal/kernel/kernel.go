// Package kernel is the single-owner state machine for Gormes. It owns the
// turn phase, the assistant draft buffer, the conversation history (in
// memory only in Phase 1), and the render snapshot. TUI, hermes, and store
// are edge adapters that communicate with the kernel through bounded mailboxes.
package kernel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/core/agent"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/plugins"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/fallback"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/lifecycle"
	kernelmodelrouting "github.com/TrebuchetDynamics/gormes-agent/internal/kernel/modelrouting"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/skills"
	kernelstreamdiag "github.com/TrebuchetDynamics/gormes-agent/internal/kernel/streamdiag"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// ErrEventMailboxFull is returned by Submit when the platform-event mailbox
// is saturated. The TUI should react by re-enabling input briefly; in
// practice this is rare with a 16-slot buffer.
var ErrEventMailboxFull = errors.New("kernel: event mailbox full")

// DefaultMaxToolIterations matches upstream Hermes' normal-turn default.
const DefaultMaxToolIterations = 90
const defaultMaxEmptyResponseRetries = 3

type FallbackClientFactory = fallback.ClientFactory

type Config struct {
	Model    string
	Provider string
	Endpoint string
	// ReasoningEffort, when non-empty, is the resident request default sent
	// on each supported provider request. Empty leaves reasoning to the
	// provider default and emits explicit default evidence in RenderFrame.
	ReasoningEffort string
	Admission       Admission
	Tools           *tools.Registry // nil → tool_calls are treated as fatal
	// SubdirectoryHints lazily discovers nested AGENTS.md/CLAUDE.md/.cursorrules
	// files from tool-call path arguments and appends them to tool results. Nil
	// defaults to a Hermes-compatible tracker for tool-capable kernels; set
	// DisableSubdirectoryHints to true to opt out in specialized tests/runtimes.
	SubdirectoryHints        *llm.SubdirectoryHintTracker
	DisableSubdirectoryHints bool
	MaxToolIterations        int           // default DefaultMaxToolIterations when zero
	MaxToolDuration          time.Duration // default 30s when zero
	// InitialSessionID primes k.sessionID at New() — used by adapters that
	// load a persisted session handle from internal/persistence/session before starting
	// the kernel. Zero value preserves pre-Phase-2.C behavior (fresh session).
	InitialSessionID string
	// ChatKey (Phase 3.C): "<platform>:<chat_id>" scope for memory recall.
	// Empty string = no scoping; recall queries skip chat filtering.
	ChatKey string
	// Recall (Phase 3.C) is optional. When non-nil, the kernel calls
	// GetContext before each turn and prepends a system message if the
	// returned string is non-empty. Nil = no memory injection (3.A/B
	// behavior preserved on platforms that don't opt in).
	Recall RecallProvider
	// RecallDeadline caps the GetContext call. Default 100ms when zero.
	// If GetContext misses the budget, its return value is discarded
	// and the turn proceeds without memory context.
	RecallDeadline time.Duration
	// Skills injects a deterministic procedural block ahead of the user turn.
	// Nil means no skill runtime.
	Skills SkillProvider
	// SystemPrompt is the base identity/context prompt prepended to every turn.
	// Empty preserves caller-owned prompt assembly for gateway paths that already
	// pass a session context block.
	SystemPrompt string
	// PrefillMessages are ephemeral few-shot messages inserted into provider
	// requests after system/context messages and before visible conversation.
	// They are never appended to kernel history or persisted.
	PrefillMessages []llm.Message
	// SkillUsage records selected skill names for later analysis. Nil disables
	// usage logging.
	SkillUsage SkillUsageRecorder
	// ToolAudit records append-only JSONL tool execution events when non-nil.
	ToolAudit audit.Recorder
	// ToolSafety can deterministically deny interactive or approval-gated tool
	// calls before registry lookup/execution.
	ToolSafety    ToolSafetyPolicy
	ContextEngine llm.ContextEngine
	Goncho        GonchoStore
	// MaxReconnectDuration caps the total wall-clock time spent retrying
	// per stream attempt (Route-B reconnect). Default 30s when zero.
	// Exceeding this budget fails the turn with "reconnect time budget exhausted".
	MaxReconnectDuration time.Duration
	// Fallback activates a Hermes-compatible fallback model/provider after
	// repeated empty provider responses. The factory is injected so tests and
	// runtime wiring can create provider clients without importing config here.
	Fallback                llm.FallbackModelPolicy
	FallbackClientFactory   FallbackClientFactory
	MaxEmptyResponseRetries int
	// AgentLifecycleHook, when non-nil, is called at agent:start (before the
	// tool loop), agent:step (after each tool batch with iteration count), and
	// agent:end (on turn exit with any error). The caller owns goroutine-safety.
	AgentLifecycleHook AgentLifecycleHook
	// TransformLLMOutput, when non-nil, runs after the tool-calling loop
	// completes but before the assistant response is committed to history.
	// Hooks receive the raw LLM output and may reshape, redact, or filter it.
	// First non-empty string result wins; errors are logged and original preserved.
	TransformLLMOutput plugins.TransformLLMOutputRunner
	// AgentMiddleware is an ordered chain of middlewares that run Before and After
	// each agent turn. Before hooks run in chain order before the turn starts;
	// a Before error aborts the turn. After hooks run in reverse chain order
	// after the turn completes (or fails). Nil means no middlewares are active.
	AgentMiddleware *agent.MiddlewareChain
	// BackgroundReview, when non-nil, is called after each successful turn to
	// run the Hermes-compatible memory/skill self-improvement review in a
	// daemon goroutine. The runner must be goroutine-safe. Nil = disabled.
	BackgroundReview BackgroundReviewSpawner
	// MemoryLifecycle, when non-nil, is invoked at the compression boundary to
	// give registered memory providers a chance to augment the compression
	// context hint via the Hermes-compatible PreCompress hook. Nil = disabled.
	MemoryLifecycle MemoryPreCompressor
	// MemoryPrefetch, when non-nil, is called before each turn to let
	// registered memory providers inject supplementary context (Hermes-compatible
	// pre-turn memory prefetch). The returned string is appended as a system
	// message after the existing Recall context. Nil = disabled.
	MemoryPrefetch MemoryPrefetcher
	// MemorySyncTurn, when non-nil, is called after each successful turn so
	// memory providers can persist the exchange (Hermes-compatible post-turn
	// SyncTurn hook). The call is fire-and-forget in a goroutine. Nil = disabled.
	MemorySyncTurn MemorySyncTurnWriter
}

// BackgroundReviewSpawner is the injection boundary for the post-turn
// background memory/skill review goroutine. Implementations must not block
// the caller; they own their own goroutine lifetime.
type BackgroundReviewSpawner interface {
	SpawnReview(ctx context.Context, messages []llm.Message, model, provider string)
}

// MemoryPreCompressor exposes the single hook called at the kernel compression
// boundary. Implementations receive the turn history and may return a context
// hint string that is appended to the compressor's FocusTopic. Nil = disabled.
type MemoryPreCompressor interface {
	// PreCompress returns an optional context hint for the compressor.
	// The hint is appended to the user-supplied focus topic. Empty = no-op.
	// Errors are logged and compression proceeds without the hint.
	PreCompress(ctx context.Context, messages []llm.Message) (string, error)
}

// MemoryPrefetcher exposes the Hermes-compatible pre-turn memory prefetch
// hook. When non-nil, the kernel calls Prefetch before each LLM request and
// injects the returned context string as a system message. Nil = disabled.
type MemoryPrefetcher interface {
	// Prefetch returns supplementary memory context for the given user query
	// and session. Empty = no-op. Errors are non-fatal; the turn proceeds.
	Prefetch(ctx context.Context, query, sessionID string) (string, error)
}

// MemorySyncTurnWriter exposes the Hermes-compatible post-turn SyncTurn hook.
// When non-nil, the kernel calls SyncTurn after each successful turn so memory
// providers can persist the exchange. Nil = disabled.
type MemorySyncTurnWriter interface {
	// SyncTurn persists one completed turn exchange. Errors are non-fatal.
	SyncTurn(ctx context.Context, userText, assistantText, platform, model, provider, sessionID string) error
}

type AgentLifecyclePoint = lifecycle.Point

const (
	AgentLifecycleStart = lifecycle.Start
	AgentLifecycleStep  = lifecycle.Step
	AgentLifecycleEnd   = lifecycle.End
)

type AgentLifecycleEvent = lifecycle.Event

type AgentLifecycleHook = lifecycle.Hook

type SkillProvider = skills.Provider

type SkillUsageRecorder = skills.UsageRecorder

type memorySyncSkipper interface {
	SkipMemorySync(ctx context.Context, turnKey, reason string) error
}

type Kernel struct {
	cfg    Config
	client llm.Client
	store  store.Store
	tm     telemetry.Telemetry
	log    *slog.Logger

	render chan RenderFrame
	events chan PlatformEvent

	// subscribers are independent render mailboxes handed out by Subscribe().
	// Unlike the single shared render channel, each gets its own copy of every
	// frame, so multiple consumers (e.g. the gateway plus cron runs sharing one
	// kernel) never steal frames from one another. Guarded by subMu.
	subMu       sync.Mutex
	subscribers map[int]chan RenderFrame
	nextSubID   int
	subsClosed  bool

	// Atomic — shared-read, kernel-write. Monotonically increasing per process.
	seq atomic.Uint64

	// All fields below this line are OWNED EXCLUSIVELY by the Run goroutine.
	// No other goroutine may read or write them without a channel-based
	// handshake. Violating this invariant is a race.
	phase           Phase
	draft           string
	draftReasoning  string // accumulated reasoning tokens for the current turn
	history         []llm.Message
	soul            []SoulEntry
	sessionID       string
	activeModel     string
	activeReasoning llm.ReasoningEffortEvidence
	lastError       string
	retryStatus     RetryStatus
	pendingSteers   []string
	// sessionModel / sessionProvider are the resident in-session override set
	// by PlatformEventSetModel. They survive across turns (unlike a per-event
	// PlatformEvent.Model) without clearing history. Empty means "no override
	// — fall back to cfg.Model". sessionProvider is recorded for the
	// cross-provider client-swap follow-up; same-provider switching is the
	// shipped core and reads sessionModel only.
	sessionModel    string
	sessionProvider string
}

func New(cfg Config, c llm.Client, s store.Store, tm telemetry.Telemetry, log *slog.Logger) *Kernel {
	if log == nil {
		log = slog.Default()
	}
	if cfg.Tools != nil && cfg.SubdirectoryHints == nil && !cfg.DisableSubdirectoryHints {
		cfg.SubdirectoryHints = llm.NewSubdirectoryHintTracker(llm.SubdirectoryHintOptions{WorkingDir: os.Getenv("TERMINAL_CWD")})
	}
	tm.SetModel(cfg.Model)
	if cfg.ContextEngine != nil {
		cfg.ContextEngine.UpdateModelContext(llm.ContextModelContext{Model: cfg.Model})
	}
	return &Kernel{
		cfg:         cfg,
		client:      c,
		store:       s,
		tm:          tm,
		log:         log,
		render:      make(chan RenderFrame, RenderMailboxCap),
		events:      make(chan PlatformEvent, PlatformEventMailboxCap),
		sessionID:   cfg.InitialSessionID,
		activeModel: cfg.Model,
		retryStatus: NewRetryStatus(),
	}
}

func (k *Kernel) liveTurnGuidanceBlocks(model string) []string {
	blocks := make([]string, 0, 6)
	if k.toolRegistered("memory") {
		blocks = append(blocks, llm.MemoryGuidance)
	}
	if k.toolRegistered("session_search") {
		blocks = append(blocks, llm.SessionSearchGuidance)
	}
	modelLower := strings.ToLower(strings.TrimSpace(model))
	if modelMatchesAny(modelLower, llm.ToolUseEnforcementModels) {
		blocks = append(blocks, llm.ToolUseEnforcementGuidance)
	}
	if modelMatchesAny(modelLower, []string{"gpt", "codex", "o1", "o3", "o4"}) {
		blocks = append(blocks, llm.OpenAIModelExecutionGuidance)
	} else if modelMatchesAny(modelLower, []string{"gemini", "gemma"}) {
		blocks = append(blocks, llm.GoogleModelOperationalGuidance)
	}
	if k.toolRegistered(tools.WebToolSearch) {
		blocks = append(blocks, llm.ResearchQualityGuidance)
	}
	return blocks
}

func (k *Kernel) toolRegistered(name string) bool {
	if k.cfg.Tools == nil {
		return false
	}
	_, ok := k.cfg.Tools.Get(name)
	return ok
}

func (k *Kernel) applySessionModel(ctx context.Context, e PlatformEvent) error {
	model := strings.TrimSpace(e.Model)
	provider := strings.TrimSpace(e.Provider)
	if k.shouldSwapSessionProvider(provider) {
		route, ok := sessionModelRoute(provider, model)
		if !ok {
			return fmt.Errorf("kernel: unknown model provider %q", provider)
		}
		if k.cfg.FallbackClientFactory == nil {
			return fmt.Errorf("kernel: cross-provider model switch requires fallback client factory")
		}
		client, err := k.cfg.FallbackClientFactory(ctx, route)
		if err != nil {
			return fmt.Errorf("kernel: switch model client: %w", err)
		}
		k.client = client
	}
	k.sessionModel = model
	k.sessionProvider = provider
	// Keep telemetry/context-engine model in sync with the new resident
	// override, mirroring New(). History is intentionally preserved (unlike
	// ResetSession): the conversation continues against the switched model.
	k.tm.SetModel(k.residentModel())
	if k.cfg.ContextEngine != nil {
		k.cfg.ContextEngine.UpdateModelContext(llm.ContextModelContext{Model: k.residentModel()})
	}
	return nil
}

func (k *Kernel) shouldSwapSessionProvider(next string) bool {
	return kernelmodelrouting.ShouldSwapProvider(k.sessionProvider, k.cfg.Provider, next)
}

func sessionModelRoute(provider, model string) (llm.ModelRoute, bool) {
	return kernelmodelrouting.Route(provider, model)
}

func sessionModelAPIMode(transport string) string {
	return kernelmodelrouting.APIMode(transport)
}

func firstNonEmpty(values ...string) string {
	return kernelmodelrouting.FirstNonEmpty(values...)
}

func modelMatchesAny(model string, needles []string) bool {
	return kernelmodelrouting.MatchesAny(model, needles)
}

// Render returns the receive side of the render mailbox. The channel is
// closed when Run exits.
func (k *Kernel) Render() <-chan RenderFrame { return k.render }

// Subscribe returns an independent render stream plus a function to release it.
// Each subscription has its own cap-RenderMailboxCap replace-latest mailbox, so
// multiple consumers do not compete on a single channel (which would let one
// steal frames meant for another). Call the returned func to unsubscribe; all
// subscriptions are also closed when Run exits. Safe to call from any goroutine.
func (k *Kernel) Subscribe() (<-chan RenderFrame, func()) {
	ch := make(chan RenderFrame, RenderMailboxCap)
	k.subMu.Lock()
	if k.subsClosed {
		k.subMu.Unlock()
		close(ch)
		return ch, func() {}
	}
	if k.subscribers == nil {
		k.subscribers = make(map[int]chan RenderFrame)
	}
	id := k.nextSubID
	k.nextSubID++
	k.subscribers[id] = ch
	k.subMu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			k.subMu.Lock()
			if c, ok := k.subscribers[id]; ok {
				delete(k.subscribers, id)
				close(c)
			}
			k.subMu.Unlock()
		})
	}
}

// closeSubscribers closes every live subscription and blocks future ones from
// receiving. Called once on Run exit.
func (k *Kernel) closeSubscribers() {
	k.subMu.Lock()
	k.subsClosed = true
	for id, ch := range k.subscribers {
		close(ch)
		delete(k.subscribers, id)
	}
	k.subMu.Unlock()
}

// fanOutFrame delivers frame to every subscription with the same replace-latest
// (drain-then-send) semantics as the shared render channel. Holds subMu so a
// concurrent unsubscribe cannot close a channel mid-send.
func (k *Kernel) fanOutFrame(frame RenderFrame) {
	k.subMu.Lock()
	for _, ch := range k.subscribers {
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- frame:
		default:
		}
	}
	k.subMu.Unlock()
}

// Submit enqueues a platform event. Returns ErrEventMailboxFull if the
// mailbox is saturated; the caller decides whether to retry or drop.
// Safe to call from any goroutine.
func (k *Kernel) Submit(e PlatformEvent) error {
	select {
	case k.events <- e:
		return nil
	default:
		return ErrEventMailboxFull
	}
}

// ResetSession clears the conversation history, server-assigned session id,
// and last error. Valid only from PhaseIdle or PhaseFailed; returns
// ErrResetDuringTurn if called during an in-flight turn. The Zero-Leak
// Invariant: never truncates streaming; callers must /stop first if they
// want to abandon an active turn.
//
// Implementation: enqueues a PlatformEventResetSession with a synchronous
// ack channel; the Run loop performs the mutation on its own goroutine,
// preserving the single-owner invariant. 500 ms ack timeout.
func (k *Kernel) ResetSession() error {
	ack := make(chan error, 1)
	select {
	case k.events <- PlatformEvent{Kind: PlatformEventResetSession, ack: ack}:
	default:
		return ErrEventMailboxFull
	}
	select {
	case err := <-ack:
		return err
	case <-time.After(500 * time.Millisecond):
		return errors.New("kernel: ResetSession ack timeout")
	}
}

// SetSessionModel installs a resident in-session model (and optional provider)
// override applied to subsequent turns via selectTurnModel precedence
// (per-event PlatformEvent.Model > resident session override > cfg.Model).
// Unlike ResetSession it does NOT clear history — the conversation continues
// against the new model. Valid only from PhaseIdle or PhaseFailed; returns
// ErrSetModelDuringTurn if called during an in-flight turn, with no resident
// mutation (the Zero-Leak Invariant: in-flight turns are never disturbed).
//
// Implementation mirrors ResetSession: enqueues a PlatformEventSetModel with a
// synchronous ack channel; the Run loop performs the mutation on its own
// goroutine, preserving the single-owner invariant. 500 ms ack timeout. The
// provider value is recorded for the cross-provider client-swap follow-up;
// same-provider model switching is the shipped core.
func (k *Kernel) SetSessionModel(provider, model string) error {
	ack := make(chan error, 1)
	select {
	case k.events <- PlatformEvent{Kind: PlatformEventSetModel, Provider: provider, Model: model, ack: ack}:
	default:
		return ErrEventMailboxFull
	}
	select {
	case err := <-ack:
		return err
	case <-time.After(500 * time.Millisecond):
		return errors.New("kernel: SetSessionModel ack timeout")
	}
}

// ResumeSession installs a resident session id and replayed visible history.
// It preserves the single-owner invariant by asking the Run loop to mutate
// state synchronously, and refuses to run while a turn is active.
func (k *Kernel) ResumeSession(sessionID string, history []llm.Message) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ErrResumeSessionIDRequired
	}
	ack := make(chan error, 1)
	select {
	case k.events <- PlatformEvent{Kind: PlatformEventResumeSession, SessionID: sessionID, History: cloneKernelMessages(history), ack: ack}:
	default:
		return ErrEventMailboxFull
	}
	select {
	case err := <-ack:
		return err
	case <-time.After(500 * time.Millisecond):
		return errors.New("kernel: ResumeSession ack timeout")
	}
}

// ManualCompress rewrites the resident visible history through the configured
// ContextEngine at an explicit operator boundary. It is synchronous for gateway
// and TUI command handlers and preserves the single-owner invariant by routing
// through the Run goroutine.
func (k *Kernel) ManualCompress(focus string) error {
	ack := make(chan error, 1)
	select {
	case k.events <- PlatformEvent{Kind: PlatformEventManualCompress, Text: focus, ack: ack}:
	default:
		return ErrEventMailboxFull
	}
	select {
	case err := <-ack:
		return err
	case <-time.After(30 * time.Second):
		return errors.New("kernel: ManualCompress ack timeout")
	}
}

// Run is the kernel loop. MUST be called from exactly one goroutine. Exits
// when ctx is cancelled or a PlatformEventQuit is received. Closes the
// render channel on exit.
func (k *Kernel) Run(ctx context.Context) error {
	defer close(k.render)
	defer k.closeSubscribers()
	k.emitFrame("idle")
	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-k.events:
			switch e.Kind {
			case PlatformEventSubmit:
				if k.phase != PhaseIdle && k.phase != PhaseFailed {
					k.lastError = ErrTurnInFlight.Error()
					k.emitFrame("still processing previous turn")
					continue
				}
				// Per-event sessionID override with defer-guarded restore.
				// The anonymous function gives defer a proper scope so the
				// restore fires after runTurn returns (or panics), not at
				// Run() exit.
				func() {
					prevSessionID := k.sessionID
					prevTools := k.cfg.Tools
					prevSkills := k.cfg.Skills
					prevToolSafety := k.cfg.ToolSafety
					if e.SessionID != "" {
						k.sessionID = e.SessionID
						defer func() { k.sessionID = prevSessionID }()
					}
					if e.Tools != nil {
						k.cfg.Tools = e.Tools
						defer func() { k.cfg.Tools = prevTools }()
					}
					if e.Skills != nil {
						k.cfg.Skills = e.Skills
						defer func() { k.cfg.Skills = prevSkills }()
					}
					if e.ToolSafety != nil {
						k.cfg.ToolSafety = ComposeToolSafetyPolicies(e.ToolSafety, prevToolSafety)
						defer func() { k.cfg.ToolSafety = prevToolSafety }()
					}
					turnCtx := ctx
					if strings.TrimSpace(e.CronApprovalMode) != "" {
						turnCtx = tools.WithCronApprovalMode(turnCtx, e.CronApprovalMode)
					}
					k.runTurn(turnCtx, e.Text, e.ContentParts, e.SessionContext, e.CronJobID, selectTurnModel(k.residentModel(), e.Model), e.ReasoningEffort)
				}()
			case PlatformEventCancel:
				// No active turn; ignore (cancel during a turn is handled
				// inside runTurn's select on k.events).
			case PlatformEventResetSession:
				if k.phase != PhaseIdle && k.phase != PhaseFailed {
					if e.ack != nil {
						e.ack <- ErrResetDuringTurn
					}
					continue
				}
				if k.cfg.ContextEngine != nil {
					outgoingHistory := append([]llm.Message(nil), k.history...)
					_ = k.cfg.ContextEngine.OnSessionEnd(ctx, k.sessionID, outgoingHistory)
					k.cfg.ContextEngine.OnSessionReset()
				}
				k.history = nil
				k.sessionID = ""
				k.lastError = ""
				k.phase = PhaseIdle
				k.emitFrame("session reset")
				if e.ack != nil {
					e.ack <- nil
				}
			case PlatformEventSetModel:
				if k.phase != PhaseIdle && k.phase != PhaseFailed {
					if e.ack != nil {
						e.ack <- ErrSetModelDuringTurn
					}
					continue
				}
				if err := k.applySessionModel(ctx, e); err != nil {
					k.lastError = err.Error()
					k.emitFrame("model switch failed")
					if e.ack != nil {
						e.ack <- err
					}
					continue
				}
				k.emitFrame("model switched")
				if e.ack != nil {
					e.ack <- nil
				}
			case PlatformEventResumeSession:
				if k.phase != PhaseIdle && k.phase != PhaseFailed {
					if e.ack != nil {
						e.ack <- ErrResumeDuringTurn
					}
					continue
				}
				sessionID := strings.TrimSpace(e.SessionID)
				if sessionID == "" {
					if e.ack != nil {
						e.ack <- ErrResumeSessionIDRequired
					}
					continue
				}
				if k.cfg.ContextEngine != nil {
					outgoingHistory := append([]llm.Message(nil), k.history...)
					_ = k.cfg.ContextEngine.OnSessionEnd(ctx, k.sessionID, outgoingHistory)
					k.cfg.ContextEngine.OnSessionReset()
				}
				k.sessionID = sessionID
				k.history = cloneKernelMessages(e.History)
				k.draft = ""
				k.lastError = ""
				k.phase = PhaseIdle
				k.emitFrame("session resumed")
				if e.ack != nil {
					e.ack <- nil
				}
			case PlatformEventManualCompress:
				if k.phase != PhaseIdle && k.phase != PhaseFailed {
					if e.ack != nil {
						e.ack <- ErrCompressDuringTurn
					}
					continue
				}
				if k.cfg.ContextEngine == nil {
					if e.ack != nil {
						e.ack <- ErrCompressionUnavailable
					}
					continue
				}
				before := cloneKernelMessages(k.history)
				currentTokens := 0
				if status := k.cfg.ContextEngine.Status(); status.LastTotalTokens > 0 {
					currentTokens = status.LastTotalTokens
				}
				// Call Hermes-compatible PreCompress hook so memory providers can
				// inject context hints into the compression request.
				focusTopic := strings.TrimSpace(e.Text)
				if lc := k.cfg.MemoryLifecycle; lc != nil {
					if hint, _ := lc.PreCompress(ctx, before); strings.TrimSpace(hint) != "" {
						if focusTopic != "" {
							focusTopic = focusTopic + "\n\n" + hint
						} else {
							focusTopic = hint
						}
					}
				}
				after, _, err := k.cfg.ContextEngine.Compress(ctx, before, llm.CompressionRequest{CurrentTokens: currentTokens, FocusTopic: focusTopic})
				if err != nil {
					k.lastError = err.Error()
					k.emitFrame("compression failed")
					if e.ack != nil {
						e.ack <- err
					}
					continue
				}
				k.history = cloneKernelMessages(after)
				k.lastError = ""
				_ = k.cfg.ContextEngine.OnCompressionBoundary(ctx, llm.CompressionBoundary{OldSessionID: k.sessionID, NewSessionID: k.sessionID, Reason: "manual_compress"})
				k.emitFrame("session compressed")
				if e.ack != nil {
					e.ack <- nil
				}
			case PlatformEventQuit:
				return nil
			case PlatformEventSteer:
				// No active turn or tool boundary: ignore here. Gateway keeps
				// its existing queue/degraded fallback for non-active turns.
			}
		}
	}
}

// runTurn handles exactly one user turn end-to-end. On entry k.phase must be
// PhaseIdle; on exit it is PhaseIdle (or PhaseFailed on a fatal error).
// All state mutations happen on the calling goroutine, which is the Run
// goroutine — this is part of the single-owner invariant.
// cronJobID is non-empty for Phase 2.D cron-fired turns; it is passed through
// to the store.Command payload and is otherwise opaque to the kernel.
func (k *Kernel) runTurn(ctx context.Context, text string, contentParts []llm.MessageContentPart, sessionContext, cronJobID, model, reasoningOverride string) {
	prov := newProvenance(k.cfg.Endpoint)
	defer func() { k.pendingSteers = nil }()

	// 0a. Middleware chain: Before hooks (chain order).
	if chain := k.cfg.AgentMiddleware; chain != nil {
		mwCtx := &agent.MiddlewareContext{
			ThreadID:  k.cfg.ChatKey,
			SessionID: k.sessionID,
			Model:     model,
		}
		if err := chain.Before(mwCtx); err != nil {
			k.lastError = err.Error()
			k.emitFrame(err.Error())
			return
		}
		defer func() {
			if afterErr := chain.After(mwCtx); afterErr != nil {
				k.lastError = afterErr.Error()
				k.emitFrame("middleware after error: " + afterErr.Error())
			}
		}()
	}

	if k.cfg.AgentLifecycleHook != nil {
		sid := k.sessionID
		defer func() {
			var turnErr error
			if k.lastError != "" {
				turnErr = errors.New(k.lastError)
			}
			k.cfg.AgentLifecycleHook(ctx, AgentLifecycleEvent{
				Point:     AgentLifecycleEnd,
				SessionID: sid,
				Err:       turnErr,
			})
		}()
	}
	turnKey := prov.LocalRunID
	model = selectTurnModel(k.cfg.Model, model)
	providerStatus := llm.ProviderStatusOf(k.client)
	reasoningEvidence := selectTurnReasoningEffort(k.cfg.ReasoningEffort, reasoningOverride, providerStatus)
	k.soul = nil

	// 1. Admission. Reject locally before any HTTP.
	if err := validateTurnAdmission(k.cfg.Admission, text, contentParts); err != nil {
		k.lastError = err.Error()
		k.emitFrame(err.Error())
		return
	}
	prov.LogAdmitted(k.log)
	userMsg := llm.Message{Role: "user", Content: text, ContentParts: cloneMessageContentParts(contentParts)}

	// 2. Persist user turn with hard 250ms ack deadline (spec §7.8 store row).
	storeCtx, storeCancel := context.WithTimeout(ctx, StoreAckDeadline)
	userPayload, _ := json.Marshal(map[string]any{
		"session_id":         k.sessionID,
		"content":            text,
		"ts_unix":            time.Now().Unix(),
		"chat_id":            k.cfg.ChatKey,
		"cron":               cronFlag(cronJobID),
		"cron_job_id":        cronJobID,
		"turn_key":           turnKey,
		"memory_sync_status": "pending",
	})
	_, err := k.store.Exec(storeCtx, store.Command{Kind: store.AppendUserTurn, Payload: userPayload})
	storeCancel()
	if err != nil {
		k.phase = PhaseFailed
		k.lastError = fmt.Sprintf("store ack timeout: %v", err)
		k.emitFrame(k.lastError)
		return
	}

	// Persist user message to Goncho for cross-session memory and evidence.
	k.writeGonchoUserTurn(ctx, text, turnKey)

	// 3. Update state for the new turn. These mutations are safe because we
	// are on the Run goroutine.
	k.history = append(k.history, userMsg)
	k.draft = ""
	k.draftReasoning = ""
	k.lastError = ""
	k.retryStatus = NewRetryStatus()
	k.activeModel = model
	k.activeReasoning = reasoningEvidence
	k.phase = PhaseConnecting
	k.emitFrame("connecting")
	prov.LogPOSTSent(k.log)

	// 4. Tool loop — wraps the Route-B retry loop. On finish_reason=="tool_calls"
	// we execute the tools in-process and issue a follow-up stream with the
	// tool results appended to the message history. Capped at MaxToolIterations
	// to prevent runaway agent loops.
	request := k.buildTurnRequest(ctx, turnRequestAssemblyInput{
		Model:          model,
		SessionID:      k.sessionID,
		UserText:       text,
		UserMessage:    userMsg,
		SessionContext: sessionContext,
		Reasoning:      reasoningEvidence,
	})
	primaryClient := k.client
	defer func() { k.client = primaryClient }()
	fallbackRoutes := append([]llm.ModelRoute(nil), k.cfg.Fallback.Routes...)
	if !k.cfg.Fallback.Enabled {
		fallbackRoutes = nil
	}
	fallbackIndex := 0
	emptyResponses := 0
	maxEmptyResponses := k.maxEmptyResponseRetries()
	maxIter := k.cfg.MaxToolIterations
	if maxIter <= 0 {
		maxIter = DefaultMaxToolIterations
	}

	var (
		cancelled       bool
		fatalErr        error
		finalDelta      llm.Event
		gotFinal        bool
		latestSessionID string
		toolIteration   = 0
		toolCallsSeen   []llm.ToolCall
	)

	start := time.Now()
	k.tm.StartTurn()
	if k.cfg.AgentLifecycleHook != nil {
		k.cfg.AgentLifecycleHook(ctx, AgentLifecycleEvent{
			Point:     AgentLifecycleStart,
			SessionID: k.sessionID,
		})
	}

toolLoop:
	for {
		// Fresh retry budget each tool iteration — reconnect retries are for
		// network drops, not for multi-round agent reasoning.
		retryBudget := NewRetryBudget()
		reconnectStart := time.Now()
		maxReconnect := k.cfg.MaxReconnectDuration
		if maxReconnect <= 0 {
			maxReconnect = 30 * time.Second
		}
		var replaceOnNextToken bool

	retryLoop:
		for {
			runCtx, cancelRun := context.WithCancel(ctx)

			stream, err := k.client.OpenStream(runCtx, request)
			if err != nil {
				cancelRun()
				classification := llm.ClassifyProviderError(err)
				if classification.Class == llm.ClassRetryable && !retryBudget.Exhausted() && time.Since(reconnectStart) < maxReconnect {
					if k.retryInterrupted(ctx) {
						cancelled = true
						break toolLoop
					}
					decision := retryBudget.NextDelayDecision(err)
					k.retryStatus = retryStatusWithDecision(k.retryStatus, decision, classification)
					k.phase = PhaseReconnecting
					k.emitStreamDropRetryFrame(err, decision, classification, false, llm.StreamDiagnosticsFromError(err))
					if k.waitForRetryDelay(ctx, decision.Delay) {
						cancelled = true
						break toolLoop
					}
					replaceOnNextToken = true
					continue retryLoop
				}
				if classification.Class == llm.ClassRetryable && time.Since(reconnectStart) >= maxReconnect {
					k.retryStatus.LastDecision = RetryDecisionBudgetExhaust
					k.phase = PhaseFailed
					k.lastError = "reconnect time budget exhausted"
					k.emitFrame("reconnect time budget exhausted")
					k.log.Warn("kernel: reconnect time budget exceeded",
						"elapsed", time.Since(reconnectStart).String(),
						"max", maxReconnect.String(),
					)
					return
				}
				prov.ErrorClass = llm.Classify(err).String()
				prov.ErrorText = err.Error()
				prov.LogError(k.log)
				k.phase = PhaseFailed
				k.lastError = err.Error()
				k.activeModel = k.cfg.Model
				k.activeReasoning = llm.ReasoningEffortEvidence{}
				k.emitFrame("open stream failed")
				return
			}

			k.phase = PhaseStreaming
			k.emitFrame("streaming")

			var streamRetryErr error
			var streamRetryDiag llm.StreamDiagnostics
			outcome := k.streamInner(ctx, runCtx, cancelRun, stream, &finalDelta, &gotFinal, &fatalErr, &cancelled, &replaceOnNextToken, &streamRetryErr, &streamRetryDiag)
			_ = stream.Close()
			if sid := stream.SessionID(); sid != "" {
				latestSessionID = sid
			}
			cancelRun()

			switch outcome {
			case streamOutcomeDone:
				break retryLoop
			case streamOutcomeCancelled:
				break toolLoop
			case streamOutcomeFatal:
				break toolLoop
			case streamOutcomeRetryable:
				if k.retryInterrupted(ctx) {
					cancelled = true
					break toolLoop
				}
				if retryBudget.Exhausted() {
					k.retryStatus.LastDecision = RetryDecisionBudgetExhaust
					k.phase = PhaseFailed
					k.lastError = "reconnect budget exhausted"
					k.emitFrame("reconnect budget exhausted")
					return
				}
				if time.Since(reconnectStart) >= maxReconnect {
					k.retryStatus.LastDecision = RetryDecisionBudgetExhaust
					k.phase = PhaseFailed
					k.lastError = "reconnect time budget exhausted"
					k.emitFrame("reconnect time budget exhausted")
					k.log.Warn("kernel: reconnect time budget exceeded",
						"elapsed", time.Since(reconnectStart).String(),
						"max", maxReconnect.String(),
					)
					return
				}
				decision := retryBudget.NextDelayDecision(streamRetryErr)
				classification := streamDropRetryClassification(streamRetryErr)
				k.retryStatus = retryStatusWithDecision(k.retryStatus, decision, classification)
				k.phase = PhaseReconnecting
				k.emitStreamDropRetryFrame(streamRetryErr, decision, classification, true, streamRetryDiag)
				if k.waitForRetryDelay(ctx, decision.Delay) {
					cancelled = true
					break toolLoop
				}
				replaceOnNextToken = true
				continue retryLoop
			}
		}

		// retryLoop exited cleanly (EventDone received). Inspect finish_reason.
		if !gotFinal {
			fatalErr = fmt.Errorf("stream closed without finish_reason")
			break toolLoop
		}
		k.updateContextEngineUsage(finalDelta)
		if len(fallbackRoutes) > 0 && emptyFinalResponse(k.draft, finalDelta) {
			emptyResponses++
			if emptyResponses <= maxEmptyResponses {
				gotFinal = false
				finalDelta = llm.Event{}
				k.phase = PhaseReconnecting
				k.emitFrame("empty response retry")
				continue toolLoop
			}
			if k.activateFallback(ctx, &request, fallbackRoutes, &fallbackIndex) {
				emptyResponses = 0
				gotFinal = false
				finalDelta = llm.Event{}
				continue toolLoop
			}
		}

		if finalDelta.FinishReason != "tool_calls" {
			// Normal end of turn. Exit the tool loop to finalise.
			break toolLoop
		}
		toolCallsSeen = append(toolCallsSeen, finalDelta.ToolCalls...)

		// tool_calls round. Execute tools and append results to the request.
		toolIteration++
		if toolIteration > maxIter {
			summaryDelta, summaryDone, summarySessionID, summaryCancelled := k.requestMaxIterationSummary(ctx, request, maxIter)
			if summarySessionID != "" {
				latestSessionID = summarySessionID
			}
			if summaryDone {
				finalDelta = summaryDelta
				gotFinal = true
				k.updateContextEngineUsage(summaryDelta)
			}
			cancelled = summaryCancelled
			break toolLoop
		}

		runCtx, cancelRun := context.WithCancel(ctx)
		toolOutcome := k.executeToolCallsInterruptible(runCtx, finalDelta.ToolCalls, turnKey)
		cancelRun()
		if toolOutcome.Cancelled {
			cancelled = true
			break toolLoop
		}
		results := toolOutcome.Results
		k.applyPendingSteersToToolResults(results)

		if k.cfg.AgentLifecycleHook != nil {
			names := make([]string, len(finalDelta.ToolCalls))
			for i, tc := range finalDelta.ToolCalls {
				names[i] = tc.Name
			}
			k.cfg.AgentLifecycleHook(ctx, AgentLifecycleEvent{
				Point:     AgentLifecycleStep,
				SessionID: k.sessionID,
				Iteration: toolIteration,
				ToolNames: names,
			})
		}

		// Append the assistant's tool-requesting message plus one tool-result
		// message per call. The draft so far is captured in the assistant
		// message.
		assistantMsg := llm.Message{
			Role:      "assistant",
			Content:   k.draft,
			ToolCalls: finalDelta.ToolCalls,
		}
		request.Messages = append(request.Messages, assistantMsg)
		for _, r := range results {
			request.Messages = append(request.Messages, llm.Message{
				Role:         "tool",
				ToolCallID:   r.ID,
				Name:         r.Name,
				Content:      r.Content,
				ContentParts: cloneMessageContentParts(r.ContentParts),
			})
		}

		// Clear draft between tool iterations — the next LLM response is a
		// fresh continuation; the assistant message we appended captures
		// what we had so far.
		k.draft = ""
		gotFinal = false
		finalDelta = llm.Event{}
		k.emitFrame("executing tools")
	}

	// 5. Finalisation (unchanged shape from Route-B).
	latency := time.Since(start)
	k.tm.FinishTurn(latency)
	prov.LatencyMs = int(latency / time.Millisecond)

	if fatalErr != nil {
		prov.ErrorClass = llm.Classify(fatalErr).String()
		prov.ErrorText = fatalErr.Error()
		prov.LogError(k.log)
		k.phase = PhaseFailed
		k.lastError = fatalErr.Error()
		k.activeModel = k.cfg.Model
		k.activeReasoning = llm.ReasoningEffortEvidence{}
		k.emitFrame("stream error")
		return
	}

	if gotFinal {
		prov.FinishReason = finalDelta.FinishReason
		prov.TokensIn = finalDelta.TokensIn
		prov.TokensOut = finalDelta.TokensOut
		if finalDelta.TokensIn > 0 {
			k.tm.SetTokensIn(finalDelta.TokensIn)
		}
		if finalDelta.TokensOut > 0 {
			k.tm.Tick(finalDelta.TokensOut)
		}
	}

	if latestSessionID != "" {
		k.sessionID = latestSessionID
		prov.ServerSessionID = latestSessionID
		prov.LogSSEStart(k.log)
	}

	if cancelled {
		k.skipMemorySync(turnKey, "interrupted")
		k.phase = PhaseCancelling
		k.emitFrame("cancelled")
	} else if k.draft != "" {
		finalContent := k.draft
		if k.cfg.TransformLLMOutput != nil {
			finalContent = k.cfg.TransformLLMOutput.Run(ctx, plugins.TransformLLMOutputInput{
				ResponseText: finalContent,
				SessionID:    k.sessionID,
				Model:        k.activeModel,
				Platform:     platformFromChatKey(k.cfg.ChatKey),
			})
		}
		assistantMsg := llm.Message{Role: "assistant", Content: finalContent}
		if k.draftReasoning != "" {
			assistantMsg.Reasoning = &llm.ReasoningContent{Text: k.draftReasoning}
		}
		k.history = append(k.history, assistantMsg)
		k.draft = ""
		k.draftReasoning = ""
		// Phase 3.A: finalize in the memory store. Fire-and-forget — the worker
		// handles I/O off the hot path. 250ms context bound kept as a safety net
		// in case someone injects a synchronous store in the future.
		payload := map[string]any{
			"session_id":         k.sessionID,
			"content":            finalContent,
			"ts_unix":            time.Now().Unix(),
			"chat_id":            k.cfg.ChatKey,
			"turn_key":           turnKey,
			"memory_sync_status": "ready",
		}
		if len(toolCallsSeen) > 0 {
			meta, _ := json.Marshal(map[string]any{"tool_calls": toolCallsSeen})
			payload["meta_json"] = string(meta)
		}
		finalPayload, _ := json.Marshal(payload)
		finalCtx, finalCancel := context.WithTimeout(ctx, StoreAckDeadline)
		if _, err := k.store.Exec(finalCtx, store.Command{Kind: store.FinalizeAssistantTurn, Payload: finalPayload}); err != nil {
			k.log.Warn("kernel: store exec FinalizeAssistantTurn failed", "err", err)
		}
		finalCancel()
		// Persist assistant response to Goncho for cross-session memory and evidence.
		k.writeGonchoAssistantTurn(ctx, finalContent, turnKey)
	}

	prov.LogDone(k.log)
	k.client = primaryClient
	// Spawn background memory/skill review (Hermes-compatible post-turn fork).
	if k.lastError == "" && k.cfg.BackgroundReview != nil {
		snapshot := cloneKernelMessages(k.history)
		rev := k.cfg.BackgroundReview
		turnModel := model
		turnProv := k.cfg.Provider
		go rev.SpawnReview(context.Background(), snapshot, turnModel, turnProv)
	}
	// Persist turn to memory providers (Hermes-compatible SyncTurn post-turn hook).
	if k.lastError == "" && !cancelled && k.cfg.MemorySyncTurn != nil {
		syncText := text
		syncAssistant := ""
		if n := len(k.history); n > 0 && k.history[n-1].Role == "assistant" {
			syncAssistant = k.history[n-1].Content
		}
		syncWriter := k.cfg.MemorySyncTurn
		syncModel := model
		syncProv := k.cfg.Provider
		syncPlatform := platformFromChatKey(k.cfg.ChatKey)
		syncSession := k.sessionID
		go func() {
			_ = syncWriter.SyncTurn(context.Background(), syncText, syncAssistant, syncPlatform, syncModel, syncProv, syncSession)
		}()
	}
	k.phase = PhaseIdle
	k.activeModel = k.cfg.Model
	k.activeReasoning = llm.ReasoningEffortEvidence{}
	k.pendingSteers = nil
	k.emitFrame("idle")
}

func validateTurnAdmission(admission Admission, text string, parts []llm.MessageContentPart) error {
	payload := text
	hasImage := false
	for _, part := range parts {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case "text", "input_text", "output_text":
			payload += "\n" + part.Text
		case "image_url", "input_image", "image":
			// Image data URIs / URLs do not count toward the text admission
			// byte limit. Image payload size is governed separately by the
			// provider-side image_shrink_retry path; admission gates user
			// text input only. A 243 KB JPEG becomes ~325 KB base64, which
			// would otherwise blow past the default MaxBytes (200_000) and
			// silently reject every photo turn from a channel.
			hasImage = true
		}
	}
	if strings.TrimSpace(payload) == "" && !hasImage {
		return ErrEmptyInput
	}
	if admission.MaxBytes > 0 && len(payload) > admission.MaxBytes {
		return ErrInputTooLarge
	}
	if admission.MaxLines > 0 && strings.Count(payload, "\n")+1 > admission.MaxLines {
		return ErrTooManyLines
	}
	return nil
}

func cloneMessageContentParts(parts []llm.MessageContentPart) []llm.MessageContentPart {
	if len(parts) == 0 {
		return nil
	}
	return append([]llm.MessageContentPart(nil), parts...)
}

func cloneKernelMessages(messages []llm.Message) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]llm.Message, len(messages))
	for i, msg := range messages {
		out[i] = msg
		out[i].ContentParts = cloneMessageContentParts(msg.ContentParts)
	}
	return out
}

const maxIterationSummaryRequest = "You've reached the maximum number of tool-calling iterations allowed. Please provide a final response summarizing what you've found and accomplished so far, without calling any more tools."

func (k *Kernel) requestMaxIterationSummary(ctx context.Context, base llm.ChatRequest, maxIter int) (llm.Event, bool, string, bool) {
	req := base
	req.Stream = true
	req.Tools = nil
	req.Messages = append(append([]llm.Message(nil), base.Messages...), llm.Message{
		Role:    "user",
		Content: maxIterationSummaryRequest,
	})

	// Discard any partial text streamed during the over-budget tool round. The
	// summary is a fresh, standalone response and replaces that interrupted
	// fragment; without this reset streamInner would append the summary onto the
	// leftover draft and persist a malformed mixed final message.
	k.draft = ""
	k.lastError = ""
	k.phase = PhaseFinalizing
	k.emitFrame(fmt.Sprintf("iteration budget exhausted (%d/%d); requesting summary", maxIter, maxIter))

	runCtx, cancelRun := context.WithCancel(ctx)
	stream, err := k.client.OpenStream(runCtx, req)
	if err != nil {
		cancelRun()
		k.draft = fmt.Sprintf("I reached the maximum iterations (%d) but couldn't summarize. Error: %s", maxIter, err)
		k.emitFrame("iteration summary failed")
		return llm.Event{}, false, "", false
	}

	k.phase = PhaseStreaming
	k.emitFrame("streaming iteration summary")
	var (
		finalDelta         llm.Event
		gotFinal           bool
		fatalErr           error
		cancelled          bool
		replaceOnNextToken bool
	)
	outcome := k.streamInner(ctx, runCtx, cancelRun, stream, &finalDelta, &gotFinal, &fatalErr, &cancelled, &replaceOnNextToken, nil, nil)
	summarySessionID := stream.SessionID()
	_ = stream.Close()

	if cancelled || outcome == streamOutcomeCancelled {
		return finalDelta, gotFinal, summarySessionID, true
	}
	if fatalErr != nil && strings.TrimSpace(k.draft) == "" {
		k.draft = fmt.Sprintf("I reached the maximum iterations (%d) but couldn't summarize. Error: %s", maxIter, fatalErr)
		k.emitFrame("iteration summary failed")
		return llm.Event{}, false, summarySessionID, false
	}
	if strings.TrimSpace(k.draft) == "" {
		k.draft = "I reached the iteration limit and couldn't generate a summary."
		k.emitFrame("iteration summary unavailable")
		return finalDelta, gotFinal, summarySessionID, false
	}
	return finalDelta, gotFinal, summarySessionID, false
}

const steerToolResultMarker = "\n\n[Gormes operator /steer guidance before next provider call]\n"

func (k *Kernel) queueSteerGuidance(text string) {
	guidance := strings.TrimSpace(text)
	if guidance == "" {
		return
	}
	const maxSteerGuidanceBytes = 4096
	if len(guidance) > maxSteerGuidanceBytes {
		guidance = guidance[:maxSteerGuidanceBytes]
	}
	k.pendingSteers = append(k.pendingSteers, guidance)
	k.addSoul("steer queued")
	k.emitFrame("steer queued")
}

func (k *Kernel) applyPendingSteersToToolResults(results []toolResult) {
	if len(k.pendingSteers) == 0 || len(results) == 0 {
		return
	}
	guidance := strings.Join(k.pendingSteers, "\n\n")
	last := len(results) - 1
	results[last].Content += steerToolResultMarker + guidance
	k.pendingSteers = nil
	k.addSoul("steer applied")
	k.emitFrame("steer applied")
}

func (k *Kernel) updateContextEngineUsage(ev llm.Event) {
	if k.cfg.ContextEngine == nil {
		return
	}
	total := ev.TokensIn + ev.TokensOut
	k.cfg.ContextEngine.UpdateFromResponse(llm.ContextUsage{
		PromptTokens:     ev.TokensIn,
		CompletionTokens: ev.TokensOut,
		TotalTokens:      total,
	})
}

type streamOutcome int

const (
	streamOutcomeDone streamOutcome = iota
	streamOutcomeCancelled
	streamOutcomeRetryable
	streamOutcomeFatal
)

func (k *Kernel) waitForRetryDelay(ctx context.Context, d time.Duration) bool {
	if k.retryInterrupted(ctx) {
		return true
	}
	if d <= 0 {
		return k.retryInterrupted(ctx)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return true
		case e := <-k.events:
			if k.handleRetryEvent(e) {
				return true
			}
		case <-timer.C:
			return k.retryInterrupted(ctx)
		}
	}
}

func (k *Kernel) retryInterrupted(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	for {
		select {
		case e := <-k.events:
			if k.handleRetryEvent(e) {
				return true
			}
		default:
			return false
		}
	}
}

func (k *Kernel) handleRetryEvent(e PlatformEvent) bool {
	switch e.Kind {
	case PlatformEventCancel, PlatformEventQuit:
		return true
	case PlatformEventSubmit:
		k.lastError = ErrTurnInFlight.Error()
		k.emitFrame("still processing previous turn")
	case PlatformEventResetSession:
		if e.ack != nil {
			e.ack <- ErrResetDuringTurn
		}
	case PlatformEventSetModel:
		// Zero-Leak Invariant: never disturb an in-flight turn. Reject the
		// switch without mutating the resident override.
		if e.ack != nil {
			e.ack <- ErrSetModelDuringTurn
		}
	case PlatformEventManualCompress:
		if e.ack != nil {
			e.ack <- ErrCompressDuringTurn
		}
	case PlatformEventSteer:
		k.queueSteerGuidance(e.Text)
	}
	return false
}

func (k *Kernel) emitStreamDropRetryFrame(err error, decision RetryDelayDecision, classification llm.ProviderErrorClassification, midStream bool, diag llm.StreamDiagnostics) {
	provider := k.streamDropProviderName()
	errorType := streamDropErrorType(err)
	errorText := streamDropErrorText(err)
	errorClass := classification.Class.String()
	errorKind := classification.Kind.String()
	dropKind := "stream drop"
	if midStream {
		dropKind = "stream drop mid-stream"
	}
	k.lastError = dropKind + ": " + errorText
	attrs := []any{
		"provider", provider,
		"endpoint", k.cfg.Endpoint,
		"attempt", decision.Attempt,
		"max_attempts", maxRetryAttempts,
		"error_type", errorType,
		"error_chain", streamDropErrorChain(err),
		"error_class", errorClass,
		"error_kind", errorKind,
		"mid_stream", midStream,
		"error", errorText,
	}
	if diag.HTTPStatus > 0 {
		attrs = append(attrs, "http_status", diag.HTTPStatus)
	}
	if diag.Bytes > 0 {
		attrs = append(attrs, "bytes", diag.Bytes)
	}
	if diag.Chunks > 0 {
		attrs = append(attrs, "chunks", diag.Chunks)
	}
	if diag.Elapsed > 0 {
		attrs = append(attrs, "elapsed", diag.Elapsed.Round(time.Millisecond))
	}
	if diag.TimeToFirstByte > 0 {
		attrs = append(attrs, "ttfb", diag.TimeToFirstByte.Round(time.Millisecond))
	}
	if headers := formatStreamDiagnosticHeaders(diag.Headers); headers != "" {
		attrs = append(attrs, "upstream_headers", headers)
	}
	k.log.Warn("kernel stream drop retry", attrs...)
	elapsedSuffix := ""
	if diag.Elapsed > 0 {
		elapsedSuffix = fmt.Sprintf(" after %.1fs", diag.Elapsed.Seconds())
	}
	k.emitFrame(fmt.Sprintf("%s %s (%s/%s)%s - reconnecting, retry %d/%d",
		provider,
		dropKind,
		errorKind,
		errorType,
		elapsedSuffix,
		decision.Attempt,
		maxRetryAttempts,
	))
}

func (k *Kernel) streamDropProviderName() string {
	status := llm.ProviderStatusOf(k.client)
	provider := strings.TrimSpace(status.Provider)
	if provider == "" {
		return "provider"
	}
	return provider
}

func streamDropRetryClassification(err error) llm.ProviderErrorClassification {
	return kernelstreamdiag.RetryClassification(err)
}

func streamDropErrorType(err error) string {
	return kernelstreamdiag.ErrorType(err)
}

func streamDropConcreteErrorType(err error) string {
	return kernelstreamdiag.ConcreteErrorType(err)
}

func streamDropErrorChain(err error) string {
	return kernelstreamdiag.ErrorChain(err)
}

func formatStreamDiagnosticHeaders(headers map[string]string) string {
	return kernelstreamdiag.FormatHeaders(headers)
}

func compactStreamDropText(text string, maxLen int) string {
	return kernelstreamdiag.CompactText(text, maxLen)
}

func streamDropErrorText(err error) string {
	return kernelstreamdiag.ErrorText(err)
}

// streamInner runs one stream attempt. Pumps events from llm.Stream.Recv
// into a bounded channel, multiplexes over the kernel's platform events and
// a 16ms flush ticker, and returns a classified outcome so the retry-loop
// caller knows what to do next.
//
// The outer ctx (from runTurn) is used for ambient cancellation checks.
// The runCtx (per-attempt) is what the pump goroutine uses for Recv; when
// this stream ends (normal, cancel, or retryable error), runCtx is cancelled
// by the caller.
func (k *Kernel) streamInner(
	ctx, runCtx context.Context,
	cancelRun context.CancelFunc,
	stream llm.Stream,
	finalDelta *llm.Event,
	gotFinal *bool,
	fatalErr *error,
	cancelled *bool,
	replaceOnNextToken *bool,
	retryErr *error,
	retryDiag *llm.StreamDiagnostics,
) streamOutcome {
	type streamResult struct {
		event llm.Event
		err   error
	}
	deltaCh := make(chan streamResult, 8)
	go func() {
		defer close(deltaCh)
		for {
			ev, err := stream.Recv(runCtx)
			select {
			case deltaCh <- streamResult{event: ev, err: err}:
			case <-runCtx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(FlushInterval)
	defer ticker.Stop()

	var (
		dirty   bool
		outcome streamOutcome
	)
	outcome = streamOutcomeFatal // default if something truly unexpected happens
	setRetryErr := func(err error) {
		if retryErr != nil && err != nil {
			*retryErr = err
		}
		if retryDiag != nil && err != nil {
			*retryDiag = llm.StreamDiagnosticsOf(stream)
		}
	}

streamLoop:
	for {
		select {
		case <-ctx.Done():
			*cancelled = true
			cancelRun()
			outcome = streamOutcomeCancelled
			break streamLoop

		case e := <-k.events:
			switch e.Kind {
			case PlatformEventCancel:
				*cancelled = true
				cancelRun()
				outcome = streamOutcomeCancelled
				break streamLoop
			case PlatformEventSubmit:
				k.lastError = ErrTurnInFlight.Error()
				k.emitFrame("still processing previous turn")
			case PlatformEventResetSession:
				// Zero-Leak Invariant: never truncate an active turn. Reject
				// the reset without mutating state; the caller must /stop
				// first if they want to abandon this stream.
				if e.ack != nil {
					e.ack <- ErrResetDuringTurn
				}
			case PlatformEventSetModel:
				// Zero-Leak Invariant: never disturb an active turn. Reject
				// the switch without mutating the resident override.
				if e.ack != nil {
					e.ack <- ErrSetModelDuringTurn
				}
			case PlatformEventManualCompress:
				if e.ack != nil {
					e.ack <- ErrCompressDuringTurn
				}
			case PlatformEventQuit:
				*cancelled = true
				cancelRun()
				outcome = streamOutcomeCancelled
				break streamLoop
			case PlatformEventSteer:
				k.queueSteerGuidance(e.Text)
			}

		case res, ok := <-deltaCh:
			if !ok {
				// Pump exited on its own — treat as retryable (unexpected EOF).
				// Only treat as Done if EventDone was already consumed (*gotFinal).
				if *gotFinal {
					outcome = streamOutcomeDone
				} else {
					setRetryErr(io.ErrUnexpectedEOF)
					outcome = streamOutcomeRetryable
				}
				break streamLoop
			}
			if res.err != nil {
				if res.err == io.EOF {
					if *gotFinal {
						outcome = streamOutcomeDone
					} else {
						// Stream ended without EventDone — treat as retryable.
						setRetryErr(io.ErrUnexpectedEOF)
						outcome = streamOutcomeRetryable
					}
					break streamLoop
				}
				if runCtx.Err() != nil {
					*cancelled = true
					outcome = streamOutcomeCancelled
					break streamLoop
				}
				// Classify the error: Retryable → caller retries; otherwise fatal.
				if llm.Classify(res.err) == llm.ClassRetryable {
					setRetryErr(res.err)
					outcome = streamOutcomeRetryable
				} else {
					*fatalErr = res.err
					outcome = streamOutcomeFatal
				}
				break streamLoop
			}
			ev := res.event
			switch ev.Kind {
			case llm.EventToken:
				if *replaceOnNextToken {
					k.draft = ""
					*replaceOnNextToken = false
				}
				k.draft += ev.Token
				k.tm.Tick(ev.TokensOut)
				dirty = true
			case llm.EventReasoning:
				if *replaceOnNextToken {
					// Reasoning doesn't count as visible content; the NEXT EventToken
					// still clears the draft. Do NOT flip replaceOnNextToken here.
				}
				k.draftReasoning += ev.Reasoning
				k.addSoul("reasoning: " + truncate(ev.Reasoning, 60))
				dirty = true
			case llm.EventDone:
				*finalDelta = ev
				*gotFinal = true
				outcome = streamOutcomeDone
				break streamLoop
			}

		case <-ticker.C:
			if dirty {
				k.emitFrame("streaming")
				dirty = false
			}
		}
	}

	// Drain deltaCh so the pump goroutine exits before we return.
	cancelRun()
	for range deltaCh {
	}
	return outcome
}

// addSoul appends a Soul Monitor entry with a ring-buffer cap.
func (k *Kernel) addSoul(text string) {
	k.soul = append(k.soul, SoulEntry{At: time.Now(), Text: text})
	if len(k.soul) > SoulBufferSize {
		k.soul = k.soul[len(k.soul)-SoulBufferSize:]
	}
}

// emitFrame builds a RenderFrame snapshot and publishes it to the render
// mailbox with replace-latest semantics: if an unread frame already sits
// in the capacity-1 buffer, drain it and drop it before enqueueing the new
// one. This is what keeps a slow TUI from backpressuring the kernel.
func (k *Kernel) emitFrame(status string) {
	var contextStatus *llm.ContextStatus
	if k.cfg.ContextEngine != nil {
		snapshot := k.cfg.ContextEngine.Status()
		contextStatus = &snapshot
	}
	providerStatus := llm.ProviderStatusOf(k.client)
	frame := RenderFrame{
		Seq:             k.seq.Add(1),
		Phase:           k.phase,
		DraftText:       k.draft,
		History:         append([]llm.Message(nil), k.history...),
		Telemetry:       k.tm.Snapshot(),
		StatusText:      status,
		SessionID:       k.sessionID,
		Model:           k.displayModel(),
		ReasoningEffort: k.displayReasoningEffort(providerStatus),
		ProviderStatus:  providerStatus,
		RetryStatus:     k.retryStatus.Snapshot(),
		LastError:       k.lastError,
		SoulEvents:      append([]SoulEntry(nil), k.soul...),
		ContextStatus:   contextStatus,
	}
	// Drain old frame if present, then enqueue new.
	select {
	case <-k.render:
	default:
	}
	select {
	case k.render <- frame:
	default:
		// Should be unreachable after the drain above.
	}
	// Deliver the same frame to every independent subscription.
	k.fanOutFrame(frame)
}

// residentModel is the kernel's between-turns model: the resident in-session
// override set by PlatformEventSetModel when present, otherwise the configured
// default. It is the "resident" argument to selectTurnModel, so a per-event
// PlatformEvent.Model still wins over it. Run-goroutine-owned.
func (k *Kernel) residentModel() string {
	if m := strings.TrimSpace(k.sessionModel); m != "" {
		return m
	}
	return k.cfg.Model
}

func (k *Kernel) displayModel() string {
	if strings.TrimSpace(k.activeModel) != "" {
		return k.activeModel
	}
	if m := strings.TrimSpace(k.sessionModel); m != "" {
		return m
	}
	return k.cfg.Model
}

func (k *Kernel) displayReasoningEffort(status llm.ProviderStatus) llm.ReasoningEffortEvidence {
	if k.activeReasoning.State != "" {
		return k.activeReasoning
	}
	return llm.ResolveReasoningEffort(k.cfg.ReasoningEffort, llm.ReasoningEffortSourceConfigDefault, status)
}

func selectTurnModel(residentModel, override string) string {
	return kernelmodelrouting.SelectTurnModel(residentModel, override)
}

func selectTurnReasoningEffort(residentEffort, override string, status llm.ProviderStatus) llm.ReasoningEffortEvidence {
	return kernelmodelrouting.SelectTurnReasoningEffort(residentEffort, override, status)
}

// cronFlag returns 1 when the turn carries a cron_job_id (Phase 2.D),
// 0 otherwise. Keeps json.Marshal output consistent: cron is always
// present as an integer (even for non-cron turns, where it's 0). The
// memory worker's payload decoder defaults cron=0 when the field is
// absent, so either encoding works — explicit is less surprising.
func cronFlag(cronJobID string) int {
	if cronJobID == "" {
		return 0
	}
	return 1
}

func (k *Kernel) skipMemorySync(turnKey, reason string) {
	skipper, ok := k.store.(memorySyncSkipper)
	if !ok || turnKey == "" {
		return
	}
	skipCtx, cancel := context.WithTimeout(context.Background(), StoreAckDeadline)
	defer cancel()
	if err := skipper.SkipMemorySync(skipCtx, turnKey, reason); err != nil {
		k.log.Warn("kernel: skip memory sync failed", "err", err)
	}
}

func platformFromChatKey(chatKey string) string {
	if chatKey == "" {
		return ""
	}
	idx := strings.IndexByte(chatKey, ':')
	if idx < 0 {
		return ""
	}
	return chatKey[:idx]
}

// truncate returns s clamped to n runes with an ellipsis suffix. Safe on
// non-ASCII input.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
