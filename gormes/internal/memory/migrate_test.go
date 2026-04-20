package memory

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestOpenSqlite_FreshDBIsV3b(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer s.Close(context.Background())

	var v string
	_ = s.db.QueryRow("SELECT v FROM schema_meta WHERE k = 'version'").Scan(&v)
	if v != "3b" {
		t.Errorf("schema version = %q, want 3b", v)
	}
}

func TestMigrate_TurnsGainsExtractedColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)
	defer s.Close(context.Background())

	for _, col := range []string{"extracted", "extraction_attempts", "extraction_error"} {
		var name string
		row := s.db.QueryRow(
			`SELECT name FROM pragma_table_info('turns') WHERE name = ?`, col)
		if err := row.Scan(&name); err != nil {
			t.Errorf("column %q missing from turns: %v", col, err)
		}
	}
}

func TestMigrate_EntitiesAndRelationshipsExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)
	defer s.Close(context.Background())

	for _, table := range []string{"entities", "relationships"} {
		var n int
		err := s.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n)
		if err != nil {
			t.Errorf("table %q missing: %v", table, err)
		}
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.Close(context.Background())

	// Re-open — migration runs against v3b, should no-op.
	s2, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatalf("re-open failed: %v", err)
	}
	defer s2.Close(context.Background())

	var v string
	_ = s2.db.QueryRow("SELECT v FROM schema_meta WHERE k = 'version'").Scan(&v)
	if v != "3b" {
		t.Errorf("version = %q after re-open, want 3b", v)
	}
}

func TestMigrate_UnknownVersionRefuses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)
	_, _ = s.db.Exec(`UPDATE schema_meta SET v = '3z' WHERE k = 'version'`)
	s.Close(context.Background())

	_, err := OpenSqlite(path, 0, nil)
	if !errors.Is(err, ErrSchemaUnknown) {
		t.Errorf("err = %v, want errors.Is(err, ErrSchemaUnknown)", err)
	}
}
