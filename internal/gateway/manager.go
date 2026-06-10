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
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
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

// KanbanSlashRunner executes a full /kanban command and returns channel-safe
// command output.
type KanbanSlashRunner func(context.Context, string) (string, error)

// ManagerConfig drives the shared gateway manager.
type ManagerConfig struct {
	AllowedChats          map[string]string
	AllowedUsers          map[string]map[string]bool
	AllowedChatWhitelists map[string]WhitelistConfig
	AllowDiscovery        map[string]bool
	CoalesceMs            int
	FreshFinalAfter       time.Duration
	// ToolProgressMode mirrors Hermes gateway display.tool_progress for
	// editable channel progress messages. Empty and unknown values default to all.
	ToolProgressMode string
	// ToolProgressCommandEnabled gates Hermes' /verbose command on messaging
	// platforms. Hermes defaults this gate off.
	ToolProgressCommandEnabled bool
	BusyInputMode              string // interrupt, queue, or steer
	// ReplyMode controls whether outbound gateway messages quote the triggering
	// inbound message via platform reply threading. Hermes parity values:
	//   "first" — only the first response per turn is a reply
	//   "all"   — every response is a reply (Hermes default, Gormes default)
	//   "off"   — no outbound message is threaded as a reply
	ReplyMode string
	// PersistToolProgressMode saves /verbose mode changes. Production writes
	// display.platforms.<platform>.tool_progress into Gormes config.toml.
	PersistToolProgressMode func(platform, mode string) error
	// ToolProgressModes mirrors Hermes display.platforms.<platform>.tool_progress
	// overrides. Values take precedence over ToolProgressMode for the named platform.
	ToolProgressModes map[string]string
	SessionMap        session.Map
	// SessionReset controls automatic session clearing on inactivity or daily
	// boundary. Mirrors Hermes session_reset config section.
	SessionResetPolicy      string // "inactivity", "daily", "both", "none" (default: "inactivity")
	SessionResetIdleMinutes int    // inactivity timeout (default: 1440 = 24h)
	SessionResetDailyHour   int    // daily reset hour 0-23 (default: 4)
	// AgentRouting enables OpenClaw-style agent/workspace bindings for live
	// gateway turns. Zero value preserves legacy single-agent chat keys.
	AgentRouting AgentRoutingConfig
	// ToolRegistry is the full process registry. Agent-routed turns receive a
	// policy-filtered view of this registry when agents.list[].tools is set.
	ToolRegistry *tools.Registry
	// SkillRuntime is the full process skills runtime. Agent-routed turns
	// receive an allowlist wrapper when agents.list[].skills is set.
	SkillRuntime *skills.Runtime
	// SkillsCommandOptions carries local /skills command dependencies such as
	// configured roots and direct-URL install seams. Nil/zero dependencies keep
	// unavailable evidence instead of submitting /skills text to the model.
	SkillsCommandOptions SkillsCommandOptions
	// AgentRuntimeFactory optionally returns an independent kernel/runtime for
	// the routed agent session. When nil, Manager falls back to the legacy
	// singleton kernel with per-turn policy overrides.
	AgentRuntimeFactory AgentRuntimeFactory
	Hooks               *Hooks
	// EventDispatcher mirrors successful outbound sends onto the shared event
	// bus after channel delivery succeeds. Nil keeps legacy send behavior.
	EventDispatcher *EventDispatcher
	RuntimeStatus   RuntimeStatusWriter
	Restart         RestartConfig
	// RestartNotifications holds per-platform restart comeback notification
	// overrides. Missing platforms default to enabled.
	RestartNotifications map[string]bool
	SessionExpiry        SessionExpiryConfig
	Now                  func() time.Time
	// SkipAutoResume disables gateway startup auto-resume of sessions
	// marked ResumePending. Tests use this flag to isolate ResumePending
	// flag handling from auto-resume scheduling.
	SkipAutoResume bool
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
	// ImageInputMode honors agent.image_input_mode for channel-attached images.
	// Empty is Hermes auto mode.
	ImageInputMode llm.ImageInputMode
	// AuxiliaryVision mirrors auxiliary.vision. Any explicit provider/model/base
	// URL routes auto image input mode through text fallback instead of native
	// image content parts.
	AuxiliaryVision llm.AuxiliaryVisionConfig
	// TitleModel is the provider boundary for auto-title generation. It is
	// called at most once per PhaseIdle frame for sessions without an existing
	// title. Nil disables the LLM call; PerformAutoTitle surfaces
	// AutoTitleCodeProviderFailed evidence through AuxiliaryFailureSink.
	TitleModel llm.TitleModelFunc
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
	// KanbanDispatcher owns the gateway-managed Kanban worker dispatcher loop.
	// Nil preserves the legacy gateway behavior with no dispatcher activity.
	KanbanDispatcher KanbanDispatcherConfig
	// KanbanSlashRunner handles gateway /kanban through the same command
	// implementation used by the local CLI/TUI. Nil consumes /kanban with
	// unavailable evidence instead of submitting it to the model.
	KanbanSlashRunner KanbanSlashRunner
	// SlashConfirmations stores confirmable slash-command prompts by session.
	// Reset boundaries clear only the target session's pending confirmation.
	SlashConfirmations *SlashConfirmationQueue
	// ReloadConfig returns a freshly loaded manager config for reloadable
	// runtime fields. Errors keep the last-good manager config active.
	ReloadConfig func(context.Context) (ManagerConfig, error)
	// GoalJudge is the optional auxiliary completion judge for the persistent
	// /goal loop. Nil deliberately fails open to continuation; GoalMaxTurns is
	// the hard backstop.
	GoalJudge    GoalJudge
	GoalMaxTurns int
	// TelegramTopicStore owns Telegram private-chat topic-mode state. Nil keeps
	// topic mutations unavailable while still allowing /topic help to render.
	TelegramTopicStore TelegramTopicStore
	// TelegramTopicCapabilities checks BotFather/private-chat topic settings.
	// Nil treats capabilities as unchecked and lets store-backed activation
	// continue.
	TelegramTopicCapabilities TelegramTopicCapabilitiesFunc
	// DynamicAgentRegistry persists runtime-spawned agents and bindings for
	// channel-native /spawn flows. Nil keeps /spawn unavailable.
	DynamicAgentRegistry SpawnAgentRegistry
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

	turnMu             sync.Mutex
	turnPlatform       string
	turnChatID         string
	turnMsgID          string
	turnSessionKey     string
	turnSessionID      string
	turnSource         SessionSource
	turnCancelled      bool
	turnFrameSeen      bool
	turnLastUserText   string // captures the last inbound submit text for auto-title
	turnAudioRequested bool
	turnKernel         KernelSubmitter
	turnReplySent      bool // tracks first reply for ReplyMode "first"
	kernelSessionKey   string
	shuttingDown       bool
	followUps          []InboundEvent
	lastUsageFrame     kernel.RenderFrame

	reasoningDispatcher *ReasoningDispatcher
	ttsConfigStore      *TTSConfigStore

	inboundDedup *MessageDeduplicator

	renderChan <-chan kernel.RenderFrame

	liveTurnPromptSeams liveTurnPromptSeams
	agentRouter         AgentRouter
	agentRoutingEnabled bool
	agentRuntimeMu      sync.Mutex
	agentRuntimes       map[string]KernelSubmitter
	agentRuntimeRender  chan kernel.RenderFrame

	typingActionMu   sync.Mutex
	typingActionLast map[string]time.Time

	toolProgressMu      sync.Mutex
	toolProgressMsgID   string
	toolProgressText    string
	toolProgressChatID  string
	toolProgressPlat    string
	toolProgressSeenIDs map[string]bool

	verboseHintMu   sync.Mutex
	verboseHintSent map[string]bool

	personalityPrompts    map[string]string
	activePersonalityName string

	kanbanDispatcherMu      sync.Mutex
	kanbanDispatcherRunning bool

	telegramTopicMu             sync.Mutex
	telegramTopicCapabilityHint map[string]time.Time

	modelPickerResolver ModelPickerResolver
	modelOverride       SessionModelOverride
}

type channelRunFailure struct {
	channel Channel
	err     error
}

type hookedPlaceholderEditor struct {
	base         placeholderEditor
	manager      *Manager
	platform     string
	threadID     string
	replyToMsgID string
}

func (h hookedPlaceholderEditor) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	const placeholderText = "⏳"

	h.manager.fireHook(ctx, HookEvent{
		Point:            HookBeforeSend,
		Platform:         h.platform,
		ChatID:           chatID,
		ThreadID:         h.threadID,
		ReplyToMessageID: h.replyToMsgID,
		Text:             placeholderText,
	})

	var (
		msgID string
		err   error
	)
	if h.replyToMsgID != "" {
		if h.threadID != "" {
			if replySender, ok := h.base.(ThreadReplyPlaceholderCapable); ok {
				msgID, err = replySender.SendThreadReplyPlaceholder(ctx, chatID, h.threadID, h.replyToMsgID)
			} else if placeholder, ok := h.base.(ThreadPlaceholderCapable); ok {
				msgID, err = placeholder.SendThreadPlaceholder(ctx, chatID, h.threadID)
			} else if replySender, ok := h.base.(ReplyPlaceholderCapable); ok {
				msgID, err = replySender.SendReplyPlaceholder(ctx, chatID, h.replyToMsgID)
			} else {
				msgID, err = h.base.SendPlaceholder(ctx, chatID)
			}
		} else if replySender, ok := h.base.(ReplyPlaceholderCapable); ok {
			msgID, err = replySender.SendReplyPlaceholder(ctx, chatID, h.replyToMsgID)
		} else {
			msgID, err = h.base.SendPlaceholder(ctx, chatID)
		}
	} else if h.threadID != "" {
		if placeholder, ok := h.base.(ThreadPlaceholderCapable); ok {
			msgID, err = placeholder.SendThreadPlaceholder(ctx, chatID, h.threadID)
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
			Point:            HookOnError,
			Platform:         h.platform,
			ChatID:           chatID,
			ThreadID:         h.threadID,
			ReplyToMessageID: h.replyToMsgID,
			Text:             placeholderText,
			Err:              err,
		})
		return "", err
	}

	h.manager.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      h.platform,
		PlatformState: PlatformStateRunning,
	})
	h.manager.fireHook(ctx, HookEvent{
		Point:            HookAfterSend,
		Platform:         h.platform,
		ChatID:           chatID,
		ThreadID:         h.threadID,
		MsgID:            msgID,
		ReplyToMessageID: h.replyToMsgID,
		Text:             placeholderText,
	})
	return msgID, nil
}

func (h hookedPlaceholderEditor) Send(ctx context.Context, chatID, text string) (string, error) {
	sender, ok := h.base.(coalescerMessageSender)
	if !ok {
		return "", errors.New("gateway: channel does not support fresh final send")
	}

	h.manager.fireHook(ctx, HookEvent{
		Point:            HookBeforeSend,
		Platform:         h.platform,
		ChatID:           chatID,
		ThreadID:         h.threadID,
		ReplyToMessageID: h.replyToMsgID,
		Text:             text,
	})

	var (
		msgID string
		err   error
	)
	if h.replyToMsgID != "" {
		if h.threadID != "" {
			if replySender, ok := h.base.(ThreadReplySender); ok {
				msgID, err = replySender.SendThreadReply(ctx, chatID, h.threadID, h.replyToMsgID, text)
			} else if threadSender, ok := h.base.(ThreadSender); ok {
				msgID, err = threadSender.SendThread(ctx, chatID, h.threadID, text)
			} else if replySender, ok := h.base.(ReplySender); ok {
				msgID, err = replySender.SendReply(ctx, chatID, h.replyToMsgID, text)
			} else {
				msgID, err = sender.Send(ctx, chatID, text)
			}
		} else if replySender, ok := h.base.(ReplySender); ok {
			msgID, err = replySender.SendReply(ctx, chatID, h.replyToMsgID, text)
		} else {
			msgID, err = sender.Send(ctx, chatID, text)
		}
	} else if h.threadID != "" {
		if threadSender, ok := h.base.(ThreadSender); ok {
			msgID, err = threadSender.SendThread(ctx, chatID, h.threadID, text)
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
			Point:            HookOnError,
			Platform:         h.platform,
			ChatID:           chatID,
			ThreadID:         h.threadID,
			ReplyToMessageID: h.replyToMsgID,
			Text:             text,
			Err:              err,
		})
		return "", err
	}

	h.manager.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      h.platform,
		PlatformState: PlatformStateRunning,
	})
	h.manager.fireHook(ctx, HookEvent{
		Point:            HookAfterSend,
		Platform:         h.platform,
		ChatID:           chatID,
		ThreadID:         h.threadID,
		MsgID:            msgID,
		ReplyToMessageID: h.replyToMsgID,
		Text:             text,
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

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	if in == nil {
		return nil
	}
	out := make(map[string]bool, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneNestedBoolMap(in map[string]map[string]bool) map[string]map[string]bool {
	if in == nil {
		return nil
	}
	out := make(map[string]map[string]bool, len(in))
	for key, nested := range in {
		out[key] = cloneBoolMap(nested)
	}
	return out
}

func cloneWhitelistConfigMap(in map[string]WhitelistConfig) map[string]WhitelistConfig {
	if in == nil {
		return nil
	}
	out := make(map[string]WhitelistConfig, len(in))
	for key, value := range in {
		value.IDs = append([]string(nil), value.IDs...)
		out[key] = value
	}
	return out
}

func cloneSkillsCommandOptions(in SkillsCommandOptions) SkillsCommandOptions {
	in.ExternalDirs = append([]string(nil), in.ExternalDirs...)
	in.HubProviders = append([]skills.HubRegistryProvider(nil), in.HubProviders...)
	if in.Disabled != nil {
		disabled := make(map[string]struct{}, len(in.Disabled))
		for key, value := range in.Disabled {
			disabled[key] = value
		}
		in.Disabled = disabled
	}
	return in
}

func cloneAgentRoutingConfig(in AgentRoutingConfig) AgentRoutingConfig {
	in.Agents.Defaults.Workspaces = append([]string(nil), in.Agents.Defaults.Workspaces...)
	in.Agents.Defaults.Channels = append([]string(nil), in.Agents.Defaults.Channels...)
	in.Agents.Defaults.Skills = append([]string(nil), in.Agents.Defaults.Skills...)
	in.Agents.List = append([]config.AgentCfg(nil), in.Agents.List...)
	for i := range in.Agents.List {
		in.Agents.List[i].Skills = append([]string(nil), in.Agents.List[i].Skills...)
		in.Agents.List[i].Tools.Allow = append([]string(nil), in.Agents.List[i].Tools.Allow...)
		in.Agents.List[i].Tools.Deny = append([]string(nil), in.Agents.List[i].Tools.Deny...)
		in.Agents.List[i].GroupChat.MentionPatterns = append([]string(nil), in.Agents.List[i].GroupChat.MentionPatterns...)
	}
	in.Bindings = append([]config.AgentBindingCfg(nil), in.Bindings...)
	for i := range in.Bindings {
		in.Bindings[i].Match.Roles = append([]string(nil), in.Bindings[i].Match.Roles...)
	}
	return in
}

func newManagerInternal(cfg ManagerConfig, k kernelSubmitter, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	cfg.AllowedChats = cloneStringMap(cfg.AllowedChats)
	cfg.AllowedUsers = cloneNestedBoolMap(cfg.AllowedUsers)
	cfg.AllowedChatWhitelists = cloneWhitelistConfigMap(cfg.AllowedChatWhitelists)
	cfg.AllowDiscovery = cloneBoolMap(cfg.AllowDiscovery)
	cfg.ToolProgressModes = cloneStringMap(cfg.ToolProgressModes)
	cfg.RestartNotifications = cloneBoolMap(cfg.RestartNotifications)
	cfg.SkillsCommandOptions = cloneSkillsCommandOptions(cfg.SkillsCommandOptions)
	cfg.AgentRouting = cloneAgentRoutingConfig(cfg.AgentRouting)
	if cfg.CoalesceMs <= 0 {
		cfg.CoalesceMs = 1000
	}
	if cfg.AllowedChats == nil {
		cfg.AllowedChats = map[string]string{}
	}
	if cfg.AllowedUsers == nil {
		cfg.AllowedUsers = map[string]map[string]bool{}
	}
	if cfg.BusyInputMode == "" {
		cfg.BusyInputMode = "interrupt"
	}
	if cfg.AllowDiscovery == nil {
		cfg.AllowDiscovery = map[string]bool{}
	}
	if cfg.SlashConfirmations == nil {
		cfg.SlashConfirmations = NewSlashConfirmationQueue()
	}
	seams := defaultLiveTurnPromptSeams()
	explicitProfile := strings.TrimSpace(cfg.ContextFilesProfile) != ""
	if dir := strings.TrimSpace(cfg.ContextFilesProfile); dir != "" {
		seams.ProfileDir = func() string { return dir }
	}
	if cwd := strings.TrimSpace(cfg.ContextFilesCWD); cwd != "" {
		seams.CWD = func() string { return cwd }
		if !explicitProfile {
			seams.ProfileDir = func() string { return defaultLiveTurnProfileDir(cwd) }
		}
		if strings.TrimSpace(cfg.ContextFilesMemoryDir) == "" {
			seams.MemoryDir = func() string { return defaultLiveTurnMemoryDir(cwd) }
		}
	}
	if memDir := strings.TrimSpace(cfg.ContextFilesMemoryDir); memDir != "" {
		seams.MemoryDir = func() string { return memDir }
	} else if explicitProfile {
		// Tests and callers that provide explicit profile fixtures without a
		// memory fixture expect durable USER.md/MEMORY.md context to be absent.
		// Production gateway wiring supplies only CWD and uses workspace-based
		// memory discovery above.
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
		cfg:                         cfg,
		kernel:                      k,
		log:                         log,
		channels:                    map[string]Channel{},
		reasoningDispatcher:         NewReasoningDispatcher(cfg.PersistReasoningGlobal),
		ttsConfigStore:              NewTTSConfigStore(),
		inboundDedup:                NewMessageDeduplicator(defaultInboundDedupMaxSize),
		liveTurnPromptSeams:         seams,
		agentRouter:                 NewAgentRouter(cfg.AgentRouting.Agents, cfg.AgentRouting.Bindings),
		agentRoutingEnabled:         cfg.AgentRouting.Enabled,
		agentRuntimes:               map[string]KernelSubmitter{},
		agentRuntimeRender:          make(chan kernel.RenderFrame, kernel.RenderMailboxCap),
		typingActionLast:            map[string]time.Time{},
		telegramTopicCapabilityHint: map[string]time.Time{},
		modelPickerResolver:         NewModelPickerResolver(&SessionModelOverride{}),
		modelOverride:               SessionModelOverride{},
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
	if m.reasoningDispatcher == nil {
		m.reasoningDispatcher = NewReasoningDispatcher(m.cfg.PersistReasoningGlobal)
	}
	return m.reasoningDispatcher.Dispatch(sessionKey, args)
}

func (m *Manager) clearSessionBoundaryControlState(sessionKey string) {
	if m == nil || m.cfg.SlashConfirmations == nil {
		return
	}
	m.cfg.SlashConfirmations.ClearSlashConfirmationSession(sessionKey)
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
	m.startKanbanDispatcher(runCtx, &wg)

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
	if !m.cfg.SkipAutoResume {
		go m.autoResumePendingSessions(runCtx, inbox)
	}

	activeChannels := len(channels)
	var firstFailure error
	for {
		select {
		case <-ctx.Done():
			cancel()
			m.safeChannelDisconnectAll(context.Background(), channels, "during shutdown")
			m.waitForChannelWorkers(&wg, DefaultChannelDisconnectTimeoutFromEnv(), "shutdown")
			zero := 0
			m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
				GatewayState: GatewayStateStopped,
				ActiveAgents: &zero,
			})
			return nil
		case failure := <-failures:
			m.safeChannelDisconnect(ctx, failure.channel, "after failed startup")
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

func (m *Manager) safeChannelDisconnectAll(ctx context.Context, channels []Channel, scope string) {
	for _, ch := range channels {
		m.safeChannelDisconnect(ctx, ch, scope)
	}
}

func (m *Manager) safeChannelDisconnect(ctx context.Context, ch Channel, scope string) {
	if disconnecter, ok := ch.(DisconnectCapable); ok {
		timeout := DefaultChannelDisconnectTimeoutFromEnv()
		if timeout <= 0 {
			if err := disconnecter.Disconnect(ctx); err != nil {
				m.log.Debug("defensive channel disconnect "+scope+" raised", "channel", ch.Name(), "err", err)
			}
			return
		}

		disconnectCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			done <- disconnecter.Disconnect(disconnectCtx)
		}()

		select {
		case err := <-done:
			if err != nil {
				m.log.Debug("defensive channel disconnect "+scope+" raised", "channel", ch.Name(), "err", err)
			}
		case <-disconnectCtx.Done():
			if errors.Is(disconnectCtx.Err(), context.DeadlineExceeded) {
				m.log.Warn("defensive channel disconnect "+scope+" timed out", "channel", ch.Name(), "timeout", timeout)
				return
			}
			m.log.Debug("defensive channel disconnect "+scope+" raised", "channel", ch.Name(), "err", disconnectCtx.Err())
		}
	}
}

func (m *Manager) waitForChannelWorkers(wg *sync.WaitGroup, timeout time.Duration, scope string) {
	if timeout <= 0 {
		wg.Wait()
		return
	}
	done := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		close(done)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		m.log.Warn("gateway channel workers did not stop before timeout", "scope", scope, "timeout", timeout)
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

	if handled, err := m.dispatchGatewayCommandEvent(ctx, ch, ev); handled {
		return err
	}

	switch ev.Kind {
	case EventSubmit:
		m.handleSubmitEvent(ctx, ch, ev)
		return nil
	case EventUnknown:
		unknownText := strings.TrimSpace(ev.Text)
		if strings.HasPrefix(unknownText, "/") || strings.HasPrefix(unknownText, "／") {
			ev.Kind = EventSubmit
			m.handleSubmitEvent(ctx, ch, ev)
			return nil
		}
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "unknown command")
		return nil
	}
	return nil
}

func (m *Manager) dispatchCommandEvent(ctx context.Context, ch Channel, ev InboundEvent) bool {
	handled, _ := m.dispatchGatewayCommandEvent(ctx, ch, ev)
	if handled {
		return true
	}
	if ev.Kind == EventUnknown {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "unknown command")
		return true
	}
	return false
}

func (m *Manager) handleReasoningCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	reply, err := m.DispatchReasoning(m.sessionKeyForInbound(ev), commandArgs(ev.Text))
	if err != nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Reasoning command error: "+reasoningCommandErrorText(err)+"\n\nUsage: /reasoning [low|medium|high|reset|show] [--global]")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, formatReasoningReply(reply))
}

func reasoningCommandErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}
	replacer := strings.NewReplacer("`", "'", "*", "'", "#", "＃")
	return strings.Join(strings.Fields(replacer.Replace(msg)), " ")
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
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "The `/verbose` command is not enabled for messaging platforms.\n\nEnable it in Gormes `config.toml`:\n```toml\n[display]\ntool_progress_command = true\n```")
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
		text += "\n_(could not save to config: " + verboseCommandErrorText(err) + ")_"
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
		return
	}
	text += "\n_(saved for **" + platform + "** — takes effect on next message)_"
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
}

func verboseCommandErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	compact := compactVerboseSecretSeparators(lower)
	for _, marker := range []string{"token", "api_key", "apikey", "authorization", "bearer", "secret", "password"} {
		if strings.Contains(lower, marker) || strings.Contains(compact, marker) {
			return "[redacted]"
		}
	}
	replacer := strings.NewReplacer("`", "'", "*", "'", "#", "＃")
	return strings.Join(strings.Fields(replacer.Replace(msg)), " ")
}

func compactVerboseSecretSeparators(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (m *Manager) handleSessionsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	if m.cfg.SessionMap == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Sessions are not available in this build.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "📋 Use `/status` for details on the current session. Use `/new` to start fresh.")
}

func (m *Manager) handleModelCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	isTelegram := strings.HasPrefix(ch.Name(), "telegram")
	if m.modelPickerResolver != nil && isTelegram {
		resp, err := m.modelPickerResolver.OpenModelPicker(ctx, ModelPickerRequest{ChatID: ev.ChatID})
		if err == nil {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, resp.Text)
			return
		}
	}
	model := "unknown"
	provider := "unknown"
	over := m.modelOverride
	if over.Model != "" {
		model = over.Model
	} else if m.cfg.LiveTurnActiveModel != nil {
		model = m.cfg.LiveTurnActiveModel()
	}
	if over.Provider != "" {
		provider = over.Provider
	} else if m.cfg.LiveTurnActiveProvider != nil {
		provider = m.cfg.LiveTurnActiveProvider()
	}
	if model == "" {
		model = "unknown"
	}
	if provider == "" {
		provider = "unknown"
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("🤖 **Model:** `%s`\n📡 **Provider:** `%s`", modelCommandFieldText(model), modelCommandFieldText(provider)))
}

func modelCommandFieldText(value string) string {
	msg := strings.TrimSpace(value)
	if msg == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("`", "'", "*", "'", "#", "＃")
	return strings.Join(strings.Fields(replacer.Replace(msg)), " ")
}

func (m *Manager) handleProfileCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	home := config.GormesHome()
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("👤 **Profile:** `(default)`\n📂 **Home:** `%s`", modelCommandFieldText(home)))
}

func (m *Manager) handlePlatformsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	platforms := m.formatConnectedPlatforms()
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("📡 **Connected Platforms:** %s\nUse `/status` for full session details.", platforms))
}

// handlePlatformControlCommand is the gateway slash-handler port of Hermes
// gateway/run.py:_handle_platform_command (PR #26600): `/platform
// <list|pause|resume> [name]`. The shared platform reconnect/circuit-breaker
// queue is a tested lifecycle seam not yet wired into the live manager (see
// the "Gateway platform reconnect isolation" row's deferred integration), so
// the live failed-platform set is currently empty and pause/resume on a
// non-queued platform truthfully reports it is not in the retry queue.
func (m *Manager) handlePlatformControlCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	m.mu.Lock()
	connected := make(map[string]Channel, len(m.channels))
	for name, channel := range m.channels {
		connected[name] = channel
	}
	m.mu.Unlock()
	// No live failed-platform set is wired into the manager yet; the deferred
	// lifecycle-integration row will pass the real queue here.
	reply := HandlePlatformCommand(ev.Text, connected, nil)
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, reply)
}

func (m *Manager) formatConnectedPlatforms() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, modelCommandFieldText(name))
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

func (m *Manager) handleQueueCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	text := strings.TrimSpace(strings.Join(commandArgs(ev.Text), " "))
	if text == "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Usage: /queue <prompt>")
		return
	}
	followUp := ev
	followUp.Kind = EventSubmit
	followUp.Text = text
	followUp.Attachments = nil
	queued, full := m.queueFollowUpIfActive(followUp)
	if full {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, followUpQueueFullNotice)
		return
	}
	if !queued {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "queue_unavailable: no active turn; send the prompt without /queue to run it now")
		return
	}
	depth := m.followUpQueueDepth()
	if depth <= 1 {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Queued for the next turn.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("Queued for the next turn. (%d queued)", depth))
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

func (m *Manager) replyTargetForTurn(msgID string) string {
	replyMode := strings.ToLower(strings.TrimSpace(m.cfg.ReplyMode))
	switch replyMode {
	case "off":
		return ""
	case "first":
		if m.turnReplySent {
			return ""
		}
		m.turnReplySent = true
		return msgID
	default:
		return msgID
	}
}

func (m *Manager) dispatchFrame(ctx context.Context, f kernel.RenderFrame, co **coalescer, coCancel *context.CancelFunc) {
	m.rememberUsageFrame(f)
	m.turnMu.Lock()
	platform := m.turnPlatform
	chatID := m.turnChatID
	msgID := m.turnMsgID
	threadID := strings.TrimSpace(m.turnSource.ThreadID)
	replyToMsgID := m.replyTargetForTurn(msgID)
	sessionKey := m.turnSessionKey
	sessionID := m.turnSessionID
	lastUserText := m.turnLastUserText
	audioRequested := m.turnAudioRequested
	cancelled := m.turnCancelled
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
	m.maybeSendTypingAction(ctx, ch, f.Phase, chatID, threadID)
	m.dispatchToolProgress(ctx, ch, platform, chatID, threadID, msgID, f)
	pe, ok := ch.(placeholderEditor)
	if !ok {
		if m.sendNoEdit(ctx, ch, f, chatID, replyToMsgID, threadID, sessionKey, audioRequested) {
			m.completeProcessingReaction(ctx, ch, processingOutcomeForFrame(f.Phase, cancelled))
			if f.Phase == kernel.PhaseIdle {
				m.maybeRunAutoTitle(ctx, f, sessionID, lastUserText)
				m.handleGoalPostTurnContinuation(ctx, ch, f)
			} else if f.Phase == kernel.PhaseFailed || f.Phase == kernel.PhaseCancelling {
				m.pauseInterruptedGoal(ctx, ch, activeTurnSnapshot{
					Platform:  platform,
					ChatID:    chatID,
					MsgID:     msgID,
					SessionID: sessionID,
					Cancelled: cancelled,
				})
			}
			m.drainNextFollowUp(ctx)
		}
		return
	}

	switch f.Phase {
	case kernel.PhaseIdle:
		finalPages, media := m.formatFinalDeliveryPagesForTurn(ctx, platform, f, sessionKey, audioRequested)
		if *co != nil {
			(*co).flushImmediateFinal(ctx, finalPages[0], true)
			(*coCancel)()
			*co = nil
			*coCancel = nil
			m.sendRemainingFinalPages(ctx, ch, chatID, threadID, replyToMsgID, finalPages[1:])
		} else {
			m.sendFinalPages(ctx, ch, chatID, threadID, "", finalPages)
		}
		m.deliverMedia(ctx, ch, chatID, replyToMsgID, threadID, media)
		m.maybeRunAutoTitle(ctx, f, sessionID, lastUserText)
		m.maybeSendVerboseHint(ctx, ch, platform, chatID, f)
		m.clearToolProgress()
		m.completeProcessingReaction(ctx, ch, processingOutcomeForFrame(f.Phase, cancelled))
		m.handleGoalPostTurnContinuation(ctx, ch, f)
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
		m.completeProcessingReaction(ctx, ch, processingOutcomeForFrame(f.Phase, cancelled))
		m.pauseInterruptedGoal(ctx, ch, activeTurnSnapshot{
			Platform:  platform,
			ChatID:    chatID,
			MsgID:     msgID,
			SessionID: sessionID,
			Cancelled: cancelled,
		})
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
				threadID:     threadID,
				replyToMsgID: replyToMsgID,
			}, time.Duration(m.cfg.CoalesceMs)*time.Millisecond, chatID, opts...)
			*co = nc
			go nc.run(cCtx)
		}
		(*co).setPending(text)
	}
}

func (m *Manager) dispatchToolProgress(ctx context.Context, ch Channel, platform, chatID, threadID, requestID string, f kernel.RenderFrame) {
	if sender, ok := ch.(ToolProgressSender); ok {
		events := FormatToolProgressEvents(f, m.toolProgressMode(platform), requestID)
		if len(events) == 0 {
			return
		}
		m.toolProgressMu.Lock()
		sameTarget := m.toolProgressPlat == platform && m.toolProgressChatID == chatID
		if !sameTarget || m.toolProgressSeenIDs == nil {
			m.toolProgressSeenIDs = map[string]bool{}
		}
		for i := range events {
			if events[i].Status != ToolProgressStarted {
				continue
			}
			if m.toolProgressSeenIDs[events[i].ID] {
				events[i].Status = ToolProgressUpdated
				events[i].Summary = toolProgressSummary(events[i].ToolName, ToolProgressUpdated)
				continue
			}
			m.toolProgressSeenIDs[events[i].ID] = true
		}
		fingerprint := toolProgressEventsFingerprint(events)
		if sameTarget && m.toolProgressText == fingerprint {
			m.toolProgressMu.Unlock()
			return
		}
		m.toolProgressMu.Unlock()

		for _, event := range events {
			_, _ = sender.SendToolProgress(ctx, chatID, event)
		}
		m.toolProgressMu.Lock()
		m.toolProgressPlat = platform
		m.toolProgressChatID = chatID
		m.toolProgressMsgID = events[len(events)-1].ID
		m.toolProgressText = fingerprint
		m.toolProgressMu.Unlock()
		return
	}

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

	newMsgID, err := m.sendWithHooksThread(ctx, ch, chatID, threadID, text)
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

func (m *Manager) sendNoEdit(ctx context.Context, ch Channel, f kernel.RenderFrame, chatID, replyToMsgID, threadID, sessionKey string, audioRequested bool) bool {
	switch f.Phase {
	case kernel.PhaseIdle:
		finalPages, media := m.formatFinalDeliveryPagesForTurn(ctx, ch.Name(), f, sessionKey, audioRequested)
		m.sendFinalPages(ctx, ch, chatID, threadID, replyToMsgID, finalPages)
		m.deliverMedia(ctx, ch, chatID, replyToMsgID, threadID, media)
		return true
	case kernel.PhaseFailed, kernel.PhaseCancelling:
		_, _ = m.sendWithHooksReplyThread(ctx, ch, chatID, threadID, replyToMsgID, m.formatError(ch.Name(), f))
		return true
	case kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseReconnecting, kernel.PhaseFinalizing:
		if text := m.formatStream(ch.Name(), f); text != "" {
			_, _ = m.sendWithHooksReplyThread(ctx, ch, chatID, threadID, replyToMsgID, text)
		}
	}
	return false
}

func (m *Manager) sendWithHooks(ctx context.Context, ch Channel, chatID, text string) (string, error) {
	return m.sendWithHooksReplyThread(ctx, ch, chatID, "", "", text)
}

func (m *Manager) sendWithHooksThread(ctx context.Context, ch Channel, chatID, threadID, text string) (string, error) {
	return m.sendWithHooksReplyThread(ctx, ch, chatID, threadID, "", text)
}

func (m *Manager) sendFinalPages(ctx context.Context, ch Channel, chatID, threadID, replyToMsgID string, pages []string) {
	m.sendFinalPagesWithReplyPolicy(ctx, ch, chatID, threadID, replyToMsgID, pages, true)
}

func (m *Manager) sendRemainingFinalPages(ctx context.Context, ch Channel, chatID, threadID, replyToMsgID string, pages []string) {
	m.sendFinalPagesWithReplyPolicy(ctx, ch, chatID, threadID, replyToMsgID, pages, false)
}

func (m *Manager) sendFinalPagesWithReplyPolicy(ctx context.Context, ch Channel, chatID, threadID, replyToMsgID string, pages []string, replyFirstPage bool) {
	replyEveryPage := telegramDMTopicReplyFallbackLane(ch.Name(), chatID, threadID) && strings.TrimSpace(replyToMsgID) != ""
	for i, page := range pages {
		if replyEveryPage || (replyFirstPage && i == 0) {
			_, _ = m.sendWithHooksReplyThread(ctx, ch, chatID, threadID, replyToMsgID, page)
			continue
		}
		_, _ = m.sendWithHooksThread(ctx, ch, chatID, threadID, page)
	}
}

func (m *Manager) sendWithHooksReply(ctx context.Context, ch Channel, chatID, replyToMsgID, text string) (string, error) {
	return m.sendWithHooksReplyThread(ctx, ch, chatID, "", replyToMsgID, text)
}

func (m *Manager) sendWithHooksReplyThread(ctx context.Context, ch Channel, chatID, threadID, replyToMsgID, text string) (string, error) {
	if ch == nil {
		return "", nil
	}
	if telegramDMTopicReplyFallbackLane(ch.Name(), chatID, threadID) && strings.TrimSpace(replyToMsgID) == "" {
		threadID = ""
	}
	ev := HookEvent{
		Point:            HookBeforeSend,
		Platform:         ch.Name(),
		ChatID:           chatID,
		ThreadID:         threadID,
		ReplyToMessageID: replyToMsgID,
		Text:             text,
	}
	m.fireHook(ctx, ev)

	var (
		msgID string
		err   error
	)
	if replyToMsgID != "" {
		if threadID != "" {
			if replySender, ok := ch.(ThreadReplySender); ok {
				msgID, err = replySender.SendThreadReply(ctx, chatID, threadID, replyToMsgID, text)
			} else if threadSender, ok := ch.(ThreadSender); ok {
				msgID, err = threadSender.SendThread(ctx, chatID, threadID, text)
			} else if replySender, ok := ch.(ReplySender); ok {
				msgID, err = replySender.SendReply(ctx, chatID, replyToMsgID, text)
			} else {
				msgID, err = ch.Send(ctx, chatID, text)
			}
		} else if replySender, ok := ch.(ReplySender); ok {
			msgID, err = replySender.SendReply(ctx, chatID, replyToMsgID, text)
		} else {
			msgID, err = ch.Send(ctx, chatID, text)
		}
	} else if threadID != "" {
		if threadSender, ok := ch.(ThreadSender); ok {
			msgID, err = threadSender.SendThread(ctx, chatID, threadID, text)
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
			Point:            HookOnError,
			Platform:         ch.Name(),
			ChatID:           chatID,
			ThreadID:         threadID,
			ReplyToMessageID: replyToMsgID,
			Text:             text,
			Err:              err,
		})
		return "", err
	}

	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      ch.Name(),
		PlatformState: PlatformStateRunning,
	})
	m.fireHook(ctx, HookEvent{
		Point:            HookAfterSend,
		Platform:         ch.Name(),
		ChatID:           chatID,
		ThreadID:         threadID,
		MsgID:            msgID,
		ReplyToMessageID: replyToMsgID,
		Text:             text,
	})
	return msgID, nil
}

func (m *Manager) fireHook(ctx context.Context, ev HookEvent) {
	if m.cfg.Hooks != nil {
		m.cfg.Hooks.Fire(ctx, ev)
	}
	m.publishMessageSentEvent(ev)
}

func (m *Manager) publishMessageSentEvent(ev HookEvent) {
	if ev.Point != HookAfterSend || m.cfg.EventDispatcher == nil || !m.cfg.EventDispatcher.Available() {
		return
	}
	kind := "message"
	if ev.ReplyToMessageID != "" {
		kind = "reply"
	}
	payload := MessageEventPayload{
		Platform:         ev.Platform,
		ChatID:           ev.ChatID,
		ThreadID:         ev.ThreadID,
		MessageID:        ev.MsgID,
		MsgID:            ev.MsgID,
		ReplyToMessageID: ev.ReplyToMessageID,
		Kind:             kind,
		Text:             ev.Text,
		Body:             ev.Text,
	}
	traceID := strings.Join([]string{"gateway", ev.Platform, ev.ChatID, ev.MsgID}, ":")
	if err := m.cfg.EventDispatcher.PublishMessageSent(ev.Platform, traceID, payload); err != nil {
		m.log.Debug("publish gateway message-sent event", "platform", ev.Platform, "chat_id", ev.ChatID, "msg_id", ev.MsgID, "err", err)
	}
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
	marker.SourcePlatform = strings.TrimSpace(marker.SourcePlatform)
	marker.ChatID = strings.TrimSpace(marker.ChatID)
	marker.ThreadID = strings.TrimSpace(marker.ThreadID)
	marker.UpdateID = strings.TrimSpace(marker.UpdateID)
	marker.MessageID = strings.TrimSpace(marker.MessageID)
	ch := m.lookupChannel(marker.SourcePlatform)
	if ch == nil {
		return nil
	}
	evidence := restartTakeoverEvidence(marker, RestartTakeoverMarkerStatusSeen, now)
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{TakeoverMarkerEvidence: &evidence})
	if !m.restartNotificationEnabled(marker.SourcePlatform) {
		m.log.Info("restart notification suppressed", "platform", marker.SourcePlatform, "chat_id", marker.ChatID)
		return store.MarkNotificationSent(ctx, marker, now)
	}
	if _, err := m.sendWithHooks(ctx, ch, marker.ChatID, "Gateway restarted successfully. Your session continues."); err != nil {
		return err
	}
	return store.MarkNotificationSent(ctx, marker, now)
}

func (m *Manager) restartNotificationEnabled(platform string) bool {
	key := normalizedPlatformName(platform)
	if key == "" || len(m.cfg.RestartNotifications) == 0 {
		return true
	}
	if enabled, ok := m.cfg.RestartNotifications[key]; ok {
		return enabled
	}
	base := platformBaseName(key)
	if base != key {
		if enabled, ok := m.cfg.RestartNotifications[base]; ok {
			return enabled
		}
	}
	return true
}

func (m *Manager) allowed(ev InboundEvent) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if isTelegramPlatform(ev.Platform) && ev.AllowlistBypassReason == AllowlistBypassTelegramGuestMention {
		if _, ok := m.cfg.AllowedChatWhitelists[ev.Platform]; ok {
			return true
		}
		if want := strings.TrimSpace(m.cfg.AllowedChats[ev.Platform]); want != "" {
			return true
		}
	}
	if wl, ok := m.cfg.AllowedChatWhitelists[ev.Platform]; ok && !wl.IsAllowed(ev.ChatID) {
		return false
	}
	want, ok := m.cfg.AllowedChats[ev.Platform]
	if ok && want != "" && ev.ChatID == want {
		return true
	}
	if users := m.cfg.AllowedUsers[ev.Platform]; len(users) > 0 {
		if users["*"] {
			return true
		}
		return users[ev.UserID]
	}
	return false
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
	m.turnAudioRequested = false
	m.turnKernel = nil
	m.turnReplySent = false
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
	m.turnAudioRequested = false
	m.turnKernel = nil
	m.resetToolProgress()
}

func (m *Manager) setTurnLastUserText(text string) {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	m.turnLastUserText = text
}

func (m *Manager) setTurnAudioRequested(requested bool) {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	m.turnAudioRequested = requested
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
	if isTelegramPlatform(platform) {
		return FormatStreamTelegram(f)
	}
	return FormatStreamPlain(f)
}

func (m *Manager) formatToolProgress(platform string, f kernel.RenderFrame) string {
	mode := m.toolProgressMode(platform)
	if isTelegramPlatform(platform) {
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
	m.toolProgressSeenIDs = nil
	m.toolProgressMu.Unlock()
}

func toolProgressEventsFingerprint(events []ToolProgressEvent) string {
	if len(events) == 0 {
		return ""
	}
	var b strings.Builder
	for _, event := range events {
		b.WriteString(event.ID)
		b.WriteByte('|')
		b.WriteString(event.ToolName)
		b.WriteByte('|')
		b.WriteString(string(event.Status))
		b.WriteByte('|')
		b.WriteString(event.Summary)
		b.WriteByte('\n')
	}
	return b.String()
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
	key := normalizedPlatformName(platform)
	if key != "" && len(m.cfg.ToolProgressModes) > 0 {
		if mode := strings.TrimSpace(m.cfg.ToolProgressModes[key]); mode != "" {
			return normalizeGatewayToolProgressMode(mode)
		}
		base := platformBaseName(key)
		if base != key {
			if mode := strings.TrimSpace(m.cfg.ToolProgressModes[base]); mode != "" {
				return normalizeGatewayToolProgressMode(mode)
			}
		}
	}
	if mode := strings.TrimSpace(m.cfg.ToolProgressMode); mode != "" {
		return normalizeGatewayToolProgressMode(mode)
	}
	return defaultToolProgressModeForPlatform(key)
}

func (m *Manager) deliverMedia(ctx context.Context, ch Channel, chatID, replyToMsgID, threadID string, media []OutboundMedia) {
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
		if item.ThreadID == "" {
			item.ThreadID = threadID
		}
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
	if isTelegramPlatform(platform) {
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

func (m *Manager) followUpQueueDepth() int {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	return len(m.followUps)
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
	m.turnAudioRequested = false
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

func (m *Manager) checkAutoReset(ctx context.Context, sessionKey string) {
	policy := m.cfg.SessionResetPolicy
	if policy == "" || policy == "none" {
		return
	}
	if m.kernel == nil {
		return
	}
	// Get session metadata to check last activity time.
	sid := sessionKey
	if resolved, err := resolveSession(ctx, m.cfg.SessionMap, sessionKey); err == nil && resolved.SessionID != "" {
		sid = resolved.SessionID
	}
	meta, ok := m.getSessionMetadata(ctx, sid)
	if !ok {
		return
	}

	now := m.now()
	idleMinutes := m.cfg.SessionResetIdleMinutes
	if idleMinutes <= 0 {
		idleMinutes = 1440
	}
	dailyHour := m.cfg.SessionResetDailyHour
	if dailyHour < 0 || dailyHour > 23 {
		dailyHour = 4
	}

	shouldReset := false
	switch policy {
	case "inactivity":
		if meta.UpdatedAt > 0 && now.Sub(time.Unix(meta.UpdatedAt, 0)) > time.Duration(idleMinutes)*time.Minute {
			shouldReset = true
		}
	case "daily":
		if now.Hour() >= dailyHour && meta.UpdatedAt < dayStart(now, dailyHour).Unix() {
			shouldReset = true
		}
	case "both":
		if meta.UpdatedAt > 0 && now.Sub(time.Unix(meta.UpdatedAt, 0)) > time.Duration(idleMinutes)*time.Minute {
			shouldReset = true
		}
		if !shouldReset && now.Hour() >= dailyHour && meta.UpdatedAt < dayStart(now, dailyHour).Unix() {
			shouldReset = true
		}
	}

	if shouldReset {
		m.log.Info("auto-reset session", "key", sessionKey, "policy", policy)
		_ = m.kernel.ResetSession()
	}
}

func dayStart(t time.Time, hour int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), hour, 0, 0, 0, t.Location())
}

func (m *Manager) submitPinned(ctx context.Context, ch Channel, ev InboundEvent) bool {
	route := m.agentRouteForInbound(ev)
	sessionKey := strings.TrimSpace(route.SessionKey)
	m.checkAutoReset(ctx, sessionKey)
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
	audioRequested := inboundRequestsAudioReply(ev)
	m.setTurnAudioRequested(audioRequested)
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
	sessionBlock = appendAudioDeliveryGuidance(sessionBlock, audioRequested)
	sessionBlock = prependChannelPromptBlock(sessionBlock, ev.ChannelPrompt)
	seams := m.liveTurnPromptSeamsForAgent(route)
	sessionContext, _, _ := assembleLiveTurnPrompt(seams, submitText, resolved.SessionID, sessionBlock)
	snapshot := m.agentRuntimeSnapshot(route)
	snapshot = m.applyChannelAutoSkills(route, snapshot, ev.AutoSkills)
	if ev.SkillSlashExpanded {
		snapshot.Skills = noSkillProvider{}
	}
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
				"No provider configured for agent `%s` (agent_runtime_unavailable).\n\n"+
					"To fix this, run in your terminal:\n"+
					"  gormes setup provider\n\n"+
					"Or see the setup guide:\n"+
					"  https://docs.gormes.ai/getting-started/first-run/\n\n"+
					"Config file: %s/config.toml",
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
	imageSubmitText, imageContentParts := m.imageModeSubmitPayload(ev.Text, submitText, ev.Attachments, route)
	if err := submitter.Submit(kernel.PlatformEvent{
		Kind:           kernel.PlatformEventSubmit,
		Text:           imageSubmitText,
		ContentParts:   imageContentParts,
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
	m.startProcessingReaction(ctx, ch, ev)
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

func (m *Manager) imageModeSubmitPayload(userText, submitText string, attachments []Attachment, route agentRuntimeRoute) (string, []llm.MessageContentPart) {
	model := strings.TrimSpace(route.Decision.Model)
	if model == "" && m.cfg.LiveTurnActiveModel != nil {
		model = strings.TrimSpace(m.cfg.LiveTurnActiveModel())
	}
	provider := ""
	if m.cfg.LiveTurnActiveProvider != nil {
		provider = strings.TrimSpace(m.cfg.LiveTurnActiveProvider())
	}
	return imagePayloadFromAttachments(userText, submitText, attachments, imageInputModeOptions{
		Mode:            m.cfg.ImageInputMode,
		AuxiliaryVision: m.cfg.AuxiliaryVision,
		Provider:        provider,
		Model:           model,
	})
}

func gormesHomeHint() string {
	return config.GormesHome()
}
