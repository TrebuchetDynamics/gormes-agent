package tuiapp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func seedTUISaveTranscriptDB(t *testing.T, sessionID, chatID string) {
	t.Helper()

	// Defensive guard against the test-pollution bug where a caller forgets to
	// t.Setenv GORMES_HOME first and the seeder writes fixtures into the
	// operator's real ~/.gormes/memory.db.
	mdb := config.MemoryDBPath()
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		realHome := filepath.Join(home, ".gormes", "memory.db")
		if mdb == realHome {
			t.Fatalf("seedTUISaveTranscriptDB refuses to write fixtures to the operator's real memory.db at %s; t.Setenv GORMES_HOME to a t.TempDir() first", mdb)
		}
	}

	store, err := memory.OpenSqlite(mdb, 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	meta := `{"tool_calls":[{"id":"call_1","name":"list_dir","arguments":{"path":"."}}]}`
	if _, err := store.DB().Exec(
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, meta_json)
		 VALUES (?, 'user', 'hello from tui', ?, ?, NULL),
		        (?, 'assistant', 'saved from transcript store', ?, ?, ?)`,
		sessionID, time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC).Unix(), chatID,
		sessionID, time.Date(2026, 4, 22, 10, 0, 4, 0, time.UTC).Unix(), chatID, meta,
	); err != nil {
		t.Fatalf("seed turns: %v", err)
	}
}
