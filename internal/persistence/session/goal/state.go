// Package goal owns the pure persistent goal-state value object for sessions.
//
// It exposes normalization, cloning, prompt formatting, and active-state checks.
// It must never know about Metadata stores, Bolt/SQL persistence, UI rendering,
// or agent orchestration loops.
package goal

import (
	"fmt"
	"strings"
)

const DefaultMaxTurns = 20

type Status string

const (
	StatusActive  Status = "active"
	StatusPaused  Status = "paused"
	StatusDone    Status = "done"
	StatusCleared Status = "cleared"
)

type State struct {
	Goal         string   `json:"goal"`
	Status       Status   `json:"status"`
	TurnsUsed    int      `json:"turns_used"`
	MaxTurns     int      `json:"max_turns"`
	CreatedAt    int64    `json:"created_at,omitempty"`
	LastTurnAt   int64    `json:"last_turn_at,omitempty"`
	LastVerdict  string   `json:"last_verdict,omitempty"`
	LastReason   string   `json:"last_reason,omitempty"`
	PausedReason string   `json:"paused_reason,omitempty"`
	Subgoals     []string `json:"subgoals,omitempty"`
}

func ContinuationPrompt(goal string, subgoals []string) string {
	prompt := "[Continuing toward your standing goal]\n" +
		"Goal: " + strings.TrimSpace(goal)
	if len(subgoals) > 0 {
		prompt += "\n\nSubgoals:\n"
		for i, sg := range subgoals {
			prompt += fmt.Sprintf("%d. %s\n", i+1, sg)
		}
	}
	prompt += "\n\n"
	prompt += "Continue working toward this goal. Take the next concrete step. "
	prompt += "If you believe the goal is complete, state so explicitly and stop. "
	prompt += "If you are blocked and need input from the user, say so clearly and stop."
	return prompt
}

func NormalizeState(state State) State {
	state.Goal = strings.TrimSpace(state.Goal)
	state.Status = Status(strings.ToLower(strings.TrimSpace(string(state.Status))))
	switch state.Status {
	case StatusActive, StatusPaused, StatusDone, StatusCleared:
	default:
		state.Status = StatusActive
	}
	if state.TurnsUsed < 0 {
		state.TurnsUsed = 0
	}
	if state.MaxTurns <= 0 {
		state.MaxTurns = DefaultMaxTurns
	}
	state.LastVerdict = strings.ToLower(strings.TrimSpace(state.LastVerdict))
	state.LastReason = strings.TrimSpace(state.LastReason)
	state.PausedReason = strings.TrimSpace(state.PausedReason)
	return state
}

func CloneState(state *State) *State {
	if state == nil {
		return nil
	}
	cloned := NormalizeState(*state)
	return &cloned
}

func IsActive(state *State) bool {
	return state != nil && NormalizeState(*state).Status == StatusActive && strings.TrimSpace(state.Goal) != ""
}
