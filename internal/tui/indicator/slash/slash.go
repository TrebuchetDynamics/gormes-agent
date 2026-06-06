package slash

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/indicator/style"
)

const Usage = "usage: /indicator [ascii|emoji|kaomoji|unicode]"

type Result struct {
	Style  style.Style
	Status string
	Apply  bool
}

// Parse resolves /indicator input into display evidence and an optional
// style mutation. Invalid invocations return Usage and Apply=false.
func Parse(input string, current style.Style) Result {
	args := strings.Fields(strings.TrimSpace(input))
	if len(args) <= 1 {
		current = style.Normalize(string(current))
		return Result{Style: current, Status: fmt.Sprintf("indicator: %s", current)}
	}
	if len(args) > 2 {
		return Result{Style: current, Status: Usage}
	}
	parsed := style.Style(strings.ToLower(strings.TrimSpace(args[1])))
	if style.Normalize(string(parsed)) != parsed {
		return Result{Style: current, Status: Usage}
	}
	return Result{Style: parsed, Status: fmt.Sprintf("indicator → %s", parsed), Apply: true}
}
