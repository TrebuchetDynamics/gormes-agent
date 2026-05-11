package kanban

import (
	"context"
	"errors"
	"fmt"
)

// ErrCrossBoardTask is returned by ValidateTaskBoard when a task does not
// belong to the queried board's store.
var ErrCrossBoardTask = errors.New("task does not belong to this board")

// ValidateTaskBoard returns nil if a task with the given ID exists in the
// store (i.e., belongs to that board). Returns ErrCrossBoardTask when the
// task is not found, enabling dispatchers to reject cross-board assignments.
func ValidateTaskBoard(ctx context.Context, store *Store, taskID string) error {
	_, err := store.GetTask(ctx, taskID)
	if err != nil {
		return ErrCrossBoardTask
	}
	return nil
}

// BoardDispatcher ties a kanban dispatch lifecycle to a specific board using
// per-board SQLite database files. Workers spawned through this dispatcher can
// only access the bound board's data because the store is opened on the
// board's dedicated DB file.
type BoardDispatcher struct {
	Dispatcher
	Board Board
}

// NewBoardDispatcher creates a board-scoped dispatcher. It opens a store on
// the board's database path and configures the underlying dispatcher to
// operate exclusively against that store. The caller must call Close on the
// returned dispatcher to release the store resources.
func NewBoardDispatcher(ctx context.Context, board Board, spawner WorkerSpawner) (*BoardDispatcher, error) {
	if board.Path == "" {
		return nil, errors.New("board path is required")
	}
	store, err := Open(ctx, board.Path)
	if err != nil {
		return nil, fmt.Errorf("open board %q store: %w", board.Name, err)
	}
	worker := board.Name
	if worker == "" {
		worker = "kanban-board"
	}
	return &BoardDispatcher{
		Dispatcher: Dispatcher{
			Store:   store,
			Spawner: spawner,
			Worker:  worker,
		},
		Board: board,
	}, nil
}

// Close releases the board's underlying store resources.
func (bd *BoardDispatcher) Close() error {
	if bd == nil || bd.Store == nil {
		return nil
	}
	return bd.Store.Close()
}
