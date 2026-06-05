package memory

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// IsSQLiteCorruptionError recognizes SQLite corruption/open errors that can be
// safely recovered by quarantining the state file and recreating an empty DB.
func IsSQLiteCorruptionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "file is not a database") ||
		strings.Contains(msg, "database disk image is malformed") ||
		strings.Contains(msg, "file is encrypted or is not a database")
}

// SelfHealCorruptGonchoSQLite quarantines a corrupt Goncho/memory SQLite file
// and recreates it with the local memory schema.
func SelfHealCorruptGonchoSQLite(path string) (string, error) {
	backup, err := QuarantineCorruptStateFile(path, []string{"-wal", "-shm"})
	if err != nil {
		return "", err
	}
	store, err := OpenSqlite(path, 0, nil)
	if err != nil {
		return backup, fmt.Errorf("recreate memory db: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := store.Close(ctx); err != nil {
		return backup, fmt.Errorf("close recreated memory db: %w", err)
	}
	return backup, nil
}

// QuarantineCorruptStateFile renames a corrupt state file and optional sidecars
// to unique .corrupt-* backup paths.
func QuarantineCorruptStateFile(path string, sidecarSuffixes []string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("empty state path")
	}
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	backup := UniqueCorruptBackupPath(path)
	if err := os.Rename(path, backup); err != nil {
		return "", fmt.Errorf("quarantine corrupt state file: %w", err)
	}
	for _, suffix := range sidecarSuffixes {
		sidecar := path + suffix
		if _, err := os.Stat(sidecar); err != nil {
			continue
		}
		_ = os.Rename(sidecar, backup+suffix)
	}
	return backup, nil
}

// UniqueCorruptBackupPath returns an unused quarantine backup path for path.
func UniqueCorruptBackupPath(path string) string {
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	base := path + ".corrupt-" + stamp
	candidate := base
	for i := 1; ; i++ {
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}
