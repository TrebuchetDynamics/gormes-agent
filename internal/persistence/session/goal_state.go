package session

import (
	"context"
	"errors"
	"strings"
	"time"

	goalpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session/goal"
)

const DefaultGoalMaxTurns = goalpkg.DefaultMaxTurns

type GoalStatus = goalpkg.Status

const (
	GoalStatusActive  = goalpkg.StatusActive
	GoalStatusPaused  = goalpkg.StatusPaused
	GoalStatusDone    = goalpkg.StatusDone
	GoalStatusCleared = goalpkg.StatusCleared
)

type GoalState = goalpkg.State

type GoalMetadataStore interface {
	GetMetadata(context.Context, string) (Metadata, bool, error)
	PutMetadata(context.Context, Metadata) error
}

func ContinuationPrompt(goal string, subgoals []string) string {
	return goalpkg.ContinuationPrompt(goal, subgoals)
}

func NormalizeGoalState(state GoalState) GoalState {
	return goalpkg.NormalizeState(state)
}

func CloneGoalState(state *GoalState) *GoalState {
	return goalpkg.CloneState(state)
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
	return goalpkg.IsActive(state)
}
