package skin

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin/command"

type Request = command.Request
type Result = command.Result
type ConfigFunc = command.ConfigFunc
type SlashResult = command.SlashResult

func HandleSlash(input string, sessionID string, configure ConfigFunc) SlashResult {
	return command.HandleSlash(input, sessionID, configure)
}

func SlashName(input string) string { return command.SlashName(input) }

func DisplayName(name string) string { return command.DisplayName(name) }
