package indicator

import (
	"fmt"
	"strings"
)

const SlashUsage = "usage: /indicator [ascii|emoji|kaomoji|unicode]"

type SlashResult struct {
	Style  Style
	Status string
	Apply  bool
}

// ParseSlash resolves /indicator input into display evidence and an optional
// style mutation. Invalid invocations return SlashUsage and Apply=false.
func ParseSlash(input string, current Style) SlashResult {
	args := strings.Fields(strings.TrimSpace(input))
	if len(args) <= 1 {
		return SlashResult{Style: NormalizeStyle(string(current)), Status: fmt.Sprintf("indicator: %s", NormalizeStyle(string(current)))}
	}
	if len(args) > 2 {
		return SlashResult{Style: current, Status: SlashUsage}
	}
	style := Style(strings.ToLower(strings.TrimSpace(args[1])))
	if NormalizeStyle(string(style)) != style {
		return SlashResult{Style: current, Status: SlashUsage}
	}
	return SlashResult{Style: style, Status: fmt.Sprintf("indicator → %s", style), Apply: true}
}
