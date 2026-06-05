package events

import eventcommands "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events/commands"

// EventKind is the normalized command kind on an inbound message.
type EventKind = eventcommands.EventKind

const (
	// EventUnknown is an unrecognized slash command.
	EventUnknown = eventcommands.EventUnknown
	// EventSubmit carries user text for kernel.PlatformEventSubmit.
	EventSubmit = eventcommands.EventSubmit
	// EventCancel maps to kernel.PlatformEventCancel.
	EventCancel = eventcommands.EventCancel
	// EventReset maps to kernel.PlatformEventResetSession.
	EventReset = eventcommands.EventReset
	// EventStart is the help or welcome command.
	EventStart = eventcommands.EventStart
	// EventRestart requests a graceful service-manager restart.
	EventRestart = eventcommands.EventRestart
	// EventSteer queues operator guidance for the active turn fallback path.
	EventSteer = eventcommands.EventSteer
	// EventQueue queues one full follow-up turn without interrupting the active run.
	EventQueue = eventcommands.EventQueue
	// EventUsage renders runtime and provider account-usage evidence.
	EventUsage = eventcommands.EventUsage
	// EventStatus renders Hermes-style gateway/session status directly in the channel.
	EventStatus = eventcommands.EventStatus
	// EventTitle sets or reads the current session title directly in the channel.
	EventTitle = eventcommands.EventTitle
	// EventVerbose cycles gateway tool-progress display for the calling platform.
	EventVerbose = eventcommands.EventVerbose
	EventModel   = eventcommands.EventModel
	// EventGateway renders gateway status.
	EventGateway = eventcommands.EventGateway
	// EventThreadLifecycle carries normalized thread open/close/archive state.
	EventThreadLifecycle = eventcommands.EventThreadLifecycle
	// EventSessions handles /sessions subcommands (list, show).
	EventSessions = eventcommands.EventSessions
	// EventProfile handles /profile subcommands (show, list).
	EventProfile = eventcommands.EventProfile
	// EventSkills handles /skills subcommands (list, inspect).
	EventSkills = eventcommands.EventSkills
	// EventCommands handles /commands [page] command and skill catalog.
	EventCommands = eventcommands.EventCommands
	// EventReasoning handles /reasoning subcommands (show, set, reset).
	EventReasoning = eventcommands.EventReasoning
	// EventBusy handles /busy subcommands (queue, steer, interrupt, status).
	EventBusy = eventcommands.EventBusy
	// EventTTS handles /tts subcommands (on, off, speed, voice, engine, language).
	EventTTS = eventcommands.EventTTS
	// EventReload reloads gateway runtime config without restarting the process.
	EventReload = eventcommands.EventReload
	// EventReloadSkills refreshes dynamic skill command catalogs without a model turn.
	EventReloadSkills = eventcommands.EventReloadSkills
	// EventRetry handles /retry (retry the last message by resending to agent).
	EventRetry = eventcommands.EventRetry
	// EventUndo handles /undo (remove the last user/assistant exchange).
	EventUndo = eventcommands.EventUndo
	// EventGoal handles /goal state and continuation loop controls.
	EventGoal = eventcommands.EventGoal
	// EventTopic handles Telegram private-chat topic-mode controls.
	EventTopic = eventcommands.EventTopic
	// EventKanban handles /kanban subcommands.
	EventKanban = eventcommands.EventKanban
	// EventSpawn handles channel-native dynamic agent spawn UX.
	EventSpawn = eventcommands.EventSpawn
	// EventPlatformControl handles `/platform <list|pause|resume> [name]`.
	EventPlatformControl = eventcommands.EventPlatformControl
	// EventPersonality handles `/personality` subcommands (list, switch, none).
	EventPersonality = eventcommands.EventPersonality
)
