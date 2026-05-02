package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

var startGreeting = gatewayHelpText()

const shutdownNotice = "Gateway is shutting down — send /stop to cancel the active turn or try again shortly."
const followUpQueueFullNotice = "Busy — follow-up queue is full; try again after the current turn."
const followUpQueueCap = kernel.PlatformEventMailboxCap
const defaultInboundDedupMaxSize = 4096

type DrainTimeoutReason string

const (
	DrainReasonRestartTimeout  DrainTimeoutReason = session.ResumeReasonRestartTimeout
	DrainReasonShutdownTimeout DrainTimeoutReason = session.ResumeReasonShutdownTimeout
)

type sessionMetadataReader interface {
	GetMetadata(context.Context, string) (session.Metadata, bool, error)
}

type sessionMetadataWriter interface {
	PutMetadata(context.Context, session.Metadata) error
}

type sessionResumeClearer interface {
	ClearResumePending(context.Context, string) (bool, error)
}

type activeTurnSnapshot struct {
	Platform     string
	ChatID       string
	MsgID        string
	SessionKey   string
	SessionID    string
	Source       SessionSource
	Cancelled    bool
	LastUserText string
}

// ManagerConfig drives the shared gateway manager.
type ManagerConfig struct {
	AllowedChats    map[string]string
	AllowDiscovery  map[string]bool
	CoalesceMs      int
	FreshFinalAfter time.Duration
	// ToolProgressMode mirrors Hermes gateway display.tool_progress for
	// editable channel progress messages. Empty and unknown values default to all.
	ToolProgressMode string
	// ToolProgressCommandEnabled gates Hermes' /verbose command on messaging
	// platforms. Hermes defaults this gate off.
	ToolProgressCommandEnabled bool
	BusyInputMode              string // interrupt, queue, or steer
	// PersistToolProgressMode saves /verbose mode changes. Production writes
	// display.platforms.<platform>.tool_progress into config.yaml.
	PersistToolProgressMode func(platform, mode string) error
	// ToolProgressModes mirrors Hermes display.platforms.<platform>.tool_progress
	// overrides. Values take precedence over ToolProgressMode for the named platform.
	ToolProgressModes map[string]string
	SessionMap        session.Map
	// AgentRouting enables OpenClaw-style agent/workspace bindings for live
	// gateway turns. Zero value preserves legacy single-agent chat keys.
	AgentRouting AgentRoutingConfig
	// ToolRegistry is the full process registry. Agent-routed turns receive a
	// policy-filtered view of this registry when agents.list[].tools is set.
	ToolRegistry *tools.Registry
	// SkillRuntime is the full process skills runtime. Agent-routed turns
	// receive an allowlist wrapper when agents.list[].skills is set.
	SkillRuntime *skills.Runtime
	// AgentRuntimeFactory optionally returns an independent kernel/runtime for
	// the routed agent session. When nil, Manager falls back to the legacy
	// singleton kernel with per-turn policy overrides.
	AgentRuntimeFactory AgentRuntimeFactory
	Hooks               *Hooks
	RuntimeStatus       RuntimeStatusWriter
	Restart             RestartConfig
	SessionExpiry       SessionExpiryConfig
	Now                 func() time.Time
	// PersistReasoningGlobal is invoked by /reasoning ... --global to persist
	// the requested effort beyond the calling session. A nil callback or one
	// that returns an error causes the command to fall back to a session-only
	// override and surface PersistFailed=true to the caller.
	PersistReasoningGlobal func(ReasoningEffort) error
	// AccountUsage renders /usage provider account-limit evidence. Runtime token
	// telemetry is read from the manager's latest render frame.
	AccountUsage AccountUsageProvider
	// ContextFilesProfile overrides the profile directory used for live-turn
	// SOUL.md discovery. Empty falls back to config.GormesHome() at call time.
	// Tests inject hermetic temp directories so no live ~/.gormes state is read.
	ContextFilesProfile string
	// ContextFilesCWD overrides the working directory used for project-context
	// discovery (HERMES.md / .hermes.md / AGENTS.md / CLAUDE.md / .cursorrules).
	// Empty falls back to os.Getwd() at call time.
	ContextFilesCWD string
	// ContextFilesMemoryDir overrides the memory directory used for live-turn
	// USER.md and MEMORY.md durable user-context discovery. Empty falls back
	// to <config.GormesHome()>/memory at call time. Tests inject hermetic
	// temp directories so no live ~/.gormes/memory state is read.
	ContextFilesMemoryDir string
	// LiveTurnNow overrides the clock used to render the live-turn timestamp
	// line. Nil leaves the clock unset, which suppresses the "Conversation
	// started: ..." line so slice-1+2 byte output stays stable. Production
	// wiring should set this to time.Now (or an equivalent UTC/zone-aware
	// closure); tests wire a fixed clock.
	LiveTurnNow func() time.Time
	// LiveTurnActiveModel returns the active model name rendered on the
	// `Model: ...` line. Nil or empty result drops the line.
	LiveTurnActiveModel func() string
	// LiveTurnActiveProvider returns the active provider name rendered on
	// the `Provider: ...` line. Nil or empty result drops the line. Only
	// non-secret display names are exposed — never base URLs or API keys.
	LiveTurnActiveProvider func() string
	// TitleModel is the provider boundary for auto-title generation. It is
	// called at most once per PhaseIdle frame for sessions without an existing
	// title. Nil disables the LLM call; PerformAutoTitle surfaces
	// AutoTitleCodeProviderFailed evidence through AuxiliaryFailureSink.
	TitleModel hermes.TitleModelFunc
	// TitleStore is the persistence boundary for auto-title generation. When
	// non-nil it is used directly; when nil and SessionMap is non-nil the
	// production wiring constructs a MetadataTitleStore at startup and injects
	// it here. Tests always inject a hermetic store via this field.
	TitleStore session.SessionTitleStore
	// AuxiliaryFailureSink receives AutoTitleEvidence for non-complete outcomes
	// (provider failures, blank results, store errors, skip evidence). Nil
	// discards non-complete evidence silently. The sink must not block or panic;
	// panics are recovered and discarded.
	AuxiliaryFailureSink AutoTitleAuxiliarySink
	// CoalescerEvidenceSink receives CoalescerEvidence for non-happy-path
	// finalize outcomes (edit_failed_fallback, send_final_failed). Nil
	// discards evidence silently; production behavior is unchanged. The sink
	// must not block or panic.
	CoalescerEvidenceSink CoalescerEvidenceSink
	// RememberedSourceStore records allowed inbound channel origins in a small
	// channel-directory source ledger. Nil disables the hook. Failures are
	// degraded and must not block user turns.
	RememberedSourceStore RememberedSourceStore
	// TypingActionEvidenceSink receives redacted non-fatal typing-action
	// failures. Nil discards evidence silently.
	TypingActionEvidenceSink TypingActionEvidenceSink
}

type KernelSubmitter interface {
	Submit(ev kernel.PlatformEvent) error
	ResetSession() error
	Render() <-chan kernel.RenderFrame
}

type kernelSubmitter = KernelSubmitter

type AgentRuntimeFactory func(context.Context, AgentRuntimeRequest) (KernelSubmitter, error)

type AgentRuntimeRequest struct {
	AgentID     string
	Name        string
	SessionKey  string
	Workspace   string
	AgentDir    string
	AuthHome    string
	AuthStore   string
	Model       string
	BindingTier string
	ToolPolicy  config.AgentToolPolicy
	SkillNames  []string
	Tools       *tools.Registry
	Skills      kernel.SkillProvider
	ToolSafety  kernel.ToolSafetyPolicy
}

// Manager owns cross-channel gateway mechanics for one binary instance.
type Manager struct {
	cfg    ManagerConfig
	kernel kernelSubmitter
	log    *slog.Logger

	mu       sync.Mutex
	channels map[string]Channel

	turnMu           sync.Mutex
	turnPlatform     string
	turnChatID       string
	turnMsgID        string
	turnSessionKey   string
	turnSessionID    string
	turnSource       SessionSource
	turnCancelled    bool
	turnFrameSeen    bool
	turnLastUserText string // captures the last inbound submit text for auto-title
	turnKernel       KernelSubmitter
	kernelSessionKey string
	shuttingDown     bool
	followUps        []InboundEvent
	lastUsageFrame   kernel.RenderFrame

	reasoningMu    sync.Mutex
	reasoningState map[string]SessionReasoningState
	ttsConfigs     map[string]TTSConfig

	inboundDedup *MessageDeduplicator

	renderChan <-chan kernel.RenderFrame

	typingStop func()
	typingKey  string

	liveTurnPromptSeams liveTurnPromptSeams
	agentRouter         AgentRouter
	agentRoutingEnabled bool
	agentRuntimeMu      sync.Mutex
	agentRuntimes       map[string]KernelSubmitter
	agentRuntimeRender  chan kernel.RenderFrame

	typingActionMu   sync.Mutex
	typingActionLast map[string]time.Time

	toolProgressMu     sync.Mutex
	toolProgressMsgID  string
	toolProgressText   string
	toolProgressChatID string
	toolProgressPlat   string

	verboseHintMu   sync.Mutex
	verboseHintSent map[string]bool
}

type channelRunFailure struct {
	channel Channel
	err     error
}

type hookedPlaceholderEditor struct {
	base         placeholderEditor
	manager      *Manager
	platform     string
	replyToMsgID string
}

func (h hookedPlaceholderEditor) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	const placeholderText = "⏳"

	h.manager.fireHook(ctx, HookEvent{
		Point:    HookBeforeSend,
		Platform: h.platform,
		ChatID:   chatID,
		Text:     placeholderText,
	})

	var (
		msgID string
		err   error
	)
	if h.replyToMsgID != "" {
		if replySender, ok := h.base.(ReplyPlaceholderCapable); ok {
			msgID, err = replySender.SendReplyPlaceholder(ctx, chatID, h.replyToMsgID)
		} else {
			msgID, err = h.base.SendPlaceholder(ctx, chatID)
		}
	} else {
		msgID, err = h.base.SendPlaceholder(ctx, chatID)
	}
	if err != nil {
		h.manager.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
			Platform:      h.platform,
			PlatformState: PlatformStateFailed,
			ErrorMessage:  err.Error(),
		})
		h.manager.fireHook(ctx, HookEvent{
			Point:    HookOnError,
			Platform: h.platform,
			ChatID:   chatID,
			Text:     placeholderText,
			Err:      err,
		})
		return "", err
	}

	h.manager.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      h.platform,
		PlatformState: PlatformStateRunning,
	})
	h.manager.fireHook(ctx, HookEvent{
		Point:    HookAfterSend,
		Platform: h.platform,
		ChatID:   chatID,
		MsgID:    msgID,
		Text:     placeholderText,
	})
	return msgID, nil
}

func (h hookedPlaceholderEditor) Send(ctx context.Context, chatID, text string) (string, error) {
	sender, ok := h.base.(coalescerMessageSender)
	if !ok {
		return "", errors.New("gateway: channel does not support fresh final send")
	}

	h.manager.fireHook(ctx, HookEvent{
		Point:    HookBeforeSend,
		Platform: h.platform,
		ChatID:   chatID,
		Text:     text,
	})

	var (
		msgID string
		err   error
	)
	if h.replyToMsgID != "" {
		if replySender, ok := h.base.(ReplySender); ok {
			msgID, err = replySender.SendReply(ctx, chatID, h.replyToMsgID, text)
		} else {
			msgID, err = sender.Send(ctx, chatID, text)
		}
	} else {
		msgID, err = sender.Send(ctx, chatID, text)
	}
	if err != nil {
		h.manager.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
			Platform:      h.platform,
			PlatformState: PlatformStateFailed,
			ErrorMessage:  err.Error(),
		})
		h.manager.fireHook(ctx, HookEvent{
			Point:    HookOnError,
			Platform: h.platform,
			ChatID:   chatID,
			Text:     text,
			Err:      err,
		})
		return "", err
	}

	h.manager.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      h.platform,
		PlatformState: PlatformStateRunning,
	})
	h.manager.fireHook(ctx, HookEvent{
		Point:    HookAfterSend,
		Platform: h.platform,
		ChatID:   chatID,
		MsgID:    msgID,
		Text:     text,
	})
	return msgID, nil
}

func (h hookedPlaceholderEditor) EditMessage(ctx context.Context, chatID, msgID, text string) error {
	return h.base.EditMessage(ctx, chatID, msgID, text)
}

func (h hookedPlaceholderEditor) EditMessageFinal(ctx context.Context, chatID, msgID, text string, finalize bool) error {
	if finalizer, ok := h.base.(FinalizingMessageEditor); ok {
		return finalizer.EditMessageFinal(ctx, chatID, msgID, text, finalize)
	}
	return h.base.EditMessage(ctx, chatID, msgID, text)
}

func (h hookedPlaceholderEditor) DeleteMessage(ctx context.Context, chatID, msgID string) error {
	if deleter, ok := h.base.(MessageDeleter); ok {
		return deleter.DeleteMessage(ctx, chatID, msgID)
	}
	return nil
}

// ErrDuplicateChannel is returned when two registered channels share a name.
var ErrDuplicateChannel = errors.New("gateway: duplicate channel name")

// ErrEmptyChannelName is returned when a channel reports an empty Name.
var ErrEmptyChannelName = errors.New("gateway: channel Name() must be non-empty")

// NewManager constructs a manager backed by a concrete kernel.
func NewManager(cfg ManagerConfig, k *kernel.Kernel, log *slog.Logger) *Manager {
	return newManagerInternal(cfg, k, log)
}

// NewManagerWithSubmitter lets tests inject a fake kernel-compatible object.
func NewManagerWithSubmitter(cfg ManagerConfig, k kernelSubmitter, log *slog.Logger) *Manager {
	return newManagerInternal(cfg, k, log)
}

func newManagerInternal(cfg ManagerConfig, k kernelSubmitter, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	if cfg.CoalesceMs <= 0 {
		cfg.CoalesceMs = 1000
	}
	if cfg.AllowedChats == nil {
		cfg.AllowedChats = map[string]string{}
	}
	if cfg.BusyInputMode == "" {
		cfg.BusyInputMode = "interrupt"
	}
	if cfg.AllowDiscovery == nil {
		cfg.AllowDiscovery = map[string]bool{}
	}
	seams := defaultLiveTurnPromptSeams()
	if dir := strings.TrimSpace(cfg.ContextFilesProfile); dir != "" {
		seams.ProfileDir = func() string { return dir }
	}
	explicitContextFixture := strings.TrimSpace(cfg.ContextFilesProfile) != "" || strings.TrimSpace(cfg.ContextFilesCWD) != ""
	if cwd := strings.TrimSpace(cfg.ContextFilesCWD); cwd != "" {
		seams.CWD = func() string { return cwd }
	}
	if memDir := strings.TrimSpace(cfg.ContextFilesMemoryDir); memDir != "" {
		seams.MemoryDir = func() string { return memDir }
	} else if explicitContextFixture {
		// Tests and callers that provide explicit profile/CWD fixtures without a
		// memory fixture expect durable USER.md/MEMORY.md context to be absent.
		// Production gateway wiring leaves these fields empty and uses the default
		// workspace/profile discovery above.
		seams.MemoryDir = func() string { return "" }
	}
	if cfg.LiveTurnNow != nil {
		seams.Now = cfg.LiveTurnNow
	}
	if cfg.LiveTurnActiveModel != nil {
		seams.ActiveModel = cfg.LiveTurnActiveModel
	}
	if cfg.LiveTurnActiveProvider != nil {
		seams.ActiveProvider = cfg.LiveTurnActiveProvider
	}
	return &Manager{
		cfg:                 cfg,
		kernel:              k,
		log:                 log,
		channels:            map[string]Channel{},
		reasoningState:      map[string]SessionReasoningState{},
		inboundDedup:        NewMessageDeduplicator(defaultInboundDedupMaxSize),
		liveTurnPromptSeams: seams,
		agentRouter:         NewAgentRouter(cfg.AgentRouting.Agents, cfg.AgentRouting.Bindings),
		agentRoutingEnabled: cfg.AgentRouting.Enabled,
		agentRuntimes:       map[string]KernelSubmitter{},
		agentRuntimeRender:  make(chan kernel.RenderFrame, kernel.RenderMailboxCap),
		typingActionLast:    map[string]time.Time{},
	}
}

// DispatchReasoning routes a /reasoning invocation through ParseReasoningCommand
// and ApplyReasoningCommand for the calling session only. The session is
// identified by sessionKey (typically "<platform>:<chat_id>"). Each session
// keeps its own SessionReasoningState — mutations are isolated per key. The
// persistGlobal callback configured on ManagerConfig is invoked only when
// /reasoning <effort> --global succeeds; otherwise the call is session-only
// and the resulting reply.PersistFailed surfaces global-save errors without
// changing other sessions' state.
func (m *Manager) DispatchReasoning(sessionKey string, args []string) (ReasoningReply, error) {
	cmd, err := ParseReasoningCommand(args)
	if err != nil {
		return ReasoningReply{}, err
	}
	persist := m.cfg.PersistReasoningGlobal
	if persist == nil {
		persist = func(ReasoningEffort) error {
			return errors.New("gateway: PersistReasoningGlobal not configured")
		}
	}

	m.reasoningMu.Lock()
	defer m.reasoningMu.Unlock()
	state, ok := m.reasoningState[sessionKey]
	if !ok {
		state = SessionReasoningState{Source: ReasoningSourceUnset}
	}
	newState, reply := ApplyReasoningCommand(state, cmd, persist)
	m.reasoningState[sessionKey] = newState
	return reply, nil
}

func (m *Manager) now() time.Time {
	if m.cfg.Now != nil {
		return m.cfg.Now().UTC()
	}
	return time.Now().UTC()
}

// Register adds a channel to the manager. It must be called before Run.
func (m *Manager) Register(ch Channel) error {
	name := ch.Name()
	if name == "" {
		return ErrEmptyChannelName
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.channels[name]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicateChannel, name)
	}
	m.channels[name] = ch
	return nil
}

// ChannelCount reports how many channels are currently registered.
func (m *Manager) ChannelCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.channels)
}

// Shutdown prevents new work from starting and waits for the currently active
// turn, if any, to drain before returning or timing out.
func (m *Manager) Shutdown(ctx context.Context) error {
	return m.ShutdownWithDrainReason(ctx, DrainReasonShutdownTimeout)
}

func (m *Manager) ShutdownWithDrainReason(ctx context.Context, reason DrainTimeoutReason) error {
	if reason == "" {
		reason = DrainReasonShutdownTimeout
	}
	m.turnMu.Lock()
	m.shuttingDown = true
	m.turnMu.Unlock()

	activeAgents := m.activeAgentCount()
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		GatewayState: GatewayStateDraining,
		ActiveAgents: &activeAgents,
	})

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if !m.hasActiveTurn() {
			return nil
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				m.markDrainTimeoutResumePending(context.Background(), reason)
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (m *Manager) setRenderChan(c <-chan kernel.RenderFrame) {
	m.renderChan = c
}

func (m *Manager) Run(ctx context.Context) error {
	m.mu.Lock()
	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.Unlock()

	restartRequested := false
	zeroActiveAgents := 0
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		GatewayState:     GatewayStateStarting,
		RestartRequested: &restartRequested,
		ActiveAgents:     &zeroActiveAgents,
	})

	inbox := make(chan InboundEvent, len(channels)*4)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	failures := make(chan channelRunFailure, len(channels))

	var wg sync.WaitGroup
	for _, ch := range channels {
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
			Platform:      ch.Name(),
			PlatformState: PlatformStateStarting,
		})
		wg.Add(1)
		go func(c Channel) {
			defer wg.Done()
			m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
				Platform:      c.Name(),
				PlatformState: PlatformStateRunning,
			})
			if err := c.Run(runCtx, inbox); err != nil && !errors.Is(err, context.Canceled) {
				m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
					Platform:      c.Name(),
					PlatformState: PlatformStateFailed,
					ErrorMessage:  err.Error(),
				})
				m.fireHook(runCtx, HookEvent{
					Point:    HookOnError,
					Platform: c.Name(),
					Err:      err,
				})
				m.log.Warn("channel exited with error", "channel", c.Name(), "err", err)
				failures <- channelRunFailure{channel: c, err: err}
				return
			}
			m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
				Platform:      c.Name(),
				PlatformState: PlatformStateStopped,
			})
		}(ch)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		m.runOutbound(runCtx)
	}()

	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		GatewayState:     GatewayStateRunning,
		RestartRequested: &restartRequested,
		ActiveAgents:     &zeroActiveAgents,
	})
	if err := m.ConsumeRestartTakeoverMarker(runCtx); err != nil {
		m.log.Debug("consume restart takeover marker", "err", err)
	}

	activeChannels := len(channels)
	var firstFailure error
	for {
		select {
		case <-ctx.Done():
			cancel()
			wg.Wait()
			zero := 0
			m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
				GatewayState: GatewayStateStopped,
				ActiveAgents: &zero,
			})
			return nil
		case failure := <-failures:
			m.safeChannelDisconnect(ctx, failure.channel)
			if firstFailure == nil {
				firstFailure = failure.err
			}
			activeChannels--
			if activeChannels <= 0 {
				cancel()
				wg.Wait()
				reason := ""
				if firstFailure != nil {
					reason = firstFailure.Error()
				}
				zero := 0
				m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
					GatewayState: GatewayStateStartupFailed,
					ExitReason:   reason,
					ActiveAgents: &zero,
				})
				return firstFailure
			}
		case ev := <-inbox:
			if err := m.handleInbound(runCtx, ev); err != nil {
				cancel()
				wg.Wait()
				return err
			}
		}
	}
}

func (m *Manager) safeChannelDisconnect(ctx context.Context, ch Channel) {
	if disconnecter, ok := ch.(DisconnectCapable); ok {
		if err := disconnecter.Disconnect(ctx); err != nil {
			m.log.Debug("defensive channel disconnect after failed startup raised", "channel", ch.Name(), "err", err)
		}
	}
}

func (m *Manager) runOutbound(ctx context.Context) {
	frames := m.renderChan
	if frames == nil && m.cfg.AgentRuntimeFactory != nil {
		frames = m.agentRuntimeRender
	}
	if frames == nil && m.kernel != nil {
		frames = m.kernel.Render()
	}
	if frames == nil {
		<-ctx.Done()
		return
	}

	var (
		co       *coalescer
		coCancel context.CancelFunc
	)

	for {
		select {
		case <-ctx.Done():
			if coCancel != nil {
				coCancel()
			}
			return
		case f, ok := <-frames:
			if !ok {
				if coCancel != nil {
					coCancel()
				}
				return
			}
			m.persistSession(ctx, f)
			m.dispatchFrame(ctx, f, &co, &coCancel)
		}
	}
}

func (m *Manager) handleInbound(ctx context.Context, ev InboundEvent) error {
	m.fireHook(ctx, HookEvent{
		Point:    HookBeforeReceive,
		Platform: ev.Platform,
		ChatID:   ev.ChatID,
		MsgID:    ev.MsgID,
		Kind:     ev.Kind,
		Text:     ev.Text,
		Inbound:  &ev,
	})
	defer m.fireHook(ctx, HookEvent{
		Point:    HookAfterReceive,
		Platform: ev.Platform,
		ChatID:   ev.ChatID,
		MsgID:    ev.MsgID,
		Kind:     ev.Kind,
		Text:     ev.Text,
		Inbound:  &ev,
	})

	if !m.allowed(ev) {
		if m.cfg.AllowDiscovery[ev.Platform] {
			m.log.Info("first-run discovery: unknown chat", "platform", ev.Platform, "chat_id", ev.ChatID)
		} else {
			m.log.Warn("unauthorised chat blocked", "platform", ev.Platform, "chat_id", ev.ChatID)
		}
		return nil
	}

	ch := m.lookupChannel(ev.Platform)
	if ch == nil {
		m.log.Warn("inbound for unknown channel", "platform", ev.Platform)
		return nil
	}
	if m.isShuttingDown() && ev.Kind != EventCancel {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, shutdownNotice)
		return nil
	}

	switch ev.Kind {
	case EventStart:
		if _, err := m.sendWithHooks(ctx, ch, ev.ChatID, startGreeting); err != nil {
			m.log.Warn("send greeting", "platform", ev.Platform, "chat_id", ev.ChatID, "err", err)
		}
		return nil
	case EventCancel:
		m.markTurnCancelled()
		if k := m.activeTurnKernel(); k != nil {
			_ = k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventCancel})
		}
		return nil
	case EventReset:
		if m.kernel == nil {
			return nil
		}
		if err := m.kernel.ResetSession(); err != nil {
			if errors.Is(err, kernel.ErrResetDuringTurn) {
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Cannot reset during active turn — send /stop first.")
			} else {
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Session reset failed: "+err.Error())
			}
			return nil
		}
		if m.cfg.SessionMap != nil {
			key := m.sessionKeyForInbound(ev)
			if err := m.cfg.SessionMap.Put(ctx, key, ""); err != nil {
				m.log.Warn("clear session mapping", "key", key, "err", err)
			}
		}
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Session reset. Next message starts fresh.")
		return nil
	case EventRestart:
		return m.handleRestartCommand(ctx, ch, ev)
	case EventSteer:
		m.handleSteerCommand(ctx, ch, ev)
		return nil
	case EventUsage:
		m.handleUsageCommand(ctx, ch, ev)
		return nil
	case EventStatus:
		m.handleStatusCommand(ctx, ch, ev)
		return nil
	case EventTitle:
		m.handleTitleCommand(ctx, ch, ev)
		return nil
	case EventSkills:
		m.handleSkillsCommand(ctx, ch, ev)
		return nil
	case EventVerbose:
		m.handleVerboseCommand(ctx, ch, ev)
		return nil
	case EventModel:
		m.handleModelCommand(ctx, ch, ev)
		return nil
	case EventSessions:
		m.handleSessionsCommand(ctx, ch, ev)
		return nil
	case EventProfile:
		m.handleProfileCommand(ctx, ch, ev)
		return nil
	case EventGateway:
		m.handlePlatformsCommand(ctx, ch, ev)
		return nil
	case EventReasoning:
		m.handleReasoningCommand(ctx, ch, ev)
		return nil
	case EventBusy:
		m.handleBusyCommand(ctx, ch, ev)
		return nil
	case EventTTS:
		m.handleTTSCommand(ctx, ch, ev)
		return nil
	case EventSubmit:
		if m.handleSlashSubmitCommand(ctx, ch, ev) {
			return nil
		}
		if m.kernel == nil && m.cfg.AgentRuntimeFactory == nil {
			return nil
		}
		if m.dropDuplicateInboundSubmit(ev) {
			return nil
		}
		queued, full := m.queueFollowUpIfActive(ev)
		if queued {
			return nil
		}
		if full {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, followUpQueueFullNotice)
			return nil
		}
		m.pinTurn(ev.Platform, ev.ChatID, ev.MsgID)
		m.submitPinned(ctx, ch, ev)
		return nil
	case EventUnknown:
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "unknown command")
		return nil
	}
	return nil
}

func (m *Manager) handleSlashSubmitCommand(ctx context.Context, ch Channel, ev InboundEvent) bool {
	body := strings.TrimSpace(ev.Text)
	if !strings.HasPrefix(body, "/") {
		return false
	}

	cmd, ok := ResolveCommand(body)
	if !ok {
		name := slashCommandName(body)
		if isRecognizedUnavailableSlashCommand(name) {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/"+name+" is recognized but unavailable in this build")
		} else {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "unknown command — no slash command by that name is available")
		}
		return true
	}
	if m.hasActiveTurn() && cmd.ActiveTurnPolicy == CommandActiveTurnPolicyReject {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Gormes is busy — finish the current turn or send /stop before /"+cmd.Name)
		return true
	}
	if cmd.ActiveTurnPolicy == CommandActiveTurnPolicyUnavailable {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/"+cmd.Name+" is recognized but unavailable in this build")
		return true
	}
	commandEvent := ev
	commandEvent.Kind = cmd.Kind
	if cmd.Kind == EventSteer || cmd.Kind == EventTitle || cmd.Kind == EventReasoning {
		commandEvent.Text = body
	} else {
		commandEvent.Text = ""
	}
	return m.dispatchCommandEvent(ctx, ch, commandEvent)
}

func (m *Manager) dispatchCommandEvent(ctx context.Context, ch Channel, ev InboundEvent) bool {
	switch ev.Kind {
	case EventStart:
		if _, err := m.sendWithHooks(ctx, ch, ev.ChatID, startGreeting); err != nil {
			m.log.Warn("send greeting", "platform", ev.Platform, "chat_id", ev.ChatID, "err", err)
		}
		return true
	case EventCancel:
		m.markTurnCancelled()
		if k := m.activeTurnKernel(); k != nil {
			_ = k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventCancel})
		}
		return true
	case EventReset:
		if m.kernel == nil {
			return true
		}
		if err := m.kernel.ResetSession(); err != nil {
			if errors.Is(err, kernel.ErrResetDuringTurn) {
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Cannot reset during active turn — send /stop first.")
			} else {
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Session reset failed: "+err.Error())
			}
			return true
		}
		if m.cfg.SessionMap != nil {
			key := m.sessionKeyForInbound(ev)
			if err := m.cfg.SessionMap.Put(ctx, key, ""); err != nil {
				m.log.Warn("clear session mapping", "key", key, "err", err)
			}
		}
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Session reset. Next message starts fresh.")
		return true
	case EventRestart:
		_ = m.handleRestartCommand(ctx, ch, ev)
		return true
	case EventSteer:
		m.handleSteerCommand(ctx, ch, ev)
		return true
	case EventUsage:
		m.handleUsageCommand(ctx, ch, ev)
		return true
	case EventStatus:
		m.handleStatusCommand(ctx, ch, ev)
		return true
	case EventTitle:
		m.handleTitleCommand(ctx, ch, ev)
		return true
	case EventSkills:
		m.handleSkillsCommand(ctx, ch, ev)
		return true
	case EventVerbose:
		m.handleVerboseCommand(ctx, ch, ev)
		return true
	case EventModel:
		m.handleModelCommand(ctx, ch, ev)
		return true
	case EventSessions:
		m.handleSessionsCommand(ctx, ch, ev)
		return true
	case EventProfile:
		m.handleProfileCommand(ctx, ch, ev)
		return true
	case EventGateway:
		m.handlePlatformsCommand(ctx, ch, ev)
		return true
	case EventReasoning:
		m.handleReasoningCommand(ctx, ch, ev)
		return true
	case EventUnknown:
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "unknown command")
		return true
	case EventTTS:
		m.handleTTSCommand(ctx, ch, ev)
		return true
	default:
		return false
	}
}

func (m *Manager) handleReasoningCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	reply, err := m.DispatchReasoning(m.sessionKeyForInbound(ev), commandArgs(ev.Text))
	if err != nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Reasoning command error: "+err.Error()+"\n\nUsage: /reasoning [low|medium|high|reset|show] [--global]")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, formatReasoningReply(reply))
}

func (m *Manager) handleBusyCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	args := strings.Fields(ev.Text)
	mode := m.cfg.BusyInputMode
	if mode == "" {
		mode = "interrupt"
	}

	if len(args) >= 2 {
		switch strings.ToLower(args[1]) {
		case "queue", "q":
			mode = "queue"
		case "steer", "s":
			mode = "steer"
		case "interrupt", "i":
			mode = "interrupt"
		case "", "status", "show":
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("⚙ Busy input mode: **%s**\n\n• interrupt — stop current task and respond to new message\n• queue — silently hold message for next turn\n• steer — inject guidance mid-turn", mode))
			return
		default:
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "⚠ Usage: /busy [queue|steer|interrupt|status]")
			return
		}
		m.cfg.BusyInputMode = mode
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("✅ Busy input mode set to **%s**", mode))
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("⚙ Busy input mode: **%s**\nUsage: /busy [queue|steer|interrupt|status]", mode))
}

func commandArgs(body string) []string {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	fields := strings.Fields(body)
	if len(fields) <= 1 {
		return nil
	}
	return fields[1:]
}

func formatReasoningReply(reply ReasoningReply) string {
	scope := strings.TrimSpace(reply.Scope)
	if scope == "" || scope == ReasoningSourceUnset {
		if reply.PersistFailed {
			return "Reasoning effort: default\n\nGlobal persistence failed; no session override is active."
		}
		return "Reasoning effort: default"
	}
	effort := strings.TrimSpace(string(reply.Effort))
	if effort == "" {
		effort = "default"
	}
	text := "Reasoning effort: " + effort + " (" + scope + ")"
	if reply.PersistFailed {
		text += "\n\nGlobal persistence failed; using a session-only override."
	}
	return text
}

func (m *Manager) handleVerboseCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	if !m.cfg.ToolProgressCommandEnabled {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "The `/verbose` command is not enabled for messaging platforms.\n\nEnable it in `config.yaml`:\n```yaml\ndisplay:\n  tool_progress_command: true\n```")
		return
	}
	platform := strings.ToLower(strings.TrimSpace(ev.Platform))
	if platform == "" {
		platform = "unknown"
	}
	mode := nextToolProgressMode(m.toolProgressMode(platform))
	if m.cfg.ToolProgressModes == nil {
		m.cfg.ToolProgressModes = map[string]string{}
	}
	m.cfg.ToolProgressModes[platform] = mode

	text := toolProgressModeDescription(mode)
	if m.cfg.PersistToolProgressMode == nil {
		text += "\n_(could not save to config: persistence unavailable)_"
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
		return
	}
	if err := m.cfg.PersistToolProgressMode(platform, mode); err != nil {
		text += "\n_(could not save to config: " + err.Error() + ")_"
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
		return
	}
	text += "\n_(saved for **" + platform + "** — takes effect on next message)_"
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
}

func (m *Manager) handleSessionsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	if m.cfg.SessionMap == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Sessions are not available in this build.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "📋 Use `/status` for details on the current session. Use `/new` to start fresh.")
}

func (m *Manager) handleModelCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	model := "unknown"
	provider := "unknown"
	if m.cfg.LiveTurnActiveModel != nil {
		model = m.cfg.LiveTurnActiveModel()
	}
	if m.cfg.LiveTurnActiveProvider != nil {
		provider = m.cfg.LiveTurnActiveProvider()
	}
	if model == "" {
		model = "unknown"
	}
	if provider == "" {
		provider = "unknown"
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("🤖 **Model:** `%s`\n📡 **Provider:** `%s`", model, provider))
}

func (m *Manager) handleProfileCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	home := config.GormesHome()
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("👤 **Profile:** `(default)`\n📂 **Home:** `%s`", home))
}

func (m *Manager) handlePlatformsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	platforms := m.formatConnectedPlatforms()
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("📡 **Connected Platforms:** %s\nUse `/status` for full session details.", platforms))
}

func (m *Manager) formatConnectedPlatforms() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

func nextToolProgressMode(current string) string {
	switch normalizeGatewayToolProgressMode(current) {
	case "off":
		return "new"
	case "new":
		return "all"
	case "all":
		return "verbose"
	default:
		return "off"
	}
}

func toolProgressModeDescription(mode string) string {
	switch normalizeGatewayToolProgressMode(mode) {
	case "off":
		return "⚙️ Tool progress: **OFF** — no tool activity shown."
	case "new":
		return "⚙️ Tool progress: **NEW** — shown when tool changes (preview length: `display.tool_preview_length`, default 40)."
	case "verbose":
		return "⚙️ Tool progress: **VERBOSE** — every tool call with safe bounded arguments."
	default:
		return "⚙️ Tool progress: **ALL** — every tool call shown (preview length: `display.tool_preview_length`, default 40)."
	}
}

func (m *Manager) handleSteerCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	parsed := ParseSteerCommand(ev.Text, steerPayloadMetadataFromInbound(ev))
	if parsed.Evidence != "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, string(parsed.Evidence))
		return
	}

	if m.hasActiveTurn() {
		if m.kernel != nil {
			if err := m.kernel.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSteer, Text: parsed.Guidance}); err == nil {
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, string(SteerEvidenceInjected)+": pending for next tool-result boundary; "+string(SteerEvidencePreview)+": "+parsed.Preview)
				return
			}
		}
	}

	followUp := ev
	followUp.Kind = EventSubmit
	followUp.Text = parsed.Guidance
	followUp.Attachments = nil
	queued, full := m.queueFollowUpIfActive(followUp)
	if full {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, followUpQueueFullNotice)
		return
	}
	if queued {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, string(SteerEvidenceUnavailable)+": mid-run injection unavailable; "+string(SteerEvidenceQueued)+"; "+string(SteerEvidencePreview)+": "+parsed.Preview)
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, string(SteerEvidenceUnavailable)+": no active turn; "+string(SteerEvidencePreview)+": "+parsed.Preview)
}

func steerPayloadMetadataFromInbound(ev InboundEvent) SteerPayloadMetadata {
	meta := SteerPayloadMetadata{AttachmentCount: len(ev.Attachments)}
	for _, attachment := range ev.Attachments {
		kind := strings.ToLower(strings.TrimSpace(attachment.Kind + " " + attachment.MediaType))
		if strings.Contains(kind, "image") {
			meta.ImageCount++
		}
	}
	return meta
}

func (m *Manager) dropDuplicateInboundSubmit(ev InboundEvent) bool {
	key := InboundDedupKey(ev)
	if key.Evidence != "" {
		m.recordInboundDedupEvidence(ev, key.Evidence)
		return false
	}

	result := m.inboundDedup.Track(key.Key)
	if result.Evidence != "" {
		m.recordInboundDedupEvidence(ev, result.Evidence)
	}
	return result.Duplicate
}

func (m *Manager) recordInboundDedupEvidence(ev InboundEvent, evidence MessageDeduplicatorEvidence) {
	if evidence == "" {
		return
	}
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      ev.Platform,
		PlatformState: PlatformStateRunning,
		ErrorMessage:  string(evidence),
	})
}

func (m *Manager) dispatchFrame(ctx context.Context, f kernel.RenderFrame, co **coalescer, coCancel *context.CancelFunc) {
	m.rememberUsageFrame(f)
	m.turnMu.Lock()
	platform := m.turnPlatform
	chatID := m.turnChatID
	replyToMsgID := m.turnMsgID
	sessionID := m.turnSessionID
	lastUserText := m.turnLastUserText
	staleInitialIdle := platform != "" && chatID != "" && !m.turnFrameSeen && isStartupIdleFrame(f)
	if !staleInitialIdle && platform != "" && chatID != "" {
		m.turnFrameSeen = true
	}
	m.turnMu.Unlock()

	if platform == "" || chatID == "" {
		return
	}
	if staleInitialIdle {
		return
	}

	ch := m.lookupChannel(platform)
	if ch == nil {
		return
	}
	m.maybeSendTypingAction(ctx, ch, f.Phase, chatID)
	m.dispatchToolProgress(ctx, ch, platform, chatID, f)
	pe, ok := ch.(placeholderEditor)
	if !ok {
		if m.sendNoEdit(ctx, ch, f, chatID, replyToMsgID) {
			if f.Phase == kernel.PhaseIdle {
				m.maybeRunAutoTitle(ctx, f, sessionID, lastUserText)
			}
			m.drainNextFollowUp(ctx)
		}
		return
	}

	switch f.Phase {
	case kernel.PhaseIdle:
		finalPages, media := m.formatFinalDeliveryPages(platform, f)
		if *co != nil {
			(*co).flushImmediateFinal(ctx, finalPages[0], true)
			(*coCancel)()
			*co = nil
			*coCancel = nil
			m.sendFinalPages(ctx, ch, chatID, "", finalPages[1:])
		} else {
			m.sendFinalPages(ctx, ch, chatID, "", finalPages)
		}
		m.deliverMedia(ctx, ch, chatID, replyToMsgID, media)
		m.maybeRunAutoTitle(ctx, f, sessionID, lastUserText)
		m.maybeSendVerboseHint(ctx, ch, platform, chatID, f)
		m.clearToolProgress()
		m.drainNextFollowUp(ctx)
	case kernel.PhaseFailed, kernel.PhaseCancelling:
		text := m.formatError(platform, f)
		if *co != nil {
			(*co).flushImmediateFinal(ctx, text, true)
			(*coCancel)()
			*co = nil
			*coCancel = nil
		} else {
			_, _ = m.sendWithHooks(ctx, ch, chatID, text)
		}
		m.clearToolProgress()
		m.drainNextFollowUp(ctx)
	case kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseReconnecting, kernel.PhaseFinalizing:
		text := m.formatStream(platform, f)
		if text == "" {
			return
		}
		if *co == nil {
			cCtx, cancel := context.WithCancel(ctx)
			*coCancel = cancel
			opts := []coalescerOption{
				coalescerFreshFinalAfter(m.cfg.FreshFinalAfter),
				coalescerNow(m.now),
				coalescerEvidenceSink(m.cfg.CoalescerEvidenceSink),
				coalescerInitialTextSend(),
			}
			nc := newCoalescer(hookedPlaceholderEditor{
				base:         pe,
				manager:      m,
				platform:     platform,
				replyToMsgID: replyToMsgID,
			}, time.Duration(m.cfg.CoalesceMs)*time.Millisecond, chatID, opts...)
			*co = nc
			go nc.run(cCtx)
		}
		(*co).setPending(text)
	}
}

func (m *Manager) dispatchToolProgress(ctx context.Context, ch Channel, platform, chatID string, f kernel.RenderFrame) {
	if _, ok := ch.(MessageEditor); !ok {
		return
	}
	text := m.formatToolProgress(platform, f)
	if strings.TrimSpace(text) == "" {
		return
	}

	m.toolProgressMu.Lock()
	sameTarget := m.toolProgressPlat == platform && m.toolProgressChatID == chatID
	if sameTarget && m.toolProgressText == text {
		m.toolProgressMu.Unlock()
		return
	}
	msgID := ""
	if sameTarget {
		msgID = m.toolProgressMsgID
	}
	m.toolProgressMu.Unlock()

	if msgID != "" {
		if editor, ok := ch.(MessageEditor); ok {
			if err := editor.EditMessage(ctx, chatID, msgID, text); err == nil {
				m.toolProgressMu.Lock()
				if m.toolProgressPlat == platform && m.toolProgressChatID == chatID && m.toolProgressMsgID == msgID {
					m.toolProgressText = text
				}
				m.toolProgressMu.Unlock()
				return
			}
		}
	}

	newMsgID, err := m.sendWithHooks(ctx, ch, chatID, text)
	if err != nil {
		return
	}
	m.toolProgressMu.Lock()
	m.toolProgressPlat = platform
	m.toolProgressChatID = chatID
	m.toolProgressMsgID = newMsgID
	m.toolProgressText = text
	m.toolProgressMu.Unlock()
}

func (m *Manager) sendNoEdit(ctx context.Context, ch Channel, f kernel.RenderFrame, chatID, replyToMsgID string) bool {
	switch f.Phase {
	case kernel.PhaseIdle:
		finalPages, media := m.formatFinalDeliveryPages(ch.Name(), f)
		m.sendFinalPages(ctx, ch, chatID, replyToMsgID, finalPages)
		m.deliverMedia(ctx, ch, chatID, replyToMsgID, media)
		return true
	case kernel.PhaseFailed, kernel.PhaseCancelling:
		_, _ = m.sendWithHooksReply(ctx, ch, chatID, replyToMsgID, m.formatError(ch.Name(), f))
		return true
	case kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseReconnecting, kernel.PhaseFinalizing:
		if text := m.formatStream(ch.Name(), f); text != "" {
			_, _ = m.sendWithHooksReply(ctx, ch, chatID, replyToMsgID, text)
		}
	}
	return false
}

func (m *Manager) sendWithHooks(ctx context.Context, ch Channel, chatID, text string) (string, error) {
	return m.sendWithHooksReply(ctx, ch, chatID, "", text)
}

func (m *Manager) sendFinalPages(ctx context.Context, ch Channel, chatID, replyToMsgID string, pages []string) {
	for i, page := range pages {
		if i == 0 {
			_, _ = m.sendWithHooksReply(ctx, ch, chatID, replyToMsgID, page)
			continue
		}
		_, _ = m.sendWithHooks(ctx, ch, chatID, page)
	}
}

func (m *Manager) sendWithHooksReply(ctx context.Context, ch Channel, chatID, replyToMsgID, text string) (string, error) {
	if ch == nil {
		return "", nil
	}
	ev := HookEvent{
		Point:    HookBeforeSend,
		Platform: ch.Name(),
		ChatID:   chatID,
		Text:     text,
	}
	m.fireHook(ctx, ev)

	var (
		msgID string
		err   error
	)
	if replyToMsgID != "" {
		if replySender, ok := ch.(ReplySender); ok {
			msgID, err = replySender.SendReply(ctx, chatID, replyToMsgID, text)
		} else {
			msgID, err = ch.Send(ctx, chatID, text)
		}
	} else {
		msgID, err = ch.Send(ctx, chatID, text)
	}
	if err != nil {
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
			Platform:      ch.Name(),
			PlatformState: PlatformStateFailed,
			ErrorMessage:  err.Error(),
		})
		m.fireHook(ctx, HookEvent{
			Point:    HookOnError,
			Platform: ch.Name(),
			ChatID:   chatID,
			Text:     text,
			Err:      err,
		})
		return "", err
	}

	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      ch.Name(),
		PlatformState: PlatformStateRunning,
	})
	m.fireHook(ctx, HookEvent{
		Point:    HookAfterSend,
		Platform: ch.Name(),
		ChatID:   chatID,
		MsgID:    msgID,
		Text:     text,
	})
	return msgID, nil
}

func (m *Manager) fireHook(ctx context.Context, ev HookEvent) {
	if m.cfg.Hooks == nil {
		return
	}
	m.cfg.Hooks.Fire(ctx, ev)
}

func (m *Manager) writeRuntimeStatus(ctx context.Context, update RuntimeStatusUpdate) {
	if m.cfg.RuntimeStatus == nil {
		return
	}
	if err := m.cfg.RuntimeStatus.UpdateRuntimeStatus(ctx, update); err != nil && !errors.Is(err, context.Canceled) {
		m.log.Debug("write gateway runtime status", "err", err)
	}
}

func (m *Manager) handleRestartCommand(ctx context.Context, ch Channel, ev InboundEvent) error {
	now := m.now()
	if marker, duplicate, err := m.restartDuplicate(ctx, ev); err != nil {
		m.log.Warn("read restart takeover marker", "err", err)
	} else if duplicate {
		evidence := restartDuplicateEvidence(marker, ev, now)
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{DuplicateRestartEvidence: &evidence})
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "duplicate_restart_suppressed")
		return nil
	}

	restartRequested := true
	activeAgents := m.activeAgentCount()
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		RestartRequested: &restartRequested,
		ActiveAgents:     &activeAgents,
	})

	serviceManagerAvailable := m.restartServiceManagerAvailable()
	selfRestartAvailable := m.cfg.Restart.SelfRestart != nil
	if !serviceManagerAvailable && !m.restartMarkerStoreAvailable() && !selfRestartAvailable {
		evidence := RuntimeServiceManagerUnavailableEvidence{
			Source:   ev.Platform,
			ChatID:   ev.ChatID,
			ThreadID: ev.ThreadID,
			Reason:   "restart marker store and service-manager restart exit path are unavailable",
			At:       now.Format(time.RFC3339Nano),
		}
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{ServiceManagerUnavailableEvidence: &evidence})
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Restart unavailable: no restart manager is configured. Restart the gateway process manually or rerun install.sh with gateway restart enabled.")
		return nil
	}

	marker, err := m.writeRestartTakeoverMarker(ctx, ev, now)
	if err != nil {
		evidence := RuntimeServiceManagerUnavailableEvidence{
			Source:   ev.Platform,
			ChatID:   ev.ChatID,
			ThreadID: ev.ThreadID,
			Reason:   "restart takeover marker write failed: " + err.Error(),
			At:       now.Format(time.RFC3339Nano),
		}
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{ServiceManagerUnavailableEvidence: &evidence})
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Restart marker could not be written; restart was not started. Fix runtime state permissions or restart the gateway process manually.")
		return nil
	}
	if m.restartMarkerStoreAvailable() {
		takeoverEvidence := restartTakeoverEvidence(marker, RestartTakeoverMarkerStatusWritten, now)
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{TakeoverMarkerEvidence: &takeoverEvidence})
	}

	if activeAgents > 0 {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("restart_requested: draining %d active agent(s) before restart.", activeAgents))
	} else if serviceManagerAvailable {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "restart_requested: handing off to service manager.")
	} else {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "restart_requested: handing off to gateway restart path.")
	}

	drainCtx, cancel := context.WithTimeout(context.Background(), m.restartDrainTimeout())
	defer cancel()
	if err := m.ShutdownWithDrainReason(drainCtx, DrainReasonRestartTimeout); err != nil &&
		!errors.Is(err, context.DeadlineExceeded) &&
		!errors.Is(err, context.Canceled) {
		m.log.Warn("gateway restart drain", "err", err)
	}

	if !serviceManagerAvailable && selfRestartAvailable {
		if err := m.cfg.Restart.SelfRestart(); err != nil {
			m.log.Warn("gateway self-restart failed", "err", err)
			return RestartRequestedError{
				Code:    GatewayServiceRestartExitCode,
				Message: fmt.Sprintf("self-restart failed: %v; process will exit and must be restarted manually", err),
			}
		}
		return nil
	}
	return RestartRequestedError{
		Code:    GatewayServiceRestartExitCode,
		Message: "gateway restart requested",
	}
}

func (m *Manager) restartDuplicate(ctx context.Context, ev InboundEvent) (RestartTakeoverMarker, bool, error) {
	store := m.cfg.Restart.MarkerStore
	if store == nil {
		return RestartTakeoverMarker{}, false, nil
	}
	return store.SuppressDuplicate(ctx, ev)
}

func (m *Manager) writeRestartTakeoverMarker(ctx context.Context, ev InboundEvent, now time.Time) (RestartTakeoverMarker, error) {
	store := m.cfg.Restart.MarkerStore
	if store == nil {
		return RestartTakeoverMarker{}, nil
	}
	marker := RestartTakeoverMarker{
		SourcePlatform: strings.ToLower(strings.TrimSpace(ev.Platform)),
		ChatID:         strings.TrimSpace(ev.ChatID),
		ThreadID:       strings.TrimSpace(ev.ThreadID),
		UpdateID:       restartUpdateID(ev),
		MessageID:      strings.TrimSpace(ev.MsgID),
		Generation:     m.runtimeStatusGeneration(ctx),
		RequestedAt:    now.Format(time.RFC3339Nano),
	}
	if err := store.Write(ctx, marker); err != nil {
		return marker, err
	}
	return marker, nil
}

func (m *Manager) restartServiceManagerAvailable() bool {
	if m.cfg.Restart.ServiceManagerAvailable == nil {
		return false
	}
	return m.cfg.Restart.ServiceManagerAvailable()
}

func (m *Manager) restartMarkerStoreAvailable() bool {
	store := m.cfg.Restart.MarkerStore
	return store != nil && strings.TrimSpace(store.path) != ""
}

func (m *Manager) restartDrainTimeout() time.Duration {
	if m.cfg.Restart.DrainTimeout > 0 {
		return m.cfg.Restart.DrainTimeout
	}
	return time.Minute
}

func (m *Manager) runtimeStatusGeneration(ctx context.Context) uint64 {
	reader, ok := m.cfg.RuntimeStatus.(interface {
		ReadRuntimeStatus(context.Context) (RuntimeStatus, error)
	})
	if !ok {
		return 0
	}
	status, err := reader.ReadRuntimeStatus(ctx)
	if err != nil {
		m.log.Debug("read gateway runtime status generation", "err", err)
		return 0
	}
	return status.Generation
}

func restartTakeoverEvidence(marker RestartTakeoverMarker, status RestartTakeoverMarkerStatus, at time.Time) RuntimeRestartTakeoverEvidence {
	return RuntimeRestartTakeoverEvidence{
		Status:     status,
		Source:     marker.SourcePlatform,
		ChatID:     marker.ChatID,
		ThreadID:   marker.ThreadID,
		UpdateID:   marker.UpdateID,
		MessageID:  marker.MessageID,
		Generation: marker.Generation,
		At:         at.UTC().Format(time.RFC3339Nano),
	}
}

func restartDuplicateEvidence(marker RestartTakeoverMarker, ev InboundEvent, at time.Time) RuntimeRestartDuplicateEvidence {
	source := marker.SourcePlatform
	if source == "" {
		source = ev.Platform
	}
	chatID := marker.ChatID
	if chatID == "" {
		chatID = ev.ChatID
	}
	threadID := marker.ThreadID
	if threadID == "" {
		threadID = ev.ThreadID
	}
	updateID := marker.UpdateID
	if updateID == "" {
		updateID = restartUpdateID(ev)
	}
	messageID := marker.MessageID
	if messageID == "" {
		messageID = ev.MsgID
	}
	return RuntimeRestartDuplicateEvidence{
		Status:     RestartDuplicateStatusSuppressed,
		Source:     source,
		ChatID:     chatID,
		ThreadID:   threadID,
		UpdateID:   updateID,
		MessageID:  messageID,
		Generation: marker.Generation,
		At:         at.UTC().Format(time.RFC3339Nano),
	}
}

func (m *Manager) ConsumeRestartTakeoverMarker(ctx context.Context) error {
	store := m.cfg.Restart.MarkerStore
	if store == nil {
		return nil
	}
	marker, ok, expired, err := store.Read(ctx)
	if err != nil {
		return err
	}
	now := m.now()
	if expired {
		evidence := restartTakeoverEvidence(marker, RestartTakeoverMarkerStatusExpired, now)
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{TakeoverMarkerEvidence: &evidence})
		return nil
	}
	if !ok || marker.NotificationSentAt != "" {
		return nil
	}
	ch := m.lookupChannel(marker.SourcePlatform)
	if ch == nil {
		return nil
	}
	evidence := restartTakeoverEvidence(marker, RestartTakeoverMarkerStatusSeen, now)
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{TakeoverMarkerEvidence: &evidence})
	if _, err := m.sendWithHooks(ctx, ch, marker.ChatID, "Gateway restarted successfully. Your session continues."); err != nil {
		return err
	}
	return store.MarkNotificationSent(ctx, marker, now)
}

func (m *Manager) allowed(ev InboundEvent) bool {
	want, ok := m.cfg.AllowedChats[ev.Platform]
	if !ok || want == "" {
		return false
	}
	return ev.ChatID == want
}

func (m *Manager) lookupChannel(name string) Channel {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.channels[name]
}

func (m *Manager) connectedPlatforms() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m *Manager) pinTurn(platform, chatID, msgID string) {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	m.turnPlatform = platform
	m.turnChatID = chatID
	m.turnMsgID = msgID
	m.turnSessionKey = ""
	m.turnSessionID = ""
	m.turnSource = SessionSource{}
	m.turnCancelled = false
	m.turnFrameSeen = false
	m.turnLastUserText = ""
	m.turnKernel = nil
	m.resetToolProgress()
}

func (m *Manager) clearTurn() {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	m.turnPlatform = ""
	m.turnChatID = ""
	m.turnMsgID = ""
	m.turnSessionKey = ""
	m.turnSessionID = ""
	m.turnSource = SessionSource{}
	m.turnCancelled = false
	m.turnFrameSeen = false
	m.turnLastUserText = ""
	m.turnKernel = nil
	m.resetToolProgress()
}

func (m *Manager) setTurnLastUserText(text string) {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	m.turnLastUserText = text
}

func (m *Manager) hasActiveTurn() bool {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	return m.hasActiveTurnLocked()
}

func (m *Manager) isShuttingDown() bool {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	return m.shuttingDown
}

func (m *Manager) activeAgentCount() int {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	if m.turnPlatform == "" {
		return 0
	}
	return 1
}

func (m *Manager) setPinnedTurnSession(sessionKey, sessionID string, source SessionSource) {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	m.turnSessionKey = sessionKey
	m.turnSessionID = sessionID
	m.turnSource = source
}

func (m *Manager) markTurnCancelled() {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	m.turnCancelled = true
}

func (m *Manager) activeTurnSnapshot() (activeTurnSnapshot, bool) {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	if m.turnPlatform == "" || m.turnChatID == "" {
		return activeTurnSnapshot{}, false
	}
	return activeTurnSnapshot{
		Platform:     m.turnPlatform,
		ChatID:       m.turnChatID,
		MsgID:        m.turnMsgID,
		SessionKey:   m.turnSessionKey,
		SessionID:    m.turnSessionID,
		Source:       m.turnSource,
		Cancelled:    m.turnCancelled,
		LastUserText: m.turnLastUserText,
	}, true
}

func (m *Manager) formatStream(platform string, f kernel.RenderFrame) string {
	if platform == "telegram" {
		return FormatStreamTelegram(f)
	}
	return FormatStreamPlain(f)
}

func (m *Manager) formatToolProgress(platform string, f kernel.RenderFrame) string {
	mode := m.toolProgressMode(platform)
	if platform == "telegram" {
		return FormatToolProgressTelegramMode(f, mode)
	}
	return FormatToolProgressPlainMode(f, mode)
}

func (m *Manager) clearToolProgress() {
	m.toolProgressMu.Lock()
	m.toolProgressMsgID = ""
	m.toolProgressText = ""
	m.toolProgressChatID = ""
	m.toolProgressPlat = ""
	m.toolProgressMu.Unlock()
}

func (m *Manager) maybeSendVerboseHint(ctx context.Context, ch Channel, platform, chatID string, f kernel.RenderFrame) {
	mode := m.toolProgressMode(platform)
	if mode != "all" {
		return
	}
	key := platform + ":" + chatID
	m.verboseHintMu.Lock()
	if m.verboseHintSent == nil {
		m.verboseHintSent = map[string]bool{}
	}
	if m.verboseHintSent[key] {
		m.verboseHintMu.Unlock()
		return
	}
	m.verboseHintSent[key] = true
	m.verboseHintMu.Unlock()

	if toolMaxDuration(f.SoulEvents) < 30*time.Second {
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, chatID, "💡 **Tip:** use `/verbose` to see detailed tool output in this chat.")
}

func toolMaxDuration(events []kernel.SoulEntry) time.Duration {
	if len(events) < 2 {
		return 0
	}
	first := events[0].At
	last := events[len(events)-1].At
	if first.After(last) {
		return 0
	}
	return last.Sub(first)
}

func (m *Manager) toolProgressMode(platform string) string {
	key := strings.ToLower(strings.TrimSpace(platform))
	if key != "" && len(m.cfg.ToolProgressModes) > 0 {
		if mode := strings.TrimSpace(m.cfg.ToolProgressModes[key]); mode != "" {
			return normalizeGatewayToolProgressMode(mode)
		}
	}
	if mode := strings.TrimSpace(m.cfg.ToolProgressMode); mode != "" {
		return normalizeGatewayToolProgressMode(mode)
	}
	return defaultToolProgressModeForPlatform(key)
}

func defaultToolProgressModeForPlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "telegram", "discord", "api_server":
		return "all"
	case "mattermost", "matrix", "feishu", "whatsapp":
		return "new"
	case "slack", "signal", "bluebubbles", "weixin", "wecom", "wecom_callback", "dingtalk",
		"email", "sms", "webhook", "homeassistant":
		return "off"
	default:
		return "all"
	}
}

func (m *Manager) formatFinal(platform string, f kernel.RenderFrame) string {
	if platform == "telegram" {
		return FormatFinalTelegram(f)
	}
	return FormatFinalPlain(f)
}

func (m *Manager) formatFinalDelivery(platform string, f kernel.RenderFrame) (string, []OutboundMedia) {
	content := PrepareMediaDeliveryContent(FinalAssistantText(f))
	text := content.Text
	if strings.TrimSpace(text) == "" && len(content.Media) > 0 {
		text = "Audio attached."
	}
	if platform == "telegram" {
		return FormatFinalTelegramText(text), content.Media
	}
	return FormatFinalPlainText(text), content.Media
}

func (m *Manager) formatFinalDeliveryPages(platform string, f kernel.RenderFrame) ([]string, []OutboundMedia) {
	text, media := m.formatFinalDelivery(platform, f)
	if platform == "telegram" {
		return paginateTelegramText(text), media
	}
	return paginatePlainText(text), media
}

func (m *Manager) deliverMedia(ctx context.Context, ch Channel, chatID, replyToMsgID string, media []OutboundMedia) {
	if len(media) == 0 || ch == nil {
		return
	}
	sender, ok := ch.(MediaSender)
	if !ok {
		for range media {
			_, _ = m.sendWithHooksReply(ctx, ch, chatID, replyToMsgID, "Media attachment unavailable.")
		}
		return
	}
	for _, item := range media {
		if _, err := sender.SendMedia(ctx, chatID, replyToMsgID, item); err != nil {
			m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
				Platform:      ch.Name(),
				PlatformState: PlatformStateFailed,
				ErrorMessage:  err.Error(),
			})
			m.fireHook(ctx, HookEvent{
				Point:    HookOnError,
				Platform: ch.Name(),
				ChatID:   chatID,
				Text:     "MEDIA:[redacted]",
				Err:      err,
			})
		}
	}
}

func (m *Manager) formatError(platform string, f kernel.RenderFrame) string {
	if platform == "telegram" {
		return FormatErrorTelegram(f)
	}
	return FormatErrorPlain(f)
}

func isStartupIdleFrame(f kernel.RenderFrame) bool {
	return f.Phase == kernel.PhaseIdle &&
		f.StatusText == "idle" &&
		f.DraftText == "" &&
		f.LastError == "" &&
		len(f.History) == 0
}

func (m *Manager) persistSession(ctx context.Context, f kernel.RenderFrame) {
	if m.cfg.SessionMap == nil {
		return
	}
	m.turnMu.Lock()
	platform := m.turnPlatform
	chatID := m.turnChatID
	m.turnMu.Unlock()
	if platform == "" || chatID == "" || f.SessionID == "" {
		return
	}
	key := platform + ":" + chatID
	if err := m.cfg.SessionMap.Put(ctx, key, f.SessionID); err != nil {
		m.log.Warn("persist session_id", "key", key, "session_id", f.SessionID, "err", err)
	}
}

func (m *Manager) hasActiveTurnLocked() bool {
	return m.turnPlatform != "" || m.turnChatID != "" || m.turnMsgID != "" || len(m.followUps) > 0
}

func (m *Manager) queueFollowUpIfActive(ev InboundEvent) (queued bool, full bool) {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	if !m.hasActiveTurnLocked() {
		return false, false
	}
	if len(m.followUps) >= followUpQueueCap {
		return false, true
	}
	m.followUps = append(m.followUps, ev)
	return true, false
}

func (m *Manager) drainNextFollowUp(ctx context.Context) {
	for {
		next, ok := m.popNextFollowUpAsActive()
		if !ok {
			return
		}
		ch := m.lookupChannel(next.Platform)
		if ch == nil {
			m.log.Warn("queued follow-up for unknown channel", "platform", next.Platform)
			m.clearTurn()
			continue
		}
		if m.submitPinned(ctx, ch, next) {
			return
		}
	}
}

func (m *Manager) popNextFollowUpAsActive() (InboundEvent, bool) {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	if len(m.followUps) == 0 {
		m.turnPlatform = ""
		m.turnChatID = ""
		m.turnMsgID = ""
		return InboundEvent{}, false
	}
	next := m.followUps[0]
	copy(m.followUps, m.followUps[1:])
	m.followUps[len(m.followUps)-1] = InboundEvent{}
	m.followUps = m.followUps[:len(m.followUps)-1]
	m.turnPlatform = next.Platform
	m.turnChatID = next.ChatID
	m.turnMsgID = next.MsgID
	m.turnSessionKey = ""
	m.turnSessionID = ""
	m.turnSource = SessionSource{}
	m.turnCancelled = false
	m.turnFrameSeen = false
	m.resetToolProgress()
	return next, true
}

func (m *Manager) resetToolProgress() {
	m.toolProgressMu.Lock()
	defer m.toolProgressMu.Unlock()
	m.toolProgressMsgID = ""
	m.toolProgressText = ""
	m.toolProgressChatID = ""
	m.toolProgressPlat = ""
}

func (m *Manager) markDrainTimeoutResumePending(ctx context.Context, reason DrainTimeoutReason) {
	state, ok := m.activeTurnSnapshot()
	if !ok {
		return
	}
	now := m.now()
	if state.Source.Platform == "" {
		state.Source.Platform = state.Platform
	}
	if state.Source.ChatID == "" {
		state.Source.ChatID = state.ChatID
	}
	if state.SessionKey == "" {
		state.SessionKey = state.Platform + ":" + state.ChatID
	}
	if state.SessionID == "" {
		resolved, err := resolveSession(ctx, m.cfg.SessionMap, state.SessionKey)
		if err != nil {
			m.log.Warn("resolve active session for drain timeout", "key", state.SessionKey, "err", err)
		}
		state.SessionID = resolved.SessionID
	}
	if state.SessionID == "" {
		return
	}

	timeoutEvidence := RuntimeDrainTimeoutEvidence{
		SessionKey:   state.SessionKey,
		SessionID:    state.SessionID,
		Source:       state.Source.Platform,
		ChatID:       state.Source.ChatID,
		UserID:       state.Source.UserID,
		Reason:       string(reason),
		TimeoutAt:    now.Format(time.RFC3339Nano),
		ActiveAgents: m.activeAgentCount(),
	}
	m.writeRuntimeStatus(ctx, RuntimeStatusUpdate{DrainTimeoutEvidence: &timeoutEvidence})

	if state.Cancelled {
		m.markNonResumable(ctx, state, session.NonResumableCancelled, now)
		return
	}
	if meta, ok := m.getSessionMetadata(ctx, state.SessionID); ok && meta.NonResumableReason != "" {
		m.writeNonResumableEvidence(ctx, RuntimeNonResumableEvidence{
			SessionKey: state.SessionKey,
			SessionID:  state.SessionID,
			Source:     state.Source.Platform,
			ChatID:     state.Source.ChatID,
			UserID:     state.Source.UserID,
			Reason:     meta.NonResumableReason,
			At:         now.Format(time.RFC3339Nano),
		})
		return
	}

	writer, ok := m.cfg.SessionMap.(sessionMetadataWriter)
	if !ok {
		return
	}
	meta := session.Metadata{
		SessionID:      state.SessionID,
		Source:         state.Source.Platform,
		ChatID:         state.Source.ChatID,
		UserID:         state.Source.UserID,
		ResumePending:  true,
		ResumeReason:   string(reason),
		ResumeMarkedAt: now.Unix(),
		UpdatedAt:      now.Unix(),
	}
	if err := writer.PutMetadata(ctx, meta); err != nil {
		m.log.Warn("mark resume pending", "session_id", state.SessionID, "err", err)
		return
	}
	evidence := RuntimeResumePendingEvidence{
		SessionKey: state.SessionKey,
		SessionID:  state.SessionID,
		Source:     state.Source.Platform,
		ChatID:     state.Source.ChatID,
		UserID:     state.Source.UserID,
		Reason:     string(reason),
		MarkedAt:   now.Format(time.RFC3339Nano),
	}
	m.writeRuntimeStatus(ctx, RuntimeStatusUpdate{ResumePendingEvidence: &evidence})
}

func (m *Manager) markNonResumable(ctx context.Context, state activeTurnSnapshot, reason string, at time.Time) {
	if writer, ok := m.cfg.SessionMap.(sessionMetadataWriter); ok && state.SessionID != "" {
		if err := writer.PutMetadata(ctx, session.Metadata{
			SessionID:          state.SessionID,
			Source:             state.Source.Platform,
			ChatID:             state.Source.ChatID,
			UserID:             state.Source.UserID,
			NonResumableReason: reason,
			NonResumableAt:     at.Unix(),
			UpdatedAt:          at.Unix(),
		}); err != nil {
			m.log.Warn("mark non-resumable session", "session_id", state.SessionID, "err", err)
		}
	}
	m.writeNonResumableEvidence(ctx, RuntimeNonResumableEvidence{
		SessionKey: state.SessionKey,
		SessionID:  state.SessionID,
		Source:     state.Source.Platform,
		ChatID:     state.Source.ChatID,
		UserID:     state.Source.UserID,
		Reason:     reason,
		At:         at.Format(time.RFC3339Nano),
	})
}

func (m *Manager) getSessionMetadata(ctx context.Context, sessionID string) (session.Metadata, bool) {
	reader, ok := m.cfg.SessionMap.(sessionMetadataReader)
	if !ok || sessionID == "" {
		return session.Metadata{}, false
	}
	meta, ok, err := reader.GetMetadata(ctx, sessionID)
	if err != nil {
		m.log.Warn("read session metadata", "session_id", sessionID, "err", err)
		return session.Metadata{}, false
	}
	if ok && meta.MigratedMemoryFlushed {
		m.writeExpiryFinalizedEvidence(ctx, RuntimeExpiryFinalizedEvidence{
			SessionID:             meta.SessionID,
			Source:                meta.Source,
			ChatID:                meta.ChatID,
			UserID:                meta.UserID,
			ExpiryFinalized:       meta.ExpiryFinalized,
			MigratedMemoryFlushed: meta.MigratedMemoryFlushed,
		})
	}
	return meta, ok
}

func (m *Manager) clearResumePending(ctx context.Context, sessionID string) {
	clearer, ok := m.cfg.SessionMap.(sessionResumeClearer)
	if !ok || sessionID == "" {
		return
	}
	if _, err := clearer.ClearResumePending(ctx, sessionID); err != nil {
		m.log.Warn("clear resume pending", "session_id", sessionID, "err", err)
	}
}

func (m *Manager) writeNonResumableEvidence(ctx context.Context, evidence RuntimeNonResumableEvidence) {
	m.writeRuntimeStatus(ctx, RuntimeStatusUpdate{NonResumableEvidence: &evidence})
}

func (m *Manager) writeExpiryFinalizedEvidence(ctx context.Context, evidence RuntimeExpiryFinalizedEvidence) {
	m.writeRuntimeStatus(ctx, RuntimeStatusUpdate{ExpiryFinalizedEvidence: &evidence})
}

func (m *Manager) writeExpiryFinalizeEvidence(ctx context.Context, evidence RuntimeExpiryFinalizeEvidence) {
	m.writeRuntimeStatus(ctx, RuntimeStatusUpdate{ExpiryFinalizeEvidence: &evidence})
}

func resumePendingNote(reason string) string {
	reasonPhrase := "a gateway interruption"
	switch reason {
	case session.ResumeReasonRestartTimeout:
		reasonPhrase = "a gateway restart"
	case session.ResumeReasonShutdownTimeout:
		reasonPhrase = "a gateway shutdown"
	}
	return "[System note: Your previous turn in this session was interrupted by " +
		reasonPhrase +
		". The conversation history below is intact. If it contains unfinished tool result(s), process them first and summarize what was accomplished, then address the user's new message below.]"
}

func (m *Manager) refreshConversationalSessionMetadata(ctx context.Context, ev InboundEvent, sessionKey string, resolved resolvedSession, source SessionSource) resolvedSession {
	key := strings.TrimSpace(sessionKey)
	if key == "" || m.cfg.SessionMap == nil {
		return resolved
	}
	sessionID := strings.TrimSpace(resolved.SessionID)
	if sessionID == "" || sessionID == key {
		sessionID = generateStatusSessionIDForKey(m.now(), ev, key)
		if err := m.cfg.SessionMap.Put(ctx, key, sessionID); err != nil {
			m.log.Warn("persist conversational session_id", "key", key, "session_id", sessionID, "err", err)
			if strings.TrimSpace(resolved.SessionID) != "" {
				return resolved
			}
		}
		resolved.SessionID = sessionID
	}
	writer, ok := m.cfg.SessionMap.(sessionMetadataWriter)
	if !ok || sessionID == "" {
		return resolved
	}
	now := m.now().Unix()
	meta := session.Metadata{
		SessionID: sessionID,
		Source:    source.Platform,
		ChatID:    source.ChatID,
		UserID:    source.UserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := writer.PutMetadata(ctx, meta); err != nil {
		m.log.Warn("refresh conversational session metadata", "session_id", sessionID, "err", err)
	}
	return resolved
}

func (m *Manager) rememberAllowedInboundSource(ctx context.Context, source SessionSource) {
	store := m.cfg.RememberedSourceStore
	if store == nil {
		return
	}
	entry := RememberedSourceEntryFromSessionSource(source)
	if entry.Platform == "" || entry.ID == "" {
		return
	}
	if err := store.RememberSource(ctx, entry); err != nil {
		m.log.Warn("channel_directory_source_unavailable", "platform", entry.Platform, "code", "channel_directory_source_unavailable")
	}
}

func (m *Manager) submitPinned(ctx context.Context, ch Channel, ev InboundEvent) bool {
	route := m.agentRouteForInbound(ev)
	sessionKey := strings.TrimSpace(route.SessionKey)
	if sessionKey == "" {
		sessionKey = ev.ChatKey()
	}
	resolved, err := resolveSession(ctx, m.cfg.SessionMap, sessionKey)
	if err != nil {
		m.log.Warn("load session mapping", "key", sessionKey, "err", err)
	}
	source := sessionSourceFromInbound(ev)
	m.rememberAllowedInboundSource(ctx, source)
	submitText := ev.SubmitText()
	var clearPendingSessionID string
	var clearBlockedMapping bool
	if meta, ok := m.getSessionMetadata(ctx, resolved.SessionID); ok {
		if meta.NonResumableReason != "" {
			resolved.NonResumableSessionID = resolved.SessionID
			resolved.NonResumableReason = meta.NonResumableReason
			resolved.SessionID = sessionKey
			clearBlockedMapping = true
			m.writeNonResumableEvidence(ctx, RuntimeNonResumableEvidence{
				SessionKey: sessionKey,
				SessionID:  resolved.NonResumableSessionID,
				Source:     source.Platform,
				ChatID:     source.ChatID,
				UserID:     source.UserID,
				Reason:     meta.NonResumableReason,
				At:         m.now().Format(time.RFC3339Nano),
			})
		} else if meta.ResumePending {
			reason := meta.ResumeReason
			if reason == "" {
				reason = session.ResumeReasonRestartTimeout
			}
			submitText = resumePendingNote(reason) + "\n\n" + submitText
			clearPendingSessionID = resolved.SessionID
		}
	}
	if !clearBlockedMapping {
		resolved = m.refreshConversationalSessionMetadata(ctx, ev, sessionKey, resolved, source)
	}
	m.setPinnedTurnSession(sessionKey, resolved.SessionID, source)
	m.setTurnLastUserText(ev.Text)
	sessionBlock := BuildSessionContextPrompt(SessionContext{
		Source:                source,
		Agent:                 route.SessionContext(),
		SessionKey:            sessionKey,
		SessionID:             resolved.SessionID,
		RequestedSessionID:    resolved.RequestedSessionID,
		ResumePath:            resolved.ResumePath,
		ResumeStatus:          resolved.ResumeStatus,
		NonResumableSessionID: resolved.NonResumableSessionID,
		NonResumableReason:    resolved.NonResumableReason,
		ConnectedPlatforms:    m.connectedPlatforms(),
	})
	seams := m.liveTurnPromptSeamsForAgent(route)
	sessionContext, _, _ := assembleLiveTurnPrompt(seams, submitText, resolved.SessionID, sessionBlock)
	snapshot := m.agentRuntimeSnapshot(route)
	submitter := KernelSubmitter(m.kernel)
	if m.cfg.AgentRuntimeFactory != nil && route.Enabled {
		runtime, err := m.agentRuntimeForRoute(ctx, route, snapshot)
		if err != nil {
			errMsg := "agent_runtime_unavailable agent_id=" + route.Decision.AgentID
			m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
				Platform:      ev.Platform,
				PlatformState: PlatformStateRunning,
				ErrorMessage:  errMsg,
			})
			m.clearTurn()
			configHint := gormesHomeHint()
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf(
				"Agent `%s` is unavailable (agent_runtime_unavailable).\n\nCheck your provider config in %s/config.toml → [hermes] endpoint + api_key.",
				route.Decision.AgentID, configHint))
			return false
		}
		submitter = runtime
	} else {
		if err := m.prepareKernelForAgentSession(sessionKey); err != nil {
			m.clearTurn()
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Busy — try again in a second.")
			return false
		}
	}
	if submitter == nil {
		m.clearTurn()
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Busy — try again in a second.")
		return false
	}
	m.setPinnedTurnKernel(submitter)
	if err := submitter.Submit(kernel.PlatformEvent{
		Kind:           kernel.PlatformEventSubmit,
		Text:           submitText,
		Tools:          snapshot.Tools,
		Skills:         snapshot.Skills,
		ToolSafety:     snapshot.ToolSafety,
		Model:          route.Decision.Model,
		SessionID:      resolved.SessionID,
		SessionContext: sessionContext,
	}); err != nil {
		m.clearTurn()
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Busy — try again in a second.")
		return false
	}
	if clearPendingSessionID != "" {
		m.clearResumePending(ctx, clearPendingSessionID)
	}
	if clearBlockedMapping && m.cfg.SessionMap != nil {
		if err := m.cfg.SessionMap.Put(ctx, sessionKey, ""); err != nil {
			m.log.Warn("clear non-resumable session mapping", "key", sessionKey, "err", err)
		}
	}
	return true
}

func gormesHomeHint() string {
	return config.GormesHome()
}
