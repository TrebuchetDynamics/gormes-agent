package session

import (
	"context"
	"errors"
	"strings"
	"time"
)

const DefaultGoalMaxTurns = 20

type GoalStatus string

const (
	GoalStatusActive  GoalStatus = "active"
	GoalStatusPaused  GoalStatus = "paused"
	GoalStatusDone    GoalStatus = "done"
	GoalStatusCleared GoalStatus = "cleared"
)

type GoalState struct {
	Goal         string     `json:"goal"`
	Status       GoalStatus `json:"status"`
	TurnsUsed    int        `json:"turns_used"`
	MaxTurns     int        `json:"max_turns"`
	CreatedAt    int64      `json:"created_at,omitempty"`
	LastTurnAt   int64      `json:"last_turn_at,omitempty"`
	LastVerdict  string     `json:"last_verdict,omitempty"`
	LastReason   string     `json:"last_reason,omitempty"`
	PausedReason string     `json:"paused_reason,omitempty"`
}

type GoalMetadataStore interface {
	GetMetadata(context.Context, string) (Metadata, bool, error)
	PutMetadata(context.Context, Metadata) error
}

func ContinuationPrompt(goal string) string {
	return "[Continuing toward your standing goal]\n" +
		"Goal: " + strings.TrimSpace(goal) + "\n\n" +
		"Continue working toward this goal. Take the next concrete step. " +
		"If you believe the goal is complete, state so explicitly and stop. " +
		"If you are blocked and need input from the user, say so clearly and stop."
}

func NormalizeGoalState(state GoalState) GoalState {
	state.Goal = strings.TrimSpace(state.Goal)
	state.Status = GoalStatus(strings.ToLower(strings.TrimSpace(string(state.Status))))
	switch state.Status {
	case GoalStatusActive, GoalStatusPaused, GoalStatusDone, GoalStatusCleared:
	default:
		state.Status = GoalStatusActive
	}
	if state.TurnsUsed < 0 {
		state.TurnsUsed = 0
	}
	if state.MaxTurns <= 0 {
		state.MaxTurns = DefaultGoalMaxTurns
	}
	state.LastVerdict = strings.ToLower(strings.TrimSpace(state.LastVerdict))
	state.LastReason = strings.TrimSpace(state.LastReason)
	state.PausedReason = strings.TrimSpace(state.PausedReason)
	return state
}

func CloneGoalState(state *GoalState) *GoalState {
	if state == nil {
		return nil
	}
	cloned := NormalizeGoalState(*state)
	return &cloned
}

func LoadGoal(ctx context.Context, store GoalMetadataStore, sessionID string) (*GoalState, bool, error) {
	if store == nil {
		return nil, false, nil
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, false, nil
	}
	meta, ok, err := store.GetMetadata(ctx, sessionID)
	if err != nil || !ok || meta.Goal == nil {
		return nil, ok && meta.Goal != nil, err
	}
	return CloneGoalState(meta.Goal), true, nil
}

func SaveGoal(ctx context.Context, store GoalMetadataStore, sessionID string, state *GoalState, now time.Time) error {
	if store == nil {
		return errors.New("session: goal metadata store unavailable")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session: goal session_id is required")
	}
	if state == nil {
		return errors.New("session: goal state is required")
	}
	normalized := NormalizeGoalState(*state)
	ts := now.Unix()
	if ts <= 0 {
		ts = time.Now().Unix()
	}
	return store.PutMetadata(ctx, Metadata{
		SessionID: sessionID,
		UpdatedAt: ts,
		Goal:      &normalized,
	})
}

func SetGoal(ctx context.Context, store GoalMetadataStore, sessionID, goal string, maxTurns int, now time.Time) (*GoalState, error) {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return nil, errors.New("session: goal cannot be empty")
	}
	ts := now.Unix()
	if ts <= 0 {
		ts = time.Now().Unix()
	}
	state := NormalizeGoalState(GoalState{
		Goal:      goal,
		Status:    GoalStatusActive,
		MaxTurns:  maxTurns,
		CreatedAt: ts,
	})
	if err := SaveGoal(ctx, store, sessionID, &state, now); err != nil {
		return nil, err
	}
	return &state, nil
}

func PauseGoal(ctx context.Context, store GoalMetadataStore, sessionID, reason string, now time.Time) (*GoalState, error) {
	state, ok, err := LoadGoal(ctx, store, sessionID)
	if err != nil || !ok {
		return nil, err
	}
	state.Status = GoalStatusPaused
	state.PausedReason = strings.TrimSpace(reason)
	if err := SaveGoal(ctx, store, sessionID, state, now); err != nil {
		return nil, err
	}
	return state, nil
}

func ResumeGoal(ctx context.Context, store GoalMetadataStore, sessionID string, resetBudget bool, now time.Time) (*GoalState, error) {
	state, ok, err := LoadGoal(ctx, store, sessionID)
	if err != nil || !ok {
		return nil, err
	}
	state.Status = GoalStatusActive
	state.PausedReason = ""
	if resetBudget {
		state.TurnsUsed = 0
	}
	if err := SaveGoal(ctx, store, sessionID, state, now); err != nil {
		return nil, err
	}
	return state, nil
}

func DoneGoal(ctx context.Context, store GoalMetadataStore, sessionID, reason string, now time.Time) (*GoalState, error) {
	state, ok, err := LoadGoal(ctx, store, sessionID)
	if err != nil || !ok {
		return nil, err
	}
	state.Status = GoalStatusDone
	state.LastVerdict = "done"
	state.LastReason = strings.TrimSpace(reason)
	if err := SaveGoal(ctx, store, sessionID, state, now); err != nil {
		return nil, err
	}
	return state, nil
}

func ClearGoal(ctx context.Context, store GoalMetadataStore, sessionID string, now time.Time) (*GoalState, error) {
	state, ok, err := LoadGoal(ctx, store, sessionID)
	if err != nil || !ok {
		return nil, err
	}
	state.Status = GoalStatusCleared
	state.PausedReason = ""
	if err := SaveGoal(ctx, store, sessionID, state, now); err != nil {
		return nil, err
	}
	return state, nil
}

func GoalIsActive(state *GoalState) bool {
	return state != nil && NormalizeGoalState(*state).Status == GoalStatusActive && strings.TrimSpace(state.Goal) != ""
}
