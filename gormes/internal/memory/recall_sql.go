package memory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// seedsExactName returns up to `limit` entity IDs whose name (lower-fold)
// matches any of the provided candidates. Silently drops short candidates
// (<3 chars) before sending to SQL. Empty candidates list returns
// (nil, nil) with no DB round-trip.
func seedsExactName(ctx context.Context, db *sql.DB, candidates []string, limit int) ([]int64, error) {
	// Pre-filter: drop empties and shorts, lower-fold for the IN-list.
	clean := make([]any, 0, len(candidates))
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if len(c) < 3 {
			continue
		}
		clean = append(clean, strings.ToLower(c))
	}
	if len(clean) == 0 {
		return nil, nil
	}

	placeholders := strings.Repeat("?,", len(clean))
	placeholders = placeholders[:len(placeholders)-1] // trim trailing comma
	args := append(clean, any(limit))
	q := fmt.Sprintf(
		`SELECT id FROM entities
		 WHERE lower(name) IN (%s)
		   AND length(name) >= 3
		 LIMIT ?`, placeholders)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("seedsExactName: %w", err)
	}
	defer rows.Close()
	return scanIDs(rows)
}

// seedsFTS5 is the Layer 2 fallback: FTS5 MATCH over turns.content, joined
// back to entities whose names appear in those turns. Per-chat scoped via
// the chat_id filter (empty string = global scope — matches any chat_id).
func seedsFTS5(ctx context.Context, db *sql.DB, userMessage, chatKey string, limit int) ([]int64, error) {
	msg := strings.TrimSpace(userMessage)
	if msg == "" {
		return nil, nil
	}

	q := `
		SELECT DISTINCT e.id
		FROM turns_fts fts
		JOIN turns t ON t.id = fts.rowid
		JOIN entities e ON lower(t.content) LIKE '%' || lower(e.name) || '%'
		WHERE turns_fts MATCH ?
		  AND (t.chat_id = ? OR ? = '')
		  AND length(e.name) >= 3
		LIMIT ?
	`
	rows, err := db.QueryContext(ctx, q, msg, chatKey, chatKey, limit)
	if err != nil {
		return nil, fmt.Errorf("seedsFTS5: %w", err)
	}
	defer rows.Close()
	return scanIDs(rows)
}

// scanIDs drains `rows` into a []int64 of ID columns.
func scanIDs(rows *sql.Rows) ([]int64, error) {
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// traverseNeighborhood runs the Recursive CTE that expands a set of seed
// entity IDs into a depth-bounded neighborhood, filtered by relationship
// weight >= threshold, sorted by depth ASC then updated_at DESC, capped
// at maxFacts.
//
// Depth 0 = seeds themselves.
// Depth N = reachable via N hops along edges with weight >= threshold.
func traverseNeighborhood(
	ctx context.Context,
	db *sql.DB,
	seedIDs []int64,
	depth int,
	threshold float64,
	maxFacts int,
) ([]recalledEntity, error) {
	if len(seedIDs) == 0 {
		return nil, nil
	}

	// Build the seeds VALUES() clause: (?), (?), ...
	seedValues := strings.Repeat("(?),", len(seedIDs))
	seedValues = seedValues[:len(seedValues)-1]
	args := make([]any, 0, len(seedIDs)+3)
	for _, id := range seedIDs {
		args = append(args, id)
	}
	args = append(args, threshold, depth, maxFacts)

	q := fmt.Sprintf(`
		WITH RECURSIVE
			seeds(entity_id) AS (VALUES %s),
			neighborhood(entity_id, depth) AS (
				SELECT entity_id, 0 FROM seeds
				UNION
				SELECT
					CASE WHEN r.source_id = n.entity_id THEN r.target_id
					     ELSE r.source_id END,
					n.depth + 1
				FROM neighborhood n
				JOIN relationships r
					ON (r.source_id = n.entity_id OR r.target_id = n.entity_id)
				   AND r.weight >= ?
				WHERE n.depth < ?
			),
			dedup_neighborhood AS (
				SELECT entity_id, MIN(depth) AS depth
				FROM neighborhood
				GROUP BY entity_id
			)
		SELECT e.name, e.type, COALESCE(e.description, '')
		FROM dedup_neighborhood dn
		JOIN entities e ON e.id = dn.entity_id
		ORDER BY dn.depth ASC, e.updated_at DESC
		LIMIT ?`, seedValues)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("traverseNeighborhood: %w", err)
	}
	defer rows.Close()

	var out []recalledEntity
	for rows.Next() {
		var e recalledEntity
		if err := rows.Scan(&e.Name, &e.Type, &e.Description); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
