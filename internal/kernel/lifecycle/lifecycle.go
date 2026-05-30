package lifecycle

import "context"

// Point identifies the lifecycle stage.
type Point string

const (
	Start Point = "agent:start"
	Step  Point = "agent:step"
	End   Point = "agent:end"
)

// Event is passed to Hook.
type Event struct {
	Point     Point
	SessionID string
	Iteration int
	ToolNames []string
	Err       error
}

// Hook is a callback for agent turn lifecycle events.
// Nil means no lifecycle events are emitted.
type Hook func(ctx context.Context, ev Event)
