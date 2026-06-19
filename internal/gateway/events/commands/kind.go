package commands

// EventKind is the normalized command kind on an inbound message.
type EventKind int

const (
	// EventUnknown is an unrecognized slash command.
	EventUnknown EventKind = iota
	// EventSubmit carries user text for kernel.PlatformEventSubmit.
	EventSubmit
	// EventCancel maps to kernel.PlatformEventCancel.
	EventCancel
	// EventReset maps to kernel.PlatformEventResetSession.
	EventReset
	// EventStart is the help or welcome command.
	EventStart
	// EventRestart requests a graceful service-manager restart.
	EventRestart
	// EventSteer queues operator guidance for the active turn fallback path.
	EventSteer
	// EventQueue queues one full follow-up turn without interrupting the active run.
	EventQueue
	// EventUsage renders runtime and provider account-usage evidence.
	EventUsage
	// EventStatus renders Hermes-style gateway/session status directly in the channel.
	EventStatus
	// EventTitle sets or reads the current session title directly in the channel.
	EventTitle
	// EventVerbose cycles gateway tool-progress display for the calling platform.
	EventVerbose
	EventModel
	// EventGateway renders gateway status.
	EventGateway
	// EventThreadLifecycle carries normalized thread open/close/archive state.
	EventThreadLifecycle
	// EventSessions handles /sessions subcommands (list, show).
	EventSessions
	// EventProfile handles /profile subcommands (show, list).
	EventProfile
	// EventSkills handles /skills subcommands (list, inspect).
	EventSkills
	// EventCommands handles /commands [page] command and skill catalog.
	EventCommands
	// EventReasoning handles /reasoning subcommands (show, set, reset).
	EventReasoning
	// EventBusy handles /busy subcommands (queue, steer, interrupt, status).
	EventBusy
	// EventTTS handles /tts subcommands (on, off, speed, voice, engine, language).
	EventTTS
	// EventReload reloads gateway runtime config without restarting the process.
	EventReload
	// EventReloadSkills refreshes dynamic skill command catalogs without a model turn.
	EventReloadSkills
	// EventRetry handles /retry (retry the last message by resending to agent).
	EventRetry
	// EventUndo handles /undo (remove the last user/assistant exchange).
	EventUndo
	// EventGoal handles /goal state and continuation loop controls.
	EventGoal
	// EventTopic handles Telegram private-chat topic-mode controls.
	EventTopic
	// EventKanban handles /kanban subcommands.
	EventKanban
	// EventSpawn handles channel-native dynamic agent spawn UX.
	EventSpawn
	// EventPlatformControl handles `/platform <list|pause|resume> [name]`.
	EventPlatformControl
	// EventPersonality handles `/personality` subcommands (list, switch, none).
	EventPersonality
	// EventCompress handles explicit session/context compression.
	EventCompress
)

// String returns the stable log/test representation of an EventKind.
func (k EventKind) String() string {
	switch k {
	case EventSubmit:
		return "submit"
	case EventCancel:
		return "cancel"
	case EventReset:
		return "reset"
	case EventStart:
		return "start"
	case EventRestart:
		return "restart"
	case EventSteer:
		return "steer"
	case EventQueue:
		return "queue"
	case EventUsage:
		return "usage"
	case EventStatus:
		return "status"
	case EventTitle:
		return "title"
	case EventVerbose:
		return "verbose"
	case EventModel:
		return "model"
	case EventGateway:
		return "gateway"
	case EventThreadLifecycle:
		return "thread_lifecycle"
	case EventSessions:
		return "sessions"
	case EventProfile:
		return "profile"
	case EventSkills:
		return "skills"
	case EventCommands:
		return "commands"
	case EventReasoning:
		return "reasoning"
	case EventBusy:
		return "busy"
	case EventTTS:
		return "tts"
	case EventReload:
		return "reload"
	case EventReloadSkills:
		return "reload_skills"
	case EventRetry:
		return "retry"
	case EventUndo:
		return "undo"
	case EventGoal:
		return "goal"
	case EventTopic:
		return "topic"
	case EventKanban:
		return "kanban"
	case EventSpawn:
		return "spawn"
	case EventPlatformControl:
		return "platform_control"
	case EventPersonality:
		return "personality"
	case EventCompress:
		return "compress"
	default:
		return "unknown"
	}
}
