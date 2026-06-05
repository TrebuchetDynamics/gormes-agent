// Package contract holds shared mixture-of-agents tool identifiers and error
// formatting used by both the provider-backed implementation and the
// compatibility stub.
package contract

import (
	"errors"
	"fmt"
)

const ToolName = "mixture_of_agents"

func Error(message string) error {
	return errors.New(ToolName + ": " + message)
}

func Errorf(format string, args ...any) error {
	return fmt.Errorf(ToolName+": "+format, args...)
}
