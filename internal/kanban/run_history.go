package kanban

import (
	"context"
	"fmt"
)

// RunHistory provides query access to task run records stored in the
// kanban store's task_runs table.
type RunHistory struct {
	store *Store
}

// NewRunHistory creates a RunHistory backed by the given store.
func NewRunHistory(store *Store) *RunHistory {
	return &RunHistory{store: store}
}

// ListTaskRuns returns all recorded runs for the given task, ordered
// by run ID (oldest first).
func (h *RunHistory) ListTaskRuns(ctx context.Context, taskID string) ([]TaskRun, error) {
	if h.store == nil {
		return nil, fmt.Errorf("kanban RunHistory store is nil")
	}
	return h.store.ListRuns(ctx, taskID)
}

// RecentRuns returns up to limit most recent runs for the given task,
// ordered by run ID (oldest first).
func (h *RunHistory) RecentRuns(ctx context.Context, taskID string, limit int) ([]TaskRun, error) {
	if limit <= 0 {
		return nil, nil
	}
	all, err := h.ListTaskRuns(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if len(all) <= limit {
		return all, nil
	}
	return all[len(all)-limit:], nil
}
