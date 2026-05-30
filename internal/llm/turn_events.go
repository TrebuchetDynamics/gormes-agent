package llm

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/turnevents"
)

const (
	TopicTurnStart    = turnevents.TopicTurnStart
	TopicTurnThought  = turnevents.TopicTurnThought
	TopicTurnAction   = turnevents.TopicTurnAction
	TopicTurnObserve  = turnevents.TopicTurnObserve
	TopicTurnComplete = turnevents.TopicTurnComplete
	TopicTurnError    = turnevents.TopicTurnError
)

type TurnEventEmitter = turnevents.TurnEventEmitter

func NewTurnEventEmitter(bus events.EventBus) *TurnEventEmitter {
	return turnevents.NewTurnEventEmitter(bus)
}
