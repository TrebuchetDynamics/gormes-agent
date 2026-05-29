package kanban

import (
	"context"
	"fmt"
)

// BoardStats mirrors Hermes' kanban board summary: non-archived task counts,
// assignee/status breakdowns, and the age of the oldest ready task.
type BoardStats struct {
	ByStatus              map[string]int            `json:"by_status"`
	ByAssignee            map[string]map[string]int `json:"by_assignee"`
	OldestReadyAgeSeconds *int64                    `json:"oldest_ready_age_seconds"`
	Now                   int64                     `json:"now"`
}

func (s *Store) BoardStats(ctx context.Context) (BoardStats, error) {
	stats := BoardStats{
		ByStatus:   map[string]int{},
		ByAssignee: map[string]map[string]int{},
		Now:        s.now().UTC().Unix(),
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT status, COUNT(*) AS n
FROM tasks
WHERE status != ?
GROUP BY status`, string(StatusArchived))
	if err != nil {
		return BoardStats{}, fmt.Errorf("kanban board stats by status: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return BoardStats{}, fmt.Errorf("scan kanban board stats by status: %w", err)
		}
		stats.ByStatus[status] = count
	}
	if err := rows.Err(); err != nil {
		return BoardStats{}, fmt.Errorf("scan kanban board stats by status: %w", err)
	}

	assigneeRows, err := s.db.QueryContext(ctx, `
SELECT assignee, status, COUNT(*) AS n
FROM tasks
WHERE status != ? AND TRIM(assignee) != ''
GROUP BY assignee, status`, string(StatusArchived))
	if err != nil {
		return BoardStats{}, fmt.Errorf("kanban board stats by assignee: %w", err)
	}
	defer assigneeRows.Close()
	for assigneeRows.Next() {
		var assignee, status string
		var count int
		if err := assigneeRows.Scan(&assignee, &status, &count); err != nil {
			return BoardStats{}, fmt.Errorf("scan kanban board stats by assignee: %w", err)
		}
		if stats.ByAssignee[assignee] == nil {
			stats.ByAssignee[assignee] = map[string]int{}
		}
		stats.ByAssignee[assignee][status] = count
	}
	if err := assigneeRows.Err(); err != nil {
		return BoardStats{}, fmt.Errorf("scan kanban board stats by assignee: %w", err)
	}

	readyRows, err := s.db.QueryContext(ctx, `
SELECT created_at
FROM tasks
WHERE status = ?`, string(StatusReady))
	if err != nil {
		return BoardStats{}, fmt.Errorf("kanban board stats oldest ready: %w", err)
	}
	defer readyRows.Close()
	var oldestMillis int64
	for readyRows.Next() {
		var createdAt safeMillis
		if err := readyRows.Scan(&createdAt); err != nil {
			return BoardStats{}, fmt.Errorf("scan kanban board stats oldest ready: %w", err)
		}
		if createdAt.value == 0 {
			continue
		}
		if oldestMillis == 0 || createdAt.value < oldestMillis {
			oldestMillis = createdAt.value
		}
	}
	if err := readyRows.Err(); err != nil {
		return BoardStats{}, fmt.Errorf("scan kanban board stats oldest ready: %w", err)
	}
	if oldestMillis != 0 {
		age := stats.Now - millisToTime(oldestMillis).Unix()
		stats.OldestReadyAgeSeconds = &age
	}

	return stats, nil
}
