package hermes

import (
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/events"
)

const (
	TopicTurnStart    = "agent.turn.start"
	TopicTurnThought  = "agent.turn.thought"
	TopicTurnAction   = "agent.turn.action"
	TopicTurnObserve  = "agent.turn.observe"
	TopicTurnComplete = "agent.turn.complete"
	TopicTurnError    = "agent.turn.error"
)

type TurnEventEmitter struct {
	bus events.EventBus
}

func NewTurnEventEmitter(bus events.EventBus) *TurnEventEmitter {
	return &TurnEventEmitter{bus: bus}
}

func (e *TurnEventEmitter) EmitStart(source, traceID string) error {
	evt := events.NewEvent(TopicTurnStart, source, nil, traceID)
	return e.bus.Publish(TopicTurnStart, evt)
}

func (e *TurnEventEmitter) EmitThought(source, traceID string, thought string) error {
	raw, _ := json.Marshal(map[string]string{"thought": thought})
	evt := events.NewEvent(TopicTurnThought, source, raw, traceID)
	return e.bus.Publish(TopicTurnThought, evt)
}

func (e *TurnEventEmitter) EmitAction(source, traceID string, toolName string, args json.RawMessage) error {
	raw, _ := json.Marshal(map[string]interface{}{"tool": toolName, "args": string(args)})
	evt := events.NewEvent(TopicTurnAction, source, raw, traceID)
	return e.bus.Publish(TopicTurnAction, evt)
}

func (e *TurnEventEmitter) EmitObserve(source, traceID string, result json.RawMessage) error {
	evt := events.NewEvent(TopicTurnObserve, source, result, traceID)
	return e.bus.Publish(TopicTurnObserve, evt)
}

func (e *TurnEventEmitter) EmitComplete(source, traceID string) error {
	evt := events.NewEvent(TopicTurnComplete, source, nil, traceID)
	return e.bus.Publish(TopicTurnComplete, evt)
}

func (e *TurnEventEmitter) EmitError(source, traceID string, errMsg string) error {
	raw, _ := json.Marshal(map[string]string{"error": errMsg})
	evt := events.NewEvent(TopicTurnError, source, raw, traceID)
	return e.bus.Publish(TopicTurnError, evt)
}
