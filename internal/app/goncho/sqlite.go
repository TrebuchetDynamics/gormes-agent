package goncho

import (
	"database/sql"
	"fmt"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

// sqlOpenGoncho opens a Goncho memory SQLite database at the given path,
// applying schema and migrations. It self-heals corrupt databases when
// possible.
func sqlOpenGoncho(path string) (*sql.DB, error) {
	db, err := sqlOpenGonchoRaw(path)
	if err == nil {
		return db, nil
	}
	if !memory.IsSQLiteCorruptionError(err) {
		return nil, err
	}
	if _, healErr := memory.SelfHealCorruptGonchoSQLite(path); healErr != nil {
		return nil, fmt.Errorf("%w; self-heal failed: %v", err, healErr)
	}
	return sqlOpenGonchoRaw(path)
}

// sqlOpenGonchoUnmigrated opens a Goncho memory SQLite database at the
// given path WITHOUT applying schema or migrations. Used for zero-byte
// DB inspection.
func sqlOpenGonchoUnmigrated(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// sqlOpenGonchoRaw opens a Goncho memory SQLite database at the given
// path, applying schema and migrations.
func sqlOpenGonchoRaw(path string) (*sql.DB, error) {
	db, err := sqlOpenGonchoUnmigrated(path)
	if err != nil {
		return nil, err
	}
	if err := memory.EnsureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := goncho.RunMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	for _, stmt := range []string{"PRAGMA journal_mode = WAL", "PRAGMA busy_timeout = 5000"} {
		if _, err := db.Exec(stmt); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("%s: %w", stmt, err)
		}
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
