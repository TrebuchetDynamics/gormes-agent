package navivox

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

var navivoxMemoryDBPath = config.MemoryDBPath

type memoryOverviewResponse struct {
	ProfileID      string         `json:"profile_id"`
	WorkspaceID    string         `json:"workspace_id"`
	DatabaseLabel  string         `json:"database_label"`
	Health         string         `json:"health"`
	DegradedReason string         `json:"degraded_reason,omitempty"`
	Counts         map[string]int `json:"counts"`
}

func (c *Channel) handleMemoryOverview(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	profileID := strings.TrimSpace(r.URL.Query().Get("profile_id"))
	if profileID == "" || profileID == "default" {
		profileID = "main"
	}
	overview := readMemoryOverview(r.Context(), navivoxMemoryDBPath(), profileID)
	writeNavivoxJSON(w, http.StatusOK, overview)
}

func readMemoryOverview(ctx context.Context, dbPath, profileID string) memoryOverviewResponse {
	overview := memoryOverviewResponse{
		ProfileID:     profileID,
		WorkspaceID:   "gormes",
		DatabaseLabel: safeMemoryDatabaseLabel(dbPath),
		Health:        "active",
		Counts: map[string]int{
			"turns":             0,
			"memory_items":      0,
			"observations":      0,
			"conclusions":       0,
			"session_summaries": 0,
			"entities":          0,
			"relationships":     0,
		},
	}
	trimmedPath := strings.TrimSpace(dbPath)
	if trimmedPath == "" {
		overview.Health = "degraded"
		overview.DegradedReason = "Gormes memory database path is not configured."
		return overview
	}
	db, err := sql.Open("sqlite3", trimmedPath)
	if err != nil {
		overview.Health = "degraded"
		overview.DegradedReason = "Gormes memory API is unavailable."
		return overview
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		overview.Health = "degraded"
		overview.DegradedReason = "Gormes memory API is unavailable."
		return overview
	}

	tables := []struct {
		key   string
		table string
		where string
	}{
		{key: "turns", table: "turns"},
		{key: "memory_items", table: "goncho_memory_items", where: "active = 1"},
		{key: "observations", table: "goncho_observations"},
		{key: "conclusions", table: "goncho_conclusions"},
		{key: "session_summaries", table: "goncho_session_summaries"},
		{key: "entities", table: "entities"},
		{key: "relationships", table: "relationships"},
	}
	for _, table := range tables {
		count, err := countMemoryTable(ctx, db, table.table, table.where)
		if err != nil {
			overview.Health = "degraded"
			overview.DegradedReason = "Gormes memory API is unavailable."
			return overview
		}
		overview.Counts[table.key] = count
	}
	if workspaceID, ok := firstMemoryWorkspaceID(ctx, db); ok {
		overview.WorkspaceID = workspaceID
	}
	return overview
}

func countMemoryTable(ctx context.Context, db *sql.DB, table, where string) (int, error) {
	exists, err := memoryTableExists(ctx, db, table)
	if err != nil || !exists {
		return 0, err
	}
	query := "SELECT COUNT(*) FROM " + table
	if where != "" {
		query += " WHERE " + where
	}
	var count int
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func memoryTableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var name string
	err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func firstMemoryWorkspaceID(ctx context.Context, db *sql.DB) (string, bool) {
	for _, table := range []string{"goncho_memory_items", "goncho_conclusions", "goncho_session_summaries"} {
		exists, err := memoryTableExists(ctx, db, table)
		if err != nil || !exists {
			continue
		}
		var workspaceID string
		err = db.QueryRowContext(ctx, "SELECT workspace_id FROM "+table+" WHERE trim(workspace_id) != '' ORDER BY rowid DESC LIMIT 1").Scan(&workspaceID)
		if err == nil && strings.TrimSpace(workspaceID) != "" {
			return strings.TrimSpace(workspaceID), true
		}
	}
	return "", false
}

func safeMemoryDatabaseLabel(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "redacted"
	}
	const marker = "/.gormes/"
	if i := strings.Index(path, marker); i >= 0 {
		return "~/.gormes/" + path[i+len(marker):]
	}
	if filepath.IsAbs(path) {
		return "redacted/" + filepath.Base(path)
	}
	return path
}
