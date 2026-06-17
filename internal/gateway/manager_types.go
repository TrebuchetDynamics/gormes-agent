package gateway

import (
	"context"
	"log/slog"
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
	// SessionHistoryStore rewrites/replays durable session transcripts for
	// /retry and /undo. Nil leaves those commands unavailable.
	SessionHistoryStore SessionHistoryStore
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

type kernelSessionResumer interface {
	ResumeSession(sessionID string, history []llm.Message) error
}

type kernelManualCompressor interface {
	ManualCompress(focus string) error
}

type SessionHistoryRewindResult struct {
	SessionID    string
	History      []llm.Message
	TargetText   string
	TurnsUndone  int
	RewoundCount int
}

type SessionHistoryStore interface {
	LoadSessionHistory(ctx context.Context, sessionID string) ([]llm.Message, error)
	RewriteSessionHistory(ctx context.Context, sessionID string, history []llm.Message) error
	RewindSessionHistory(ctx context.Context, sessionID string, userTurns int) (SessionHistoryRewindResult, error)
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

	turnMu              sync.Mutex
	turnPlatform        string
	turnChatID          string
	turnMsgID           string
	turnSessionKey      string
	turnSessionID       string
	turnSource          SessionSource
	turnCancelled       bool
	turnFrameSeen       bool
	turnLastUserText    string // captures the last inbound submit text for auto-title
	turnContentDedupKey string
	turnAudioRequested  bool
	turnKernel          KernelSubmitter
	turnReplySent       bool // tracks first reply for ReplyMode "first"
	kernelSessionKey    string
	shuttingDown        bool
	followUps           []InboundEvent
	lastUsageFrame      kernel.RenderFrame

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
