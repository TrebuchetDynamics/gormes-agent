// Package parser parses pure gateway /reasoning command arguments.
package parser

import (
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/reasoning/model"
)

// Parse turns the raw split arguments of /reasoning into a typed
// ReasoningCommand. It is pure: no I/O, no clock, no state.
func Parse(args []string) (model.ReasoningCommand, error) {
	global := false
	tokens := make([]string, 0, len(args))
	for _, raw := range args {
		if raw == "--global" {
			global = true
			continue
		}
		tokens = append(tokens, raw)
	}

	if len(tokens) == 0 && !global {
		return model.ReasoningCommand{Action: model.ReasoningActionShow}, nil
	}

	if len(tokens) == 0 {
		return model.ReasoningCommand{}, fmt.Errorf("%w: missing argument", model.ErrInvalidEffort)
	}

	if len(tokens) > 1 {
		return model.ReasoningCommand{}, fmt.Errorf("%w: %q", model.ErrInvalidEffort, tokens)
	}

	switch tokens[0] {
	case "reset":
		if global {
			return model.ReasoningCommand{}, model.ErrResetGlobalUnsupported
		}
		return model.ReasoningCommand{Action: model.ReasoningActionReset}, nil
	case string(model.ReasoningEffortHigh), string(model.ReasoningEffortLow), string(model.ReasoningEffortMedium):
		return model.ReasoningCommand{
			Action: model.ReasoningActionSet,
			Effort: model.ReasoningEffort(tokens[0]),
			Global: global,
		}, nil
	default:
		return model.ReasoningCommand{}, fmt.Errorf("%w: %q", model.ErrInvalidEffort, tokens[0])
	}
}
