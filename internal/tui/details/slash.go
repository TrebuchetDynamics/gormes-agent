package details

import "strings"

const SlashUsage = "usage: /details [hidden|collapsed|expanded|cycle]  or  /details <section> [hidden|collapsed|expanded|reset]"
const SectionSlashUsage = "usage: /details <section> [hidden|collapsed|expanded|reset]"

// ApplySlash resolves /details slash-command input against the current detail
// visibility state. It returns the next state and the status text that the root
// TUI should display; model availability and editor side effects stay in the
// root package.
func ApplySlash(input string, state State) (State, string) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) <= 1 {
		state = NormalizeState(state)
		return state, state.Status()
	}
	first := strings.ToLower(fields[1])
	if section, ok := ParseSection(first); ok {
		if len(fields) != 3 {
			return state, SectionSlashUsage
		}
		second := strings.ToLower(fields[2])
		if second == "reset" || second == "clear" || second == "default" {
			state = state.WithoutSection(section)
			return state, "details " + string(section) + ": reset"
		}
		mode, ok := ParseMode(second)
		if !ok {
			return state, SectionSlashUsage
		}
		state = state.WithSection(section, mode)
		return state, "details " + string(section) + ": " + string(mode)
	}
	if len(fields) != 2 {
		return state, SlashUsage
	}
	var next Mode
	switch first {
	case "cycle", "toggle":
		next = NextMode(NormalizeState(state).Global)
	default:
		mode, ok := ParseMode(first)
		if !ok {
			return state, SlashUsage
		}
		next = mode
	}
	state = state.WithGlobal(next)
	return state, "details: " + string(next)
}
