package tui

import "strings"

// goalSlashHandler dispatches /goal to the injected session-goal controller.
// The controller owns persistence; the TUI only renders the reply.
func goalSlashHandler(input string, model *Model) SlashResult {
	if model == nil || model.goalCommand == nil {
		return SlashResult{Handled: true, StatusMessage: "goal: goal store unavailable"}
	}
	sessionID := strings.TrimSpace(model.SessionID())
	if sessionID == "" {
		return SlashResult{Handled: true, StatusMessage: "goal: session unavailable"}
	}
	if model.turnActive() && goalSlashSetsGoal(input) {
		return SlashResult{Handled: true, StatusMessage: "Agent is running — use /goal status / pause / clear mid-run, or /stop before setting a new goal."}
	}
	reply, err := model.goalCommand(sessionID, input)
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "goal: " + err.Error()}
	}
	if strings.TrimSpace(reply) == "" {
		reply = "goal: no reply"
	}
	model.refreshGoalLine()
	return SlashResult{Handled: true, StatusMessage: reply}
}

func goalSlashSetsGoal(input string) bool {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) < 2 {
		return false
	}
	switch strings.ToLower(fields[1]) {
	case "status", "pause", "resume", "clear", "stop", "done", "subgoal", "subgoals":
		return false
	default:
		return true
	}
}
